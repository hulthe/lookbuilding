package main

import (
	"fmt"
	"github.com/heroku/docker-registry-client/registry"
	d "github.com/opencontainers/go-digest"
	//"github.com/docker/distribution/digest"
	//"github.com/docker/distribution/manifest"
	//"github.com/docker/libtrust"
)

type Tag struct {
	Name   string
	SemVer *SemVerTag
	Digest d.Digest
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

func getDockerRepoTags(hub *registry.Registry, maybe_owner *string, repository string) ([]Tag, error) {
	if maybe_owner != nil {
		repository = fmt.Sprintf("%s/%s", *maybe_owner, repository)
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

		svt := parseTagAsSemVer(tag)

		out = append(out, Tag{tag, svt, digest})
	}

	return out, nil
}
