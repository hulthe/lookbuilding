package main

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
	return nil // TODO: implement me
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

