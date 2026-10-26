package release

import (
	"github.com/brewbits-co/releasedesk/internal/domains/release"
	"github.com/brewbits-co/releasedesk/internal/values"
	"github.com/brewbits-co/releasedesk/pkg/schemas"
	"github.com/brewbits-co/releasedesk/pkg/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"net/http"
	"strconv"
	"strings"
)

// HandleUpdateBasicInfo handles the update of a release's BasicInfo
// Only the TargetChannel and Status fields can be updated
func (c *releaseController) HandleUpdateBasicInfo(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.PlainText(w, r, err.Error())
		return
	}

	// Extract the release ID from the URL
	releaseID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, schemas.NewErrorResponse("Invalid release ID", []string{err.Error()}))
		return
	}

	var updateRequest release.BasicInfo
	updateRequest.ID = releaseID

	err = utils.NewDecoder().Decode(&updateRequest, r.Form)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.PlainText(w, r, err.Error())
		return
	}
	selections := make(map[values.OS]int)
	if updateRequest.BuildSelection == release.Manual {
		for field, valuesForField := range r.Form {
			if !strings.HasPrefix(field, "Build_") || len(valuesForField) == 0 || valuesForField[0] == "" {
				continue
			}
			buildID, err := strconv.Atoi(valuesForField[0])
			if err != nil || buildID <= 0 {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, schemas.NewErrorResponse("Invalid build selection", []string{field}))
				return
			}
			selections[values.OS(strings.TrimPrefix(field, "Build_"))] = buildID
		}
	}

	_, err = c.service.UpdateReleaseBasicInfo(updateRequest)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, schemas.NewErrorResponse("Failed to update release", []string{err.Error()}))
		return
	}
	if err := c.service.UpdateReleaseBuilds(releaseID, selections); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, schemas.NewErrorResponse("Failed to update release builds", []string{err.Error()}))
		return
	}

	// Return a 200 OK status
	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]interface{}{
		"success": true,
	})
}
