package release

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/brewbits-co/releasedesk/internal/domains/release"
	"github.com/brewbits-co/releasedesk/internal/values"
)

func (s *service) CreateRelease(slug values.Slug, info release.BasicInfo) (release.Release, error) {
	appEntity, err := s.appRepo.GetBySlug(slug)
	if err != nil {
		return release.Release{}, err
	}
	if appEntity.ID == 0 {
		return release.Release{}, fmt.Errorf("application %q not found", slug)
	}

	existing, err := s.releaseRepo.GetByAppIDAndVersion(appEntity.ID, info.Version)
	if err == nil && existing.ID != 0 {
		return release.Release{}, release.ErrReleaseVersionExists
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return release.Release{}, err
	}

	channels, err := s.releaseRepo.FindChannelsByAppID(appEntity.ID)
	if err != nil {
		return release.Release{}, err
	}
	channelBelongsToApp := false
	for _, channel := range channels {
		if channel.ID == info.TargetChannel {
			channelBelongsToApp = true
			break
		}
	}
	if !channelBelongsToApp {
		return release.Release{}, release.ErrReleaseChannelNotFound
	}

	info.AppID = appEntity.ID
	info.Status = release.Unpublished
	info.BuildSelection = release.Last

	newRelease := release.NewRelease(info)

	err = s.releaseRepo.Save(&newRelease)
	if err != nil {
		return release.Release{}, err
	}

	return newRelease, nil
}
