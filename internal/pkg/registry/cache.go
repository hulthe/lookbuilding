package registry

import (
	"fmt"
	"net/http"

	l "hulthe.net/lookbuilding/internal/pkg/logging"
	"hulthe.net/lookbuilding/internal/pkg/semver"

	"github.com/heroku/docker-registry-client/registry"
	d "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

type tagListReq struct {
	repository string
	responder  chan<- tagListResp
}

type tagListResp struct {
	Data  []Tag
	Error error
}

type digestReq struct {
	repository string
	tag        string
	responder  chan<- digestResp
}

type digestResp struct {
	Data  d.Digest
	Error error
}

type repoCache struct {
	Tags []Tag

	// Map tags to digests
	Digests map[string]d.Digest
}

type cache struct {
	TagListReq chan<- tagListReq
	DigestReq  chan<- digestReq
}

func newCache(registry registry.Registry) cache {
	tagListReq := make(chan tagListReq)
	digestReq := make(chan digestReq)
	cache := cache{
		tagListReq,
		digestReq,
	}

	store := map[string]repoCache{}

	go func() {
		for {
			select {
			case req := <-digestReq:
				repo, isPresent := store[req.repository]
				if !isPresent {
					req.responder <- digestResp{Error: errors.Errorf(
						`repo "%s" not present in cache, can't fetch digest'`, req.repository,
					)}
				}

				digest, isPresent := repo.Digests[req.tag]
				if isPresent {
					req.responder <- digestResp{Data: digest}
				} else {
					url := fmt.Sprintf("%s/v2/%s/manifests/%s", registry.URL, req.repository, req.tag)
					l.Logger.Infof("registry.manifest.head url=%s repository=%s reference=%s", url, req.repository, req.tag)

					httpReq, _ := http.NewRequest("HEAD", url, nil)
					httpReq.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
					resp, err := registry.Client.Do(httpReq)
					if resp != nil {
						defer resp.Body.Close()
					}
					if err != nil {
						req.responder <- digestResp{Error: errors.Wrapf(
							err, "failed to get digest for repo=%s tag=%s", req.repository, req.tag,
						)}
					}
					digest, err := d.Parse(resp.Header.Get("Docker-Content-Digest"))

					/*
						digest, err := registry.ManifestDigest(req.repository, req.tag)
						if err != nil {
							req.responder <- digestResp{Error: errors.Wrapf(
									err, `failed to get digest for repo=%s tag=%s`, req.repository, req.tag,
								)}
						}
					*/

					repo.Digests[req.tag] = digest
					req.responder <- digestResp{Data: digest}
				}

			case req := <-tagListReq:
				repo, isPresent := store[req.repository]

				if isPresent {
					// Tag list was already in cache, just return it
					req.responder <- tagListResp{Data: repo.Tags}

				} else {
					// tag list was not in cache, we have to fetch it
					tagNames, err := registry.Tags(req.repository)

					if err != nil {
						req.responder <- tagListResp{
							Error: errors.Wrapf(err, `failed to list tags for registry repo "%s"`, req.repository),
						}
					}

					// convert names to Tag{}
					var tags []Tag
					for _, tagName := range tagNames {
						tags = append(tags, Tag{
							Name:       tagName,
							SemVer:     semver.ParseTagAsSemVer(tagName),
							repository: req.repository,
							cache:      cache,
						})
					}

					// store result in cache
					store[req.repository] = repoCache{
						Tags:    tags,
						Digests: map[string]d.Digest{},
					}

					req.responder <- tagListResp{
						Data: tags,
					}
				}
			}
		}
	}()

	return cache
}
