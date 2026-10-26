package app

import (
	"github.com/brewbits-co/releasedesk/pkg/session"
)

type DashboardData struct {
	CurrentPage string
	session.SessionData
	session.CurrentAppData
	SetupGuideCompleted bool
}

type FeaturePageData struct {
	Title       string
	CurrentPage string
	session.SessionData
	session.CurrentAppData
}
