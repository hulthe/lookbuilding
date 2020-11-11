package docker

import (
	"hulthe.net/lookbuilding/internal/pkg/versioning"

	"github.com/docker/docker/api/types"
)

type LabeledContainer struct {
	Container types.Container
	Mode      versioning.Mode
}
