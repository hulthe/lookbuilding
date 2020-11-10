package semver

import (
	"fmt"
	"regexp"

	"github.com/coreos/go-semver/semver"
)

var (
	rxSemVerPrefix = regexp.MustCompile(`^[^\d]*`)
)

// Returns nil if the tag did not parse as semver
func ParseTagAsSemVer(dockerTag string) *Tag {
	var prefix string
	loc := rxSemVerPrefix.FindStringIndex(dockerTag)
	if loc != nil {
		prefix = dockerTag[:loc[1]]
		dockerTag = dockerTag[loc[1]:]
	}

	version, err := semver.NewVersion(dockerTag)
	if err != nil {
		return nil
	}

	svt := Tag{
		prefix,
		*version,
	}

	return &svt
}

func (svt Tag) String() string {
	return fmt.Sprintf("%s%s", svt.Prefix, svt.Version.String())
}
