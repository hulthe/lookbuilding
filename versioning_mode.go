package main

import (
	"sort"
)

const (
	versioningModeLabel = "lookbuilding.mode"
)

type VersioningMode interface {
	Label() string
	ShouldUpdate(currentTag string, availableTags []Tag) *Tag
}

type SameTag struct {}
type SemVerMajor struct {}
type SemVerMinor struct {}
type SemVerPatch struct {}

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
	return nil // TODO: implement me
}

func (SemVerMajor) Label() string { return "semver_major" }
func (SemVerMajor) ShouldUpdate(currentTag string, availableTags []Tag) *Tag {
	currentSemVer := parseTagAsSemVer(currentTag)
	if currentSemVer == nil {
		return nil
	}

	semverTags := make([]Tag, 0)

	for _, tag := range availableTags {
		if tag.SemVer != nil {
			semverTags = append(semverTags, tag);
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

func (SemVerMinor) Label() string { return "semver_minor" }
func (SemVerMinor) ShouldUpdate(currentTag string, availableTags []Tag) *Tag {
	return nil // TODO: implement me
}

func (SemVerPatch) Label() string { return "semver_patch" }
func (SemVerPatch) ShouldUpdate(currentTag string, availableTags []Tag) *Tag {
	return nil // TODO: implement me
}


func ParseVersioningMode(input string) *VersioningMode {
	for _, mode := range AllModes {
		if mode.Label() == input {
			return &mode
		}
	}
	return nil
}

