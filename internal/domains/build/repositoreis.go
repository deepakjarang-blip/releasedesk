package build

type BuildRepository interface {
	Save(build *Build) error
	FindByPlatformID(platformID int) ([]BasicInfo, error)
	GetByPlatformIDAndNumber(platformID int, number string) (Build, error)
	GetByIDAndPlatformID(buildID int, platformID int) (Build, error)
	GetPublicArtifactByChecksum(checksum string) (Artifact, error)
}
