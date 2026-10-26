package main

import (
	portalCtrl "github.com/brewbits-co/releasedesk/internal/controllers/portal"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/dig"
	"log"
	"net/http"
)

// buildPortal sets up the Chi router with necessary middleware, resolves dependencies, configures routes, and serves static files.
func buildPortal(container *dig.Container) *http.Server {
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.NoCache)

	err := container.Invoke(func(portalController portalCtrl.PortalController) {
		router.Get("/", portalController.RenderHome)
		router.Get("/artifacts/{checksum}", portalController.HandleArtifactDownload)
		router.Get("/{slug}", portalController.RenderApp)
	})
	if err != nil {
		log.Fatalf("Failed to initialize Download Portal: %v", err)
	}

	// Handle static files from the assets folder
	router.Handle("/assets/*", http.FileServer(http.FS(assetsFS)))

	// Create HTTP server
	server := &http.Server{
		Addr:    ":9090",
		Handler: router,
	}
	return server
}
