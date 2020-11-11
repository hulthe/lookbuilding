package docker

import (
	"fmt"
	"strings"
)

// Extract the repository owner (if any), repository and tag (if any) from a docker image name
func (lc LabeledContainer) SplitImageParts() (*string, string, *string) {
	name := lc.Container.Image

	var repository string
	var owner *string
	var tag *string

	slashIndex := strings.Index(name, "/")
	if slashIndex >= 0 {
		tmp := name[:slashIndex]
		owner = &tmp
		name = name[slashIndex+1:]
	}

	colonIndex := strings.Index(name, ":")
	if colonIndex >= 0 {
		tmp := name[colonIndex+1:]
		tag = &tmp

		repository = name[:colonIndex]
	} else {
		repository = name
	}

	return owner, repository, tag
}

func (lc LabeledContainer) GetName() string {
	if len(lc.Container.Names) >= 0 {
		// trim prefixed "/"
		return lc.Container.Names[0][1:]
	} else {
		return lc.Container.ID[:10]
	}
}

func CombineImageParts(owner *string, repository string, tag *string) string {
	image := repository
	if owner != nil {
		image = fmt.Sprintf("%s/%s", *owner, image)
	}
	if tag != nil {
		image = fmt.Sprintf("%s:%s", image, *tag)
	}

	return image
}
