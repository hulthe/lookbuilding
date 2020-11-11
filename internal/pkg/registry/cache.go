package registry

import (
	"hulthe.net/lookbuilding/internal/pkg/semver"

	"github.com/heroku/docker-registry-client/registry"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

type tagListReq struct {
	repository string
	responder chan<- tagListResp
}

type tagListResp struct {
	Data []Tag
	Error error
}

type digestReq struct {
	repository string
	tag string
	responder chan<- digestResp
}

type digestResp struct {
	Data digest.Digest
	Error error
}

type repoCache struct {
	Tags []Tag

	// Map tags to digests
	Digests map[string]digest.Digest
}

type cache struct {
	TagListReq chan<- tagListReq
	DigestReq chan<- digestReq
}

func newCache(registry registry.Registry) cache {
	tagListReq := make(chan tagListReq)
	digestReq := make(chan digestReq)

	store := map[string]repoCache{}

	go func() {
		for {
			select {
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
							Name:   tagName,
							SemVer: semver.ParseTagAsSemVer(tagName),
						})
					}

					// store result in cache
					store[req.repository] = repoCache{
						Tags:    tags,
						Digests: map[string]digest.Digest{},
					}

					req.responder <- tagListResp{
						Data: tags,
					}
				}

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
					digest, err := registry.ManifestDigest(req.repository, req.tag)
					if err != nil {
						req.responder <- digestResp{Error: errors.Wrapf(
							err, `failed to get digest for repo=%s tag=%s`, req.repository, req.tag,
						)}
					}

					repo.Digests[req.tag] = digest
					req.responder <- digestResp{Data: digest}
				}
			}
		}
	}()

	return cache {
		tagListReq,
		digestReq,
	}
}