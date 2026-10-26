package portal

import (
	"github.com/brewbits-co/releasedesk/internal/domains/release"
	"github.com/brewbits-co/releasedesk/internal/values"
	"github.com/brewbits-co/releasedesk/internal/views"
	"github.com/go-chi/chi/v5"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strings"
)

func (c *controller) RenderHome(w http.ResponseWriter, r *http.Request) {
	apps, err := c.apps.GetPublicApps()
	if err != nil {
		http.Error(w, "Unable to load applications", http.StatusInternalServerError)
		return
	}
	sort.Slice(apps, func(i, j int) bool { return apps[i].Name < apps[j].Name })
	c.render(w, "templates/portal/home.gohtml", homeData{
		Apps:       apps,
		PortalBase: portalBasePath(r),
	}, portalBasePath(r))
}

func (c *controller) RenderApp(w http.ResponseWriter, r *http.Request) {
	slug := values.Slug(chi.URLParam(r, "slug"))
	application, err := c.apps.GetPublicApp(slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	currentApp, err := c.apps.GetCurrentAppData(slug)
	if err != nil {
		http.Error(w, "Unable to load application platforms", http.StatusInternalServerError)
		return
	}
	platformIDs := make(map[values.OS]int, len(currentApp.Platforms))
	for _, platform := range currentApp.Platforms {
		platformIDs[platform.OS] = platform.PlatformID
	}

	channels, err := c.releases.GetReleaseChannels(application.ID)
	if err != nil {
		http.Error(w, "Unable to load releases", http.StatusInternalServerError)
		return
	}

	data := appData{
		App:        application,
		Releases:   make([]portalRelease, 0),
		PortalBase: portalBasePath(r),
	}
	for _, channel := range channels {
		if channel.Closed {
			continue
		}
		channelReleases, err := c.releases.ListReleasesByChannel(application.ID, channel.ID)
		if err != nil {
			http.Error(w, "Unable to load releases", http.StatusInternalServerError)
			return
		}
		for _, releaseInfo := range channelReleases {
			if releaseInfo.Status != release.Published {
				continue
			}
			releaseDetails, err := c.releases.GetReleaseByID(releaseInfo.ID)
			if err != nil {
				http.Error(w, "Unable to load release details", http.StatusInternalServerError)
				return
			}

			portalItem := portalRelease{
				Version:   releaseDetails.Version,
				Channel:   channel.Name,
				CreatedAt: releaseDetails.CreatedAt,
				Builds:    make([]portalBuild, 0, len(releaseDetails.Builds)),
			}
			for _, platform := range currentApp.Platforms {
				buildInfo, linked := releaseDetails.Builds[platform.OS]
				if releaseDetails.BuildSelection == release.Last {
					availableBuilds, err := c.builds.GetPlatformBuilds(platform.PlatformID)
					if err != nil {
						http.Error(w, "Unable to load builds", http.StatusInternalServerError)
						return
					}
					if len(availableBuilds) > 0 {
						buildInfo = availableBuilds[0]
						linked = true
					}
				}
				if !linked {
					continue
				}
				platformID, ok := platformIDs[platform.OS]
				if !ok {
					continue
				}
				buildDetails, err := c.builds.GetBuildDetails(platformID, buildInfo.Number)
				if err != nil {
					log.Printf("portal build lookup failed for %s %s: %v", platform.OS, buildInfo.Number, err)
					continue
				}
				portalItem.Builds = append(portalItem.Builds, portalBuild{
					OS:        platform.OS,
					Version:   buildDetails.Version,
					Number:    buildDetails.Number,
					Artifacts: buildDetails.Artifacts,
				})
			}
			data.Releases = append(data.Releases, portalItem)
		}
	}
	sort.Slice(data.Releases, func(i, j int) bool {
		return data.Releases[i].CreatedAt.After(data.Releases[j].CreatedAt)
	})
	c.render(w, "templates/portal/app.gohtml", data, data.PortalBase)
}

func portalBasePath(r *http.Request) string {
	if r.URL.Path == "/portal" || strings.HasPrefix(r.URL.Path, "/portal/") {
		return "/portal"
	}
	return ""
}

func (c *controller) render(w http.ResponseWriter, templatePath string, data any, portalBase string) {
	tmpl, err := template.New(templatePath).Funcs(template.FuncMap{
		"artifactURL": func(checksum, filename string) string {
			return portalBase + "/artifacts/" + checksum + "?filename=" + template.URLQueryEscaper(filename)
		},
	}).ParseFS(views.ViewsFS, templatePath)
	if err != nil {
		http.Error(w, "Unable to render page", http.StatusInternalServerError)
		log.Printf("portal template parse failed: %v", err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, templatePath[strings.LastIndex(templatePath, "/")+1:], data); err != nil {
		log.Printf("portal template execution failed: %v", err)
	}
}
