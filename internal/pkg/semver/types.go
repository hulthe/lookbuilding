package semver

import "github.com/coreos/go-semver/semver"

type Tag struct {
	Prefix  string
	Version semver.Version
}
