package docker

import (
	"context"
	"fmt"
	"strings"

	l "hulthe.net/lookbuilding/internal/pkg/logging"
	"hulthe.net/lookbuilding/internal/pkg/registry"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	d "github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

func GetLabeledContainers(cli *client.Client) ([]LabeledContainer, error) {
	out := make([]LabeledContainer, 0)

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	l.Logger.Infof("scanning running container labels")
	for _, container := range containers {
		lc := LabeledContainer{Container: container}

		if container.Image == container.ImageID {
			l.Logger.Errorf("ignoring container %s which has an untagged image", lc.GetName())
			continue
		}

		l.Logger.Debugf("checking %s %s", lc.GetName(), container.Image)
		for k, v := range container.Labels {
			l.Logger.Debugf(`  - "%s": "%s"`, k, v)

			if k == VersioningModeLabel {
				mode := ParseMode(v)
				if mode == nil {
					l.Logger.Errorf(`failed to parse "%s" as a versioning mode`, v)
					continue
				}
				lc.Mode = *mode

				inspect, _, err := cli.ImageInspectWithRaw(context.Background(), container.ImageID)
				if err != nil {
					errors.Wrapf(err, "failed to inspect container %s", lc.GetName())
				}

				if len(inspect.RepoDigests) >= 2 {
					// TODO: find out if having more than one digest could break same_tag version mode
					l.Logger.Warnf("unexpected: container %s had more than one RepoDigest", lc.GetName())
				} else if len(inspect.RepoDigests) == 0 {
					return nil, errors.Errorf("unexpected: container %s has no RepoDigests", lc.GetName())
				}

				imageDigest := inspect.RepoDigests[0]
				atIndex := strings.Index(imageDigest, "@")
				lc.ImageDigest, err = d.Parse(imageDigest[atIndex+1:])
				if err != nil {
					return nil, errors.Wrapf(err, "failed to parse image digest of running container %s", lc.GetName())
				}

				out = append(out, lc)
				continue
			}
		}
	}

	return out, nil
}

func (lc LabeledContainer) UpdateTo(cli *client.Client, tag registry.Tag) error {
	ctx := context.Background()

	owner, repository, _ := lc.SplitImageParts()
	image := CombineImageParts(owner, repository, &tag.Name)
	canonicalImage := fmt.Sprintf("docker.io/%s", image)
	l.Logger.Infof(`pulling image "%s"`, canonicalImage)

	imageReader, err := cli.ImagePull(ctx, canonicalImage, types.ImagePullOptions{})
	if err != nil {
		return errors.Wrapf(err, `failed to pull image "%s"`, canonicalImage)
	}

	loadResponse, err := cli.ImageLoad(ctx, imageReader, false)
	if err != nil {
		return errors.Wrapf(err, `failed to load pulled image "%s"`, canonicalImage)
	}

	err = loadResponse.Body.Close()
	if err != nil {
		return errors.Wrapf(err, `failed to close rpc response when loading image "%s"`, canonicalImage)
	}
	err = imageReader.Close()
	if err != nil {
		return errors.Wrapf(err, `failed to close rpc response when pulling image "%s"`, canonicalImage)
	}

	l.Logger.Infof("stopping container %s", lc.GetName())
	err = cli.ContainerStop(ctx, lc.Container.ID, nil)
	if err != nil {
		return errors.Wrapf(err, `failed to stop container "%s"`, lc.GetName())
	}

	oldContainer, err := cli.ContainerInspect(ctx, lc.Container.ID)
	if err != nil {
		return errors.Wrapf(err, `failed to inspect container "%s"`, lc.GetName())
	}

	oldTmpName := fmt.Sprintf("%s.lb.old", lc.GetName())

	config := *oldContainer.Config
	config.Image = image

	hostConfig := *oldContainer.HostConfig
	hostConfig.VolumesFrom = []string{oldTmpName}

	l.Logger.Infof(`renaming container %s to %s`, lc.GetName(), oldTmpName)
	err = cli.ContainerRename(ctx, lc.Container.ID, oldTmpName)
	if err != nil {
		return errors.Wrapf(err, `failed to rename container "%s" to "%s"`, lc.GetName(), oldTmpName)
	}

	l.Logger.Infof("creating new container %s", lc.GetName())
	new, err := cli.ContainerCreate(ctx, &config, &hostConfig, &network.NetworkingConfig{
		EndpointsConfig: oldContainer.NetworkSettings.Networks,
	}, lc.GetName())

	if err != nil {
		return errors.Wrapf(err, `failed to create container for new version of %s`, image)
	}

	l.Logger.Infof("starting new container %s with id %s", lc.GetName(), new.ID)
	err = cli.ContainerStart(ctx, new.ID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}

	l.Logger.Infof("removing old container %s", oldTmpName)
	err = cli.ContainerRemove(ctx, oldContainer.ID, types.ContainerRemoveOptions{
		RemoveVolumes: false,
		RemoveLinks:   false,
		Force:         false,
	})
	if err != nil {
		return err
	}

	return nil
}
