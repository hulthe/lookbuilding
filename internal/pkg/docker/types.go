package docker

import (
	"github.com/docker/docker/api/types"
	d "github.com/opencontainers/go-digest"
)

type LabeledContainer struct {
	Container   types.Container
	Mode        VersioningMode
	ImageDigest d.Digest
}
