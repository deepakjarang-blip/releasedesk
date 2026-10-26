package app

import (
	"database/sql"
	"github.com/brewbits-co/releasedesk/internal/domains/app"
	"github.com/brewbits-co/releasedesk/internal/values"
	"strings"
)

func (s *service) CreateApp(info app.BasicInfo) (app.App, error) {
	info.SetupGuideCompleted = false
	info.Slug.Format()

	newApp := app.NewApp(info)

	err := newApp.IsValid()
	if err != nil {
		return app.App{}, err
	}

	err = s.appRepo.Save(&newApp)
	if err != nil {
		return app.App{}, err
	}

	return newApp, nil
}

func (s *service) UpdateAppPublisher(slug values.Slug, publisher string) error {
	application, err := s.appRepo.GetBySlug(slug)
	if err != nil {
		return err
	}
	publisher = strings.TrimSpace(publisher)
	application.Publisher = sql.NullString{String: publisher, Valid: publisher != ""}
	return s.appRepo.Update(application)
}
