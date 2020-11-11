package versioning

import (
	"sort"

	l "hulthe.net/lookbuilding/internal/pkg/logging"
	"hulthe.net/lookbuilding/internal/pkg/registry"
	"hulthe.net/lookbuilding/internal/pkg/semver"
)

const ModeLabel = "lookbuilding.mode"

type Mode interface {
	Label() string
	ShouldUpdate(currentTag string, availableTags []registry.Tag) *registry.Tag
}

type SameTag struct{}
type SemVerMajor struct{}
type SemVerMinor struct{}
type SemVerPatch struct{}

var (
	AllModes = [...]Mode{
		SameTag{},
		SemVerMajor{},
		SemVerMinor{},
		SemVerPatch{},
	}
)

func (SameTag) Label() string { return "same_tag" }
func (SameTag) ShouldUpdate(currentTag string, availableTags []registry.Tag) *registry.Tag {
	l.Logger.Errorf("Not implemented: 'same_tag' versioning mode")
	return nil // TODO: implement me
}

func semVerShouldUpdate(currentTag string, availableTags []registry.Tag, isValid func(current, available semver.Tag) bool) *registry.Tag {
	currentSemVer := semver.ParseTagAsSemVer(currentTag)
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
func (SemVerMajor) ShouldUpdate(currentTag string, availableTags []registry.Tag) *registry.Tag {
	return semVerShouldUpdate(currentTag, availableTags, func(current, available semver.Tag) bool {
		// The new version should be greater
		return current.Version.LessThan(available.Version)
	})
}

func (SemVerMinor) Label() string { return "semver_minor" }
func (SemVerMinor) ShouldUpdate(currentTag string, availableTags []registry.Tag) *registry.Tag {
	return semVerShouldUpdate(currentTag, availableTags, func(current, available semver.Tag) bool {
		// The new version should be greater, but still the same major number
		return current.Version.LessThan(available.Version) &&
			current.Version.Major == available.Version.Major
	})
}

func (SemVerPatch) Label() string { return "semver_patch" }
func (SemVerPatch) ShouldUpdate(currentTag string, availableTags []registry.Tag) *registry.Tag {
	return semVerShouldUpdate(currentTag, availableTags, func(current, available semver.Tag) bool {
		// The new version should be greater, but still the same major & minor number
		return current.Version.LessThan(available.Version) &&
			current.Version.Major == available.Version.Major &&
			current.Version.Minor == available.Version.Minor
	})
}

func ParseMode(input string) *Mode {
	for _, mode := range AllModes {
		if mode.Label() == input {
			return &mode
		}
	}
	return nil
}
