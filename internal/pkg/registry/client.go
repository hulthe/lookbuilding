package registry

import (
	"fmt"

	l "hulthe.net/lookbuilding/internal/pkg/logging"
	"hulthe.net/lookbuilding/internal/pkg/semver"

	"github.com/heroku/docker-registry-client/registry"
	"github.com/opencontainers/go-digest"
)

type Tag struct {
	Name       string
	SemVer     *semver.Tag
	repository string
	cache      cache
}

type Client struct {
	cache cache
}

func (tag Tag) GetDigest() (digest.Digest, error) {
	responseCh := make(chan digestResp)
	tag.cache.DigestReq <- digestReq{tag.repository, tag.Name, responseCh}
	resp := <-responseCh
	return resp.Data, resp.Error
}

func (client Client) GetRepoTags(maybeOwner *string, repository string) ([]Tag, error) {
	if maybeOwner != nil {
		repository = fmt.Sprintf("%s/%s", *maybeOwner, repository)
	}

	responseCh := make(chan tagListResp)
	client.cache.TagListReq <- tagListReq{repository, responseCh}
	resp := <-responseCh
	return resp.Data, resp.Error
}

func AnonymousClient() (*Client, error) {
	url := "https://registry-1.docker.io/"
	username := "" // anonymous
	password := "" // anonymous

	registry, err := registry.New(url, username, password)
	if err != nil {
		return nil, err
	}

	registry.Logf = l.Logger.Infof

	client := Client{
		cache: newCache(*registry),
	}

	return &client, nil
}
