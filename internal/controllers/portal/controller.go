package portal

import (
	appservice "github.com/brewbits-co/releasedesk/internal/services/app"
	buildservice "github.com/brewbits-co/releasedesk/internal/services/build"
	releaseservice "github.com/brewbits-co/releasedesk/internal/services/release"
	"net/http"
)

type PortalController interface {
	RenderHome(w http.ResponseWriter, r *http.Request)
	RenderApp(w http.ResponseWriter, r *http.Request)
	HandleArtifactDownload(w http.ResponseWriter, r *http.Request)
}

func NewPortalController(apps appservice.Service, releases releaseservice.Service, builds buildservice.Service) PortalController {
	return &controller{apps: apps, releases: releases, builds: builds}
}

type controller struct {
	apps     appservice.Service
	releases releaseservice.Service
	builds   buildservice.Service
}
