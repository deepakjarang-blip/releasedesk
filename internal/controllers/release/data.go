package release

import (
	"github.com/brewbits-co/releasedesk/internal/domains/build"
	"github.com/brewbits-co/releasedesk/internal/domains/release"
	"github.com/brewbits-co/releasedesk/internal/values"
	"github.com/brewbits-co/releasedesk/pkg/session"
)

type ReleaseListData struct {
	CurrentPage string
	session.SessionData
	session.CurrentAppData
	Releases         []release.BasicInfo
	Channels         []release.Channel
	CurrentChannelID int
}

type ReleaseSummaryData struct {
	CurrentPage string
	session.SessionData
	session.CurrentAppData
	release.Release
	Channels        []release.Channel
	BuildSelections []BuildSelectionOption
}

type BuildSelectionOption struct {
	OS              values.OS
	Builds          []build.BasicInfo
	SelectedBuildID int
}

type ReleaseNotesData struct {
	CurrentPage string
	session.SessionData
	session.CurrentAppData
	Release      release.Release
	ReleaseNotes release.ReleaseNotes
}
