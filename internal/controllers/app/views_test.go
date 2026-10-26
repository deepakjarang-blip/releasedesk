package app

import (
	"strings"
	"testing"

	"github.com/brewbits-co/releasedesk/internal/views"
	"github.com/brewbits-co/releasedesk/pkg/session"
)

func TestFeatureUnavailableTemplateRenders(t *testing.T) {
	tmpl, err := views.ParseTemplate(views.SidebarLayout, "templates/console/feature_unavailable.gohtml")
	if err != nil {
		t.Fatalf("parse feature page template: %v", err)
	}

	data := FeaturePageData{
		Title:       "Settings",
		CurrentPage: "Settings",
		CurrentAppData: session.CurrentAppData{
			AppName: "Example App",
			AppSlug: "example-app",
		},
	}

	var output strings.Builder
	if err := tmpl.ExecuteTemplate(&output, "index.gohtml", data); err != nil {
		t.Fatalf("render feature page template: %v", err)
	}
	for _, expected := range []string{
		"Settings",
		"This section is not available yet.",
		`href="/dashboard/example-app"`,
	} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("rendered page does not contain %q", expected)
		}
	}
}
