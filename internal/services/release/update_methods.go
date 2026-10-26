package release

import (
	"fmt"
	"github.com/brewbits-co/releasedesk/internal/domains/release"
	"github.com/brewbits-co/releasedesk/internal/values"
)

// UpdateReleaseBasicInfo updates the BasicInfo of a release
// The ID, AppID, and Version fields cannot be updated.
func (s *service) UpdateReleaseBasicInfo(info release.BasicInfo) (release.Release, error) {
	// Get the existing release by ID
	existingRelease, err := s.releaseRepo.GetByID(info.ID)
	if err != nil {
		return release.Release{}, err
	}

	// Update only the fields that are allowed to be updated
	existingRelease.TargetChannel = info.TargetChannel
	existingRelease.Status = info.Status
	existingRelease.BuildSelection = info.BuildSelection

	// Save the updated release
	err = s.releaseRepo.Update(&existingRelease)
	if err != nil {
		return release.Release{}, err
	}

	return existingRelease, nil
}

func (s *service) UpdateReleaseBuilds(releaseID int, selections map[values.OS]int) error {
	releaseEntity, err := s.releaseRepo.GetByID(releaseID)
	if err != nil {
		return err
	}
	platforms, err := s.platformRepo.FindByAppID(releaseEntity.AppID)
	if err != nil {
		return err
	}
	platformIDs := make(map[values.OS]int, len(platforms))
	for _, platform := range platforms {
		platformIDs[platform.OS] = platform.ID
	}

	links := make([]release.LinkedBuilds, 0, len(selections))
	for os, buildID := range selections {
		if buildID <= 0 {
			continue
		}
		platformID, ok := platformIDs[os]
		if !ok {
			return fmt.Errorf("platform %s is not configured for this app", os)
		}
		if _, err := s.buildRepo.GetByIDAndPlatformID(buildID, platformID); err != nil {
			return fmt.Errorf("build %d is not available for %s", buildID, os)
		}
		links = append(links, release.LinkedBuilds{BuildID: buildID, OS: os})
	}
	return s.releaseRepo.SetBuilds(releaseID, links)
}
