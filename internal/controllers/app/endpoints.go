package app

import (
	"github.com/brewbits-co/releasedesk/internal/domains/app"
	"github.com/brewbits-co/releasedesk/internal/values"
	"github.com/brewbits-co/releasedesk/pkg/schemas"
	"github.com/brewbits-co/releasedesk/pkg/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"log"
	"net/http"
)

func (c *appController) HandleCreateApp(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.PlainText(w, r, err.Error())
		return
	}

	var appCreationRequest app.BasicInfo

	err = utils.NewDecoder().Decode(&appCreationRequest, r.Form)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.PlainText(w, r, err.Error())
		return
	}

	newApp, err := c.service.CreateApp(appCreationRequest)
	if err != nil {
		render.Status(r, http.StatusConflict)
		render.JSON(w, r, schemas.NewErrorResponse("An application with the same identifier already exists.", []string{
			"Please use a unique application name and slug.",
			"Check the existing applications before creating a new one.",
		}))
		return
	}

	log.Println(newApp.ID)

	render.Status(r, http.StatusCreated)
}

func (c *appController) HandleAppSetupGuide(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.PlainText(w, r, err.Error())
		return
	}
	slug := chi.URLParam(r, "slug")

	err = c.service.ApplyAppSetupGuide(
		values.Slug(slug),
		values.VersionFormat(r.Form.Get("VersionFormat")),
		app.SetupChannelsOption(r.Form.Get("Channels")),
		r.Form["CustomChannels"],
	)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, schemas.NewErrorResponse(err.Error(), nil))
		return
	}
}

func (c *appController) HandleUpdatePublisher(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid publisher details", http.StatusBadRequest)
		return
	}
	slug := values.Slug(chi.URLParam(r, "slug"))
	if err := c.service.UpdateAppPublisher(slug, r.Form.Get("Publisher")); err != nil {
		http.Error(w, "Unable to update publisher details", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/dashboard/"+string(slug), http.StatusSeeOther)
}
