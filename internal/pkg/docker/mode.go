package docker

import (
	"sort"

	l "hulthe.net/lookbuilding/internal/pkg/logging"
	"hulthe.net/lookbuilding/internal/pkg/registry"
	"hulthe.net/lookbuilding/internal/pkg/semver"
)

const VersioningModeLabel = "lookbuilding.mode"

type VersioningMode interface {
	Label() string
	ShouldUpdate(container LabeledContainer, availableTags []registry.Tag) *registry.Tag
}

type SameTag struct{}
type SemVerMajor struct{}
type SemVerMinor struct{}
type SemVerPatch struct{}

var (
	AllModes = [...]VersioningMode{
		SameTag{},
		SemVerMajor{},
		SemVerMinor{},
		SemVerPatch{},
	}
)

func (SameTag) Label() string { return "same_tag" }
func (SameTag) ShouldUpdate(container LabeledContainer, availableTags []registry.Tag) *registry.Tag {
	_, _, currentTag := container.SplitImageParts()
	if currentTag == nil {
		l.Logger.Errorf("container %s has no tag, won't try to update", container.GetName())
		return nil
	}

	for _, tag := range availableTags {
		if tag.Name == *currentTag {
			remoteDigest, err := tag.GetDigest()
			if err != nil {
				l.Logger.Errorf("failed to get digest for tag %s", tag.Name)
				l.Logger.Error(err)
			}

			l.Logger.Debug(remoteDigest.String(), container.ImageDigest)
			if container.ImageDigest != remoteDigest {
				return &tag
			}
			return nil
		}
	}

	return nil
}

func semVerShouldUpdate(container LabeledContainer, availableTags []registry.Tag, isValid func(current, available semver.Tag) bool) *registry.Tag {
	_, _, currentTag := container.SplitImageParts()
	if currentTag == nil {
		l.Logger.Errorf("container %s has no tag, won't try to update", container.GetName())
		return nil
	}

	currentSemVer := semver.ParseTagAsSemVer(*currentTag)
	if currentSemVer == nil {
		return nil
	}

	semverTags := make([]registry.Tag, 0)

	for _, tag := range availableTags {
		if tag.SemVer != nil && isValid(*currentSemVer, *tag.SemVer) {
			semverTags = append(semverTags, tag)
		}
	}

	if len(semverTags) == 0 {
		return nil
	}

	sort.Slice(semverTags, func(i, j int) bool {
		a := semverTags[i].SemVer.Version
		b := semverTags[j].SemVer.Version
		return b.LessThan(a)
	})

	return &semverTags[0]
}

func (SemVerMajor) Label() string { return "semver_major" }
func (SemVerMajor) ShouldUpdate(container LabeledContainer, availableTags []registry.Tag) *registry.Tag {
	return semVerShouldUpdate(container, availableTags, func(current, available semver.Tag) bool {
		// The new version should be greater
		return current.Version.LessThan(available.Version)
	})
}

func (SemVerMinor) Label() string { return "semver_minor" }
func (SemVerMinor) ShouldUpdate(container LabeledContainer, availableTags []registry.Tag) *registry.Tag {
	return semVerShouldUpdate(container, availableTags, func(current, available semver.Tag) bool {
		// The new version should be greater, but still the same major number
		return current.Version.LessThan(available.Version) &&
			current.Version.Major == available.Version.Major
	})
}

func (SemVerPatch) Label() string { return "semver_patch" }
func (SemVerPatch) ShouldUpdate(container LabeledContainer, availableTags []registry.Tag) *registry.Tag {
	return semVerShouldUpdate(container, availableTags, func(current, available semver.Tag) bool {
		// The new version should be greater, but still the same major & minor number
		return current.Version.LessThan(available.Version) &&
			current.Version.Major == available.Version.Major &&
			current.Version.Minor == available.Version.Minor
	})
}

func ParseMode(input string) *VersioningMode {
	for _, mode := range AllModes {
		if mode.Label() == input {
			return &mode
		}
	}
	return nil
}
