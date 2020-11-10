package main

import (
	"fmt"
	"github.com/heroku/docker-registry-client/registry"
	digest "github.com/opencontainers/go-digest"
	"hulthe.net/lookbuilding/internal/pkg/semver"
)

type Tag struct {
	Name   string
	SemVer *semver.Tag
	Digest digest.Digest
}

func anonymousClient() (*registry.Registry, error) {
	url := "https://registry-1.docker.io/"
	username := "" // anonymous
	password := "" // anonymous

	registry, err := registry.New(url, username, password)
	if err != nil {
		return nil, err
	}

	registry.Logf = Logger.Infof

	return registry, nil
}

func getDockerRepoTags(hub *registry.Registry, maybeOwner *string, repository string) ([]Tag, error) {
	if maybeOwner != nil {
		repository = fmt.Sprintf("%s/%s", *maybeOwner, repository)
	}

	tags, err := hub.Tags(repository)
	if err != nil {
		return nil, err
	}

	var out []Tag

	for _, tag := range tags {
		digest, err := hub.ManifestDigest(repository, tag)
		if err != nil {
			return nil, err
		}

		svt := semver.ParseTagAsSemVer(tag)

		out = append(out, Tag{tag, svt, digest})
	}

	return out, nil
}
