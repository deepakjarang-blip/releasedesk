package release

import (
	"database/sql"
	"errors"
	"testing"

	appdomain "github.com/brewbits-co/releasedesk/internal/domains/app"
	releasedomain "github.com/brewbits-co/releasedesk/internal/domains/release"
	"github.com/brewbits-co/releasedesk/internal/values"
)

type testAppRepository struct {
	apps map[values.Slug]appdomain.App
}

func (r *testAppRepository) Save(*appdomain.App) error { return nil }
func (r *testAppRepository) Find() ([]appdomain.App, error) {
	return nil, nil
}
func (r *testAppRepository) GetBySlug(slug values.Slug) (appdomain.App, error) {
	app, ok := r.apps[slug]
	if !ok {
		return appdomain.App{}, nil
	}
	return app, nil
}
func (r *testAppRepository) Update(appdomain.App) error { return nil }
func (r *testAppRepository) Delete(appdomain.App) error { return nil }
func (r *testAppRepository) GetPlatformAvailability(*appdomain.App) error {
	return nil
}
func (r *testAppRepository) SaveSetupGuide(appdomain.SetupGuide) error { return nil }

type testReleaseRepository struct {
	channels []releasedomain.Channel
	releases []releasedomain.Release
}

func (r *testReleaseRepository) Save(item *releasedomain.Release) error {
	item.ID = len(r.releases) + 1
	r.releases = append(r.releases, *item)
	return nil
}
func (r *testReleaseRepository) Update(*releasedomain.Release) error { return nil }
func (r *testReleaseRepository) FindByAppIDAndChannel(appID, channelID int) ([]releasedomain.BasicInfo, error) {
	return nil, nil
}
func (r *testReleaseRepository) FindChannelsByAppID(appID int) ([]releasedomain.Channel, error) {
	var channels []releasedomain.Channel
	for _, channel := range r.channels {
		if channel.AppID == appID {
			channels = append(channels, channel)
		}
	}
	return channels, nil
}
func (r *testReleaseRepository) GetByAppIDAndVersion(appID int, version string) (releasedomain.Release, error) {
	for _, item := range r.releases {
		if item.AppID == appID && item.Version == version {
			return item, nil
		}
	}
	return releasedomain.Release{}, sql.ErrNoRows
}
func (r *testReleaseRepository) GetByID(id int) (releasedomain.Release, error) {
	return releasedomain.Release{}, sql.ErrNoRows
}
func (r *testReleaseRepository) LinkBuild(int, int, values.OS) error { return nil }
func (r *testReleaseRepository) UnlinkBuild(int, int, values.OS) error {
	return nil
}
func (r *testReleaseRepository) SetBuilds(int, []releasedomain.LinkedBuilds) error {
	return nil
}

func TestCreateReleaseIsScopedToApplication(t *testing.T) {
	appRepo := &testAppRepository{apps: map[values.Slug]appdomain.App{
		"app-one": {BasicInfo: appdomain.BasicInfo{ID: 1, Slug: "app-one"}},
		"app-two": {BasicInfo: appdomain.BasicInfo{ID: 2, Slug: "app-two"}},
	}}
	releaseRepo := &testReleaseRepository{channels: []releasedomain.Channel{
		{ID: 10, AppID: 1},
		{ID: 20, AppID: 2},
	}}
	service := &service{appRepo: appRepo, releaseRepo: releaseRepo}

	first, err := service.CreateRelease("app-one", releasedomain.BasicInfo{Version: "1.0", TargetChannel: 10})
	if err != nil {
		t.Fatalf("create first app release: %v", err)
	}
	second, err := service.CreateRelease("app-two", releasedomain.BasicInfo{Version: "1.0", TargetChannel: 20})
	if err != nil {
		t.Fatalf("create second app release with same version: %v", err)
	}
	if first.AppID != 1 || second.AppID != 2 {
		t.Fatalf("releases assigned to wrong apps: first=%d second=%d", first.AppID, second.AppID)
	}

	_, err = service.CreateRelease("app-one", releasedomain.BasicInfo{Version: "1.0", TargetChannel: 10})
	if !errors.Is(err, releasedomain.ErrReleaseVersionExists) {
		t.Fatalf("duplicate version error = %v, want %v", err, releasedomain.ErrReleaseVersionExists)
	}

	_, err = service.CreateRelease("app-two", releasedomain.BasicInfo{Version: "2.0", TargetChannel: 10})
	if !errors.Is(err, releasedomain.ErrReleaseChannelNotFound) {
		t.Fatalf("foreign channel error = %v, want %v", err, releasedomain.ErrReleaseChannelNotFound)
	}
}

var _ appdomain.AppRepository = (*testAppRepository)(nil)
var _ releasedomain.ReleaseRepository = (*testReleaseRepository)(nil)
