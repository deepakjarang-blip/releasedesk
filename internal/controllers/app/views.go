package app

import (
	"github.com/brewbits-co/releasedesk/internal/values"
	"github.com/brewbits-co/releasedesk/internal/views"
	"github.com/brewbits-co/releasedesk/pkg/session"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
)

func (c *appController) RenderDashboard(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	currentApp, err := c.service.GetCurrentAppData(values.Slug(slug))
	if err != nil {
		log.Printf("application dashboard lookup failed for %q: %v", slug, err)
		http.Error(w, "Unable to load application", http.StatusInternalServerError)
		return
	}
	if currentApp.AppID == 0 {
		http.NotFound(w, r)
		return
	}

	overview, err := c.service.GetAppOverview(values.Slug(slug))
	if err != nil {
		log.Printf("application overview lookup failed for %q: %v", slug, err)
		http.Error(w, "Unable to load application overview", http.StatusInternalServerError)
		return
	}

	data := DashboardData{
		CurrentPage:         "Overview",
		SessionData:         session.NewSessionData(r.Context()),
		CurrentAppData:      currentApp,
		SetupGuideCompleted: overview.SetupGuideCompleted,
	}

	tmpl, err := views.ParseTemplate(views.SidebarLayout, "templates/console/dashboard.gohtml")
	if err != nil {
		log.Println(err)
		http.Error(w, "Unable to render application dashboard", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "index.gohtml", data); err != nil {
		log.Printf("application dashboard render failed: %v", err)
	}
}

func (c *appController) RenderFeaturePage(w http.ResponseWriter, r *http.Request) {
	section := chi.URLParam(r, "section")
	sectionTitles := map[string]string{
		"distribution-groups": "Distribution Groups",
		"announcements":       "Announcements",
		"feeds":               "Feeds",
		"analytics":           "Analytics",
		"settings":            "Settings",
		"configuration":       "Configuration",
		"configs":             "Configurations",
	}
	title := sectionTitles[section]
	if section == "" || section == "configs" {
		platform := chi.URLParam(r, "platform")
		if platform == "" {
			section = "configs"
			title = sectionTitles[section]
		} else {
			title = platform + " Configurations"
		}
	} else if title == "" {
		http.NotFound(w, r)
		return
	}

	slug := values.Slug(chi.URLParam(r, "slug"))
	currentApp, err := c.service.GetCurrentAppData(slug)
	if err != nil {
		log.Printf("application feature lookup failed for %q: %v", slug, err)
		http.Error(w, "Unable to load application", http.StatusInternalServerError)
		return
	}
	if currentApp.AppID == 0 {
		http.NotFound(w, r)
		return
	}

	data := FeaturePageData{
		Title:          title,
		CurrentPage:    title,
		SessionData:    session.NewSessionData(r.Context()),
		CurrentAppData: currentApp,
	}
	tmpl, err := views.ParseTemplate(views.SidebarLayout, "templates/console/feature_unavailable.gohtml")
	if err != nil {
		log.Printf("feature page template parse failed: %v", err)
		http.Error(w, "Unable to render page", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "index.gohtml", data); err != nil {
		log.Printf("feature page render failed: %v", err)
	}
}
