package misc

import (
	"github.com/brewbits-co/releasedesk/internal/views"
	"github.com/brewbits-co/releasedesk/pkg/session"
	"log"
	"net/http"
)

func (c *miscController) RenderHomepage(w http.ResponseWriter, r *http.Request) {
	apps, err := c.service.GetUserAccessibleApps(r.Context().Value("userID").(int))
	if err != nil {
		log.Printf("unable to list applications: %v", err)
		http.Error(w, "Unable to load applications", http.StatusInternalServerError)
		return
	}
	data := HomepageData{
		SessionData: session.NewSessionData(r.Context()),
		Apps:        apps,
	}

	tmpl, err := views.ParseTemplate(views.NavbarLayout, "templates/console/homepage.gohtml")
	if err != nil {
		log.Printf("application list template parse failed: %v", err)
		http.Error(w, "Unable to render applications", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "index.gohtml", data); err != nil {
		log.Printf("application list render failed: %v", err)
	}
}
