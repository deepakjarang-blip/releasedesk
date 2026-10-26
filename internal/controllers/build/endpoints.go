package build

import (
	"encoding/json"
	"errors"
	"fmt"
	domainbuild "github.com/brewbits-co/releasedesk/internal/domains/build"
	buildservice "github.com/brewbits-co/releasedesk/internal/services/build"
	"github.com/brewbits-co/releasedesk/internal/values"
	"github.com/brewbits-co/releasedesk/pkg/fields"
	"github.com/brewbits-co/releasedesk/pkg/storage"
	"github.com/brewbits-co/releasedesk/pkg/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const uploadChunkSize = 1 << 20

var uploadIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

type chunkedBuildUpload struct {
	Number   string                       `json:"number"`
	Version  string                       `json:"version"`
	Metadata map[string]string            `json:"metadata"`
	Files    map[string]chunkedUploadFile `json:"files"`
}

type chunkedUploadFile struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Chunks   int    `json:"chunks"`
}

func (c *buildController) HandleBuildUploadChunk(w http.ResponseWriter, r *http.Request) {
	uploadID := chi.URLParam(r, "uploadID")
	field := chi.URLParam(r, "field")
	chunkIndex, err := strconv.Atoi(chi.URLParam(r, "index"))
	if !uploadIDPattern.MatchString(uploadID) || !isUploadField(field) || err != nil || chunkIndex < 0 {
		http.Error(w, "Invalid upload chunk", http.StatusBadRequest)
		return
	}

	uploadDir, err := getUploadDirectory(uploadID)
	if err != nil {
		http.Error(w, "Unable to prepare upload", http.StatusInternalServerError)
		return
	}

	chunkPath := filepath.Join(uploadDir, fmt.Sprintf("%s-%08d.part", field, chunkIndex))
	chunkFile, err := os.OpenFile(chunkPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			http.Error(w, "Upload chunk already received", http.StatusConflict)
			return
		}
		http.Error(w, "Unable to store upload chunk", http.StatusInternalServerError)
		return
	}

	limitedBody := http.MaxBytesReader(w, r.Body, uploadChunkSize)
	chunkBytes, copyErr := io.Copy(chunkFile, limitedBody)
	closeErr := chunkFile.Close()
	if copyErr != nil || closeErr != nil || chunkBytes == 0 {
		os.Remove(chunkPath)
		if copyErr != nil {
			var maxBytesError *http.MaxBytesError
			if errors.As(copyErr, &maxBytesError) {
				http.Error(w, "Upload chunk exceeds 1 MiB", http.StatusRequestEntityTooLarge)
				return
			}
		}
		http.Error(w, "Unable to store upload chunk", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (c *buildController) HandleBuildUploadComplete(w http.ResponseWriter, r *http.Request) {
	uploadID := chi.URLParam(r, "uploadID")
	if !uploadIDPattern.MatchString(uploadID) {
		http.Error(w, "Invalid upload ID", http.StatusBadRequest)
		return
	}

	uploadDir, err := getUploadDirectory(uploadID)
	if err != nil {
		http.Error(w, "Unable to prepare upload", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(uploadDir)

	var request chunkedBuildUpload
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "Invalid upload details", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(request.Number) == "" || strings.TrimSpace(request.Version) == "" || len(request.Files) == 0 {
		http.Error(w, "Build number, version, and at least one artifact are required", http.StatusBadRequest)
		return
	}

	files := make(map[values.Architecture]buildservice.FileUpload, len(request.Files))
	for field, description := range request.Files {
		architecture, ok := uploadFieldArchitecture(field)
		if !ok || description.Filename == "" || description.Size <= 0 || description.Chunks <= 0 || int64(description.Chunks) != (description.Size+uploadChunkSize-1)/uploadChunkSize {
			http.Error(w, "Invalid artifact details", http.StatusBadRequest)
			return
		}

		assembled, err := os.CreateTemp(uploadDir, "assembled-")
		if err != nil {
			http.Error(w, "Unable to assemble artifact", http.StatusInternalServerError)
			return
		}
		assembledPath := assembled.Name()
		var assembledSize int64
		for index := 0; index < description.Chunks; index++ {
			chunkPath := filepath.Join(uploadDir, fmt.Sprintf("%s-%08d.part", field, index))
			chunk, err := os.Open(chunkPath)
			if err != nil {
				assembled.Close()
				os.Remove(assembledPath)
				http.Error(w, "Artifact upload is incomplete", http.StatusBadRequest)
				return
			}
			chunkSize, copyErr := io.Copy(assembled, chunk)
			chunk.Close()
			if copyErr != nil || chunkSize == 0 || chunkSize > uploadChunkSize || (index < description.Chunks-1 && chunkSize != uploadChunkSize) {
				assembled.Close()
				os.Remove(assembledPath)
				http.Error(w, "Invalid artifact chunk", http.StatusBadRequest)
				return
			}
			assembledSize += chunkSize
		}
		if err := assembled.Close(); err != nil || assembledSize != description.Size {
			os.Remove(assembledPath)
			http.Error(w, "Artifact size does not match upload details", http.StatusBadRequest)
			return
		}

		files[architecture] = buildservice.FileUpload{
			Filename: description.Filename,
			Size:     assembledSize,
			Open: func() (multipart.File, error) {
				return os.Open(assembledPath)
			},
		}
	}

	_, err = c.service.UploadBuild(
		values.Slug(chi.URLParam(r, "slug")),
		values.OS(chi.URLParam(r, "platform")),
		domainbuild.BasicInfo{Number: request.Number, Version: request.Version},
		files,
		request.Metadata,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func getUploadDirectory(uploadID string) (string, error) {
	root := filepath.Join(os.TempDir(), "releasedesk-build-uploads")
	if err := os.MkdirAll(root, 0700); err != nil {
		return "", err
	}
	if err := os.Chmod(root, 0700); err != nil {
		return "", err
	}
	directory := filepath.Join(root, uploadID)
	if err := os.Mkdir(directory, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return "", err
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() {
		return "", errors.New("invalid upload directory")
	}
	return directory, nil
}

func isUploadField(field string) bool {
	_, ok := uploadFieldArchitecture(field)
	return ok
}

func uploadFieldArchitecture(field string) (values.Architecture, bool) {
	architectures := map[string]values.Architecture{
		"File":       values.NoArch,
		"File_x86":   values.X86,
		"File_x64":   values.X64,
		"File_ARM64": values.ARM64,
		"File_ARM":   values.ARM,
	}
	architecture, ok := architectures[field]
	return architecture, ok
}

func (c *buildController) HandleBuildUpload(w http.ResponseWriter, r *http.Request) {
	// Parse the incoming multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Invalid multipart upload", http.StatusBadRequest)
		return
	}
	defer r.MultipartForm.RemoveAll()

	slug := chi.URLParam(r, "slug")
	platform := chi.URLParam(r, "platform")

	var buildUploadRequest = struct {
		domainbuild.BasicInfo
		fields.Extendable
	}{}

	err := utils.NewDecoder().Decode(&buildUploadRequest, r.Form)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.PlainText(w, r, err.Error())
		return
	}

	// Allow multiple file upload for certain platforms than need different architectures (ex: Windows)
	// The field name should by default File and for specify an architecture File_x86, File_x64, etc.
	var files = make(map[values.Architecture]buildservice.FileUpload)

	if fhs := r.MultipartForm.File["File"]; len(fhs) > 0 {
		files[values.NoArch] = fileUploadFromHeader(fhs[0])
	}
	if fhs := r.MultipartForm.File["File_x86"]; len(fhs) > 0 {
		files[values.X86] = fileUploadFromHeader(fhs[0])
	}
	if fhs := r.MultipartForm.File["File_x64"]; len(fhs) > 0 {
		files[values.X64] = fileUploadFromHeader(fhs[0])
	}
	if fhs := r.MultipartForm.File["File_ARM64"]; len(fhs) > 0 {
		files[values.ARM64] = fileUploadFromHeader(fhs[0])
	}
	if fhs := r.MultipartForm.File["File_ARM"]; len(fhs) > 0 {
		files[values.ARM] = fileUploadFromHeader(fhs[0])
	}

	if len(files) == 0 {
		render.Status(r, http.StatusBadRequest)
		return
	}

	_, err = c.service.UploadBuild(
		values.Slug(slug),
		values.OS(platform),
		buildUploadRequest.BasicInfo,
		files,
		buildUploadRequest.Metadata,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	render.Status(r, http.StatusCreated)
}

func fileUploadFromHeader(header *multipart.FileHeader) buildservice.FileUpload {
	return buildservice.FileUpload{
		Filename: header.Filename,
		Size:     header.Size,
		Open: func() (multipart.File, error) {
			return header.Open()
		},
	}
}

func (c *buildController) HandleArtifactDownload(w http.ResponseWriter, r *http.Request) {
	checksum := chi.URLParam(r, "checksum")
	if len(checksum) < 2 {
		http.Error(w, "Invalid checksum", http.StatusBadRequest)
		return
	}

	// Determine file path based on checksum
	subFolder := checksum[:2]
	filePath := filepath.Join(storage.StorageDir, subFolder, checksum)

	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	// Serve the file as a download
	w.Header().Set("Content-Disposition", `attachment; filename="AnyDesk.exe"`)
	w.Header().Set("Content-Type", "application/octet-stream")
	if _, err := io.Copy(w, file); err != nil {
		http.Error(w, "Error serving file", http.StatusInternalServerError)
		return
	}
}
