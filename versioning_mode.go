package main

import (
	"fmt"
	"sort"
)

const (
	versioningModeLabel = "lookbuilding.mode"
)

type VersioningMode interface {
	Label() string
	ShouldUpdate(currentTag string, availableTags []Tag) *Tag
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
func (SameTag) ShouldUpdate(currentTag string, availableTags []Tag) *Tag {
	fmt.Println("Not implemented: 'same_tag' versioning mode")
	return nil // TODO: implement me
}

func semVerShouldUpdate(currentTag string, availableTags []Tag, isValid func(current, available SemVerTag) bool) *Tag {
	currentSemVer := parseTagAsSemVer(currentTag)
	if currentSemVer == nil {
		return nil
	}

	semverTags := make([]Tag, 0)

	for _, tag := range availableTags {
		if tag.SemVer != nil && isValid(*currentSemVer, *tag.SemVer) {
			semverTags = append(semverTags, tag)
		}
	}

	if len(semverTags) == 0 {
		return nil
	}

	sort.Slice(semverTags, func(i, j int) bool {
		a := semverTags[i].SemVer.version
		b := semverTags[j].SemVer.version
		return b.LessThan(a)
	})

	return &semverTags[0]
}

func (SemVerMajor) Label() string { return "semver_major" }
func (SemVerMajor) ShouldUpdate(currentTag string, availableTags []Tag) *Tag {
	return semVerShouldUpdate(currentTag, availableTags, func(current, available SemVerTag) bool {
		// The new version should be greater
		return current.version.LessThan(available.version)
	})
}

func (SemVerMinor) Label() string { return "semver_minor" }
func (SemVerMinor) ShouldUpdate(currentTag string, availableTags []Tag) *Tag {
	return semVerShouldUpdate(currentTag, availableTags, func(current, available SemVerTag) bool {
		// The new version should be greater, but still the same major number
		return current.version.LessThan(available.version) &&
			current.version.Major == available.version.Major
	})
}

func (SemVerPatch) Label() string { return "semver_patch" }
func (SemVerPatch) ShouldUpdate(currentTag string, availableTags []Tag) *Tag {
	return semVerShouldUpdate(currentTag, availableTags, func(current, available SemVerTag) bool {
		// The new version should be greater, but still the same major & minor number
		return current.version.LessThan(available.version) &&
			current.version.Major == available.version.Major &&
			current.version.Minor == available.version.Minor
	})
}

func ParseVersioningMode(input string) *VersioningMode {
	for _, mode := range AllModes {
		if mode.Label() == input {
			return &mode
		}
	}
	return nil
}
