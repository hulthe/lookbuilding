package main

import (
	"regexp"

	"github.com/coreos/go-semver/semver"
)

var (
	rxSemVerPrefix = regexp.MustCompile(`^[^\d]*`)
)

type SemVerTag struct {
	prefix  string
	version semver.Version
}

// Return a
// Returns nil if the tag did not parse as semver
func parseTagAsSemVer(tag string) *SemVerTag {
	var prefix string
	loc := rxSemVerPrefix.FindStringIndex(tag)
	if loc != nil {
		prefix = tag[:loc[1]]
		tag = tag[loc[1]:]
	}

	version, err := semver.NewVersion(tag)
	if err != nil {
		return nil
	}

	svt := SemVerTag{
		prefix,
		*version,
	}

	return &svt
}

func (SemVerTag) asTag() string {
	return ""
}
