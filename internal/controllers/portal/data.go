package portal

import (
	"github.com/brewbits-co/releasedesk/internal/domains/app"
	"github.com/brewbits-co/releasedesk/internal/domains/build"
	"github.com/brewbits-co/releasedesk/internal/values"
	"time"
)

type homeData struct {
	Apps []app.App
}

type appData struct {
	App      app.App
	Releases []portalRelease
}

type portalRelease struct {
	Version   string
	Channel   string
	CreatedAt time.Time
	Builds    []portalBuild
}

type portalBuild struct {
	OS        values.OS
	Version   string
	Number    string
	Artifacts []build.Artifact
}
