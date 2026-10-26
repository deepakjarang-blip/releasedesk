package release

import (
	"database/sql"
	"errors"
	"github.com/brewbits-co/releasedesk/internal/domains/release"
	"github.com/brewbits-co/releasedesk/internal/values"
	"github.com/brewbits-co/releasedesk/internal/views"
	"github.com/brewbits-co/releasedesk/pkg/session"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"strconv"
)

func (c *releaseController) RenderReleaseList(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	currentApp, err := c.appService.GetCurrentAppData(values.Slug(slug))
	if err != nil {
		log.Printf("release list app lookup failed for %q: %v", slug, err)
		http.Error(w, "Unable to load application", http.StatusInternalServerError)
		return
	}
	if currentApp.AppID == 0 {
		http.NotFound(w, r)
		return
	}

	channels, err := c.service.GetReleaseChannels(currentApp.AppID)
	if err != nil {
		log.Printf("release list channels lookup failed for %q: %v", slug, err)
		http.Error(w, "Unable to load release channels", http.StatusInternalServerError)
		return
	}

	var currentChannelID int
	if len(channels) > 0 {
		requestedChannelID, _ := strconv.Atoi(r.URL.Query().Get("channel"))
		currentChannelID = channels[0].ID
		for _, channel := range channels {
			if channel.ID == requestedChannelID {
				currentChannelID = requestedChannelID
				break
			}
		}
	}

	var releases []release.BasicInfo
	if currentChannelID != 0 {
		releases, err = c.service.ListReleasesByChannel(currentApp.AppID, currentChannelID)
		if err != nil {
			log.Printf("release list lookup failed for %q: %v", slug, err)
			http.Error(w, "Unable to load releases", http.StatusInternalServerError)
			return
		}
	}

	data := ReleaseListData{
		CurrentPage:      "Releases",
		SessionData:      session.NewSessionData(r.Context()),
		CurrentAppData:   currentApp,
		Releases:         releases,
		Channels:         channels,
		CurrentChannelID: currentChannelID,
	}

	tmpl, err := views.ParseTemplate(views.SidebarLayout, "templates/console/release_list.gohtml")
	if err != nil {
		log.Printf("release list template parse failed: %v", err)
		http.Error(w, "Unable to render releases", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "index.gohtml", data); err != nil {
		log.Printf("release list render failed: %v", err)
	}

}

func (c *releaseController) RenderReleaseSummary(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	currentApp, err := c.appService.GetCurrentAppData(values.Slug(slug))
	if err != nil {
		log.Printf("release summary app lookup failed for %q: %v", slug, err)
		http.Error(w, "Unable to load application", http.StatusInternalServerError)
		return
	}
	if currentApp.AppID == 0 {
		http.NotFound(w, r)
		return
	}

	releaseSummary, err := c.service.GetReleaseSummary(currentApp.AppID, chi.URLParam(r, "version"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
		} else {
			log.Printf("release summary lookup failed for %q: %v", slug, err)
			http.Error(w, "Unable to load release", http.StatusInternalServerError)
		}
		return
	}

	c.renderReleaseSummary(w, r, currentApp, releaseSummary)
}

func (c *releaseController) RenderReleaseSummaryByID(w http.ResponseWriter, r *http.Request) {
	currentApp, ok := c.getCurrentApp(w, r)
	if !ok {
		return
	}

	releaseSummary, ok := c.getReleaseByID(w, r, currentApp.AppID)
	if !ok {
		return
	}
	c.renderReleaseSummary(w, r, currentApp, releaseSummary)
}

func (c *releaseController) renderReleaseSummary(w http.ResponseWriter, r *http.Request, currentApp session.CurrentAppData, releaseSummary release.Release) {
	channels, err := c.service.GetReleaseChannels(currentApp.AppID)
	if err != nil {
		log.Printf("release channels lookup failed for app %q: %v", currentApp.AppSlug, err)
		http.Error(w, "Unable to load release channels", http.StatusInternalServerError)
		return
	}

	tmpl, err := views.ParseTemplate(views.SidebarLayout, "templates/console/release_summary.gohtml")
	if err != nil {
		log.Printf("release summary template parse failed: %v", err)
		http.Error(w, "Unable to render release", http.StatusInternalServerError)
		return
	}

	buildSelections := make([]BuildSelectionOption, 0, len(currentApp.Platforms))
	for _, platform := range currentApp.Platforms {
		builds, err := c.buildService.GetPlatformBuilds(platform.PlatformID)
		if err != nil {
			http.Error(w, "Unable to load platform builds", http.StatusInternalServerError)
			return
		}
		selection := BuildSelectionOption{OS: platform.OS, Builds: builds}
		if selected, ok := releaseSummary.Builds[platform.OS]; ok {
			selection.SelectedBuildID = selected.ID
		}
		buildSelections = append(buildSelections, selection)
	}

	data := ReleaseSummaryData{
		CurrentPage:     "Releases",
		SessionData:     session.NewSessionData(r.Context()),
		CurrentAppData:  currentApp,
		Channels:        channels,
		Release:         releaseSummary,
		BuildSelections: buildSelections,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "index.gohtml", data); err != nil {
		log.Printf("release summary render failed: %v", err)
	}
}

func (c *releaseController) RenderReleaseNotes(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	currentApp, err := c.appService.GetCurrentAppData(values.Slug(slug))
	if err != nil {
		log.Printf("release notes app lookup failed for %q: %v", slug, err)
		http.Error(w, "Unable to load application", http.StatusInternalServerError)
		return
	}
	if currentApp.AppID == 0 {
		http.NotFound(w, r)
		return
	}

	releaseSummary, err := c.service.GetReleaseSummary(currentApp.AppID, chi.URLParam(r, "version"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
		} else {
			log.Printf("release notes release lookup failed for %q: %v", slug, err)
			http.Error(w, "Unable to load release", http.StatusInternalServerError)
		}
		return
	}
	c.renderReleaseNotes(w, r, currentApp, releaseSummary)
}

func (c *releaseController) RenderReleaseNotesByID(w http.ResponseWriter, r *http.Request) {
	currentApp, ok := c.getCurrentApp(w, r)
	if !ok {
		return
	}
	releaseSummary, ok := c.getReleaseByID(w, r, currentApp.AppID)
	if !ok {
		return
	}
	c.renderReleaseNotes(w, r, currentApp, releaseSummary)
}

func (c *releaseController) renderReleaseNotes(w http.ResponseWriter, r *http.Request, currentApp session.CurrentAppData, releaseSummary release.Release) {
	releaseNotes, err := c.service.GetReleaseNotes(releaseSummary.ID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("release notes lookup failed for release %d: %v", releaseSummary.ID, err)
			http.Error(w, "Unable to load release notes", http.StatusInternalServerError)
			return
		}
		releaseNotes = release.NewReleaseNotes(releaseSummary.ID, "")
	}

	tmpl, err := views.ParseTemplate(views.SidebarLayout, "templates/console/release_notes.gohtml")
	if err != nil {
		log.Printf("release notes template parse failed: %v", err)
		http.Error(w, "Unable to render release notes", http.StatusInternalServerError)
		return
	}

	data := ReleaseNotesData{
		CurrentPage:    "Releases",
		SessionData:    session.NewSessionData(r.Context()),
		CurrentAppData: currentApp,
		Release:        releaseSummary,
		ReleaseNotes:   releaseNotes,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "index.gohtml", data); err != nil {
		log.Printf("release notes render failed: %v", err)
	}
}

func (c *releaseController) getCurrentApp(w http.ResponseWriter, r *http.Request) (session.CurrentAppData, bool) {
	slug := chi.URLParam(r, "slug")
	currentApp, err := c.appService.GetCurrentAppData(values.Slug(slug))
	if err != nil {
		log.Printf("application lookup failed for %q: %v", slug, err)
		http.Error(w, "Unable to load application", http.StatusInternalServerError)
		return session.CurrentAppData{}, false
	}
	if currentApp.AppID == 0 {
		http.NotFound(w, r)
		return session.CurrentAppData{}, false
	}
	return currentApp, true
}

func (c *releaseController) getReleaseByID(w http.ResponseWriter, r *http.Request, appID int) (release.Release, bool) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return release.Release{}, false
	}
	releaseRecord, err := c.service.GetReleaseByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
		} else {
			log.Printf("release lookup failed for id %d: %v", id, err)
			http.Error(w, "Unable to load release", http.StatusInternalServerError)
		}
		return release.Release{}, false
	}
	if releaseRecord.AppID != appID {
		http.NotFound(w, r)
		return release.Release{}, false
	}
	return releaseRecord, true
}
