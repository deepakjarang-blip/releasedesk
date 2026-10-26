package portal

import (
	"encoding/hex"
	"github.com/brewbits-co/releasedesk/pkg/storage"
	"github.com/go-chi/chi/v5"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (c *controller) HandleArtifactDownload(w http.ResponseWriter, r *http.Request) {
	checksum := chi.URLParam(r, "checksum")
	if len(checksum) != 64 {
		http.Error(w, "Invalid artifact checksum", http.StatusBadRequest)
		return
	}
	if _, err := hex.DecodeString(checksum); err != nil {
		http.Error(w, "Invalid artifact checksum", http.StatusBadRequest)
		return
	}
	if _, err := c.builds.GetPublicArtifactByChecksum(checksum); err != nil {
		http.NotFound(w, r)
		return
	}

	filePath := storage.ConvertChecksumToPath(checksum)
	file, err := os.Open(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		http.Error(w, "Unable to read artifact", http.StatusInternalServerError)
		return
	}

	filename := filepath.Base(r.URL.Query().Get("filename"))
	if filename == "." || filename == string(filepath.Separator) || filename == "" {
		filename = checksum
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	w.Header().Set("Content-Type", "application/octet-stream")
	if strings.TrimSpace(filename) == "" {
		http.Error(w, "Invalid artifact filename", http.StatusBadRequest)
		return
	}
	http.ServeContent(w, r, filename, info.ModTime(), file)
}
