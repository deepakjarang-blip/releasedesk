package app

import (
	"errors"
	"github.com/brewbits-co/releasedesk/internal/domains/app"
	"github.com/brewbits-co/releasedesk/internal/values"
	"github.com/brewbits-co/releasedesk/pkg/session"
	"sort"
)

func (s *service) GetPublicApps() ([]app.App, error) {
	applications, err := s.appRepo.Find()
	if err != nil {
		return nil, err
	}

	publicApps := make([]app.App, 0, len(applications))
	for _, application := range applications {
		if !application.Private {
			publicApps = append(publicApps, application)
		}
	}
	return publicApps, nil
}

func (s *service) GetPublicApp(slug values.Slug) (app.App, error) {
	application, err := s.appRepo.GetBySlug(slug)
	if err != nil {
		return app.App{}, err
	}
	if application.ID == 0 || application.Private {
		return app.App{}, errors.New("app not found")
	}
	return application, nil
}

func (s *service) GetUserAccessibleApps(userID int) ([]app.App, error) {
	applications, err := s.appRepo.Find()
	if err != nil {
		return nil, err
	}

	for i := range applications {
		if err := s.appRepo.GetPlatformAvailability(&applications[i]); err != nil {
			return nil, err
		}
	}

	return applications, nil
}

func (s *service) GetAppOverview(slug values.Slug) (app.Overview, error) {
	applicationEntity, err := s.appRepo.GetBySlug(slug)
	if err != nil {
		return app.Overview{}, err
	}

	data := app.Overview{
		SetupGuideCompleted: applicationEntity.SetupGuideCompleted,
	}

	return data, nil
}

func (s *service) GetCurrentAppData(slug values.Slug) (session.CurrentAppData, error) {
	applicationEntity, err := s.appRepo.GetBySlug(slug)
	if err != nil {
		return session.CurrentAppData{}, err
	}

	platforms, err := s.platformRepo.FindByAppID(applicationEntity.ID)
	if err != nil {
		return session.CurrentAppData{}, err
	}

	var appPlatforms []session.CurrentPlatformData
	for _, platformEntity := range platforms {
		appPlatforms = append(appPlatforms, session.CurrentPlatformData{
			PlatformID: platformEntity.ID,
			OS:         platformEntity.OS,
			AppSlug:    slug,
		})
	}

	// Desired platform order
	platformOrder := map[values.OS]int{
		values.Android: 1,
		values.IOS:     2,
		values.Windows: 3,
		values.MacOS:   4,
		values.Linux:   5,
	}

	// Sorting based on the platform order
	sort.Slice(appPlatforms, func(i, j int) bool {
		return platformOrder[appPlatforms[i].OS] < platformOrder[appPlatforms[j].OS]
	})

	return session.CurrentAppData{
		AppID:     applicationEntity.ID,
		AppName:   applicationEntity.Name,
		AppSlug:   applicationEntity.Slug,
		Publisher: applicationEntity.Publisher.String,
		Platforms: appPlatforms,
	}, nil
}
