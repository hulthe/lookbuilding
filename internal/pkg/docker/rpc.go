package docker

import (
	"context"
	"fmt"

	l "hulthe.net/lookbuilding/internal/pkg/logging"
	"hulthe.net/lookbuilding/internal/pkg/registry"
	"hulthe.net/lookbuilding/internal/pkg/versioning"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

func GetLabeledContainers(cli *client.Client) []LabeledContainer {
	out := make([]LabeledContainer, 0)

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	l.Logger.Infof("scanning running container labels")
	for _, container := range containers {
		l.Logger.Debugf("checking %s %s", container.ID[:10], container.Image)
		for k, v := range container.Labels {
			l.Logger.Debugf(`  - "%s": "%s"`, k, v)
			if k == versioning.ModeLabel {
				mode := versioning.ParseMode(v)
				if mode == nil {
					l.Logger.Errorf(`Failed to parse "%s" as a versioning mode`, v)
					continue
				}

				lc := LabeledContainer{
					container,
					*mode,
				}

				out = append(out, lc)
				continue
			}
		}
	}

	return out
}

func (lc LabeledContainer) UpdateTo(cli *client.Client, tag registry.Tag) error {
	ctx := context.Background()

	owner, repository, _ := lc.SplitImageParts()
	image := CombineImageParts(owner, repository, &tag.Name)
	canonicalImage := fmt.Sprintf("docker.io/%s", image)
	l.Logger.Infof(`pulling image "%s"`, canonicalImage)

	//containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	imageReader, err := cli.ImagePull(ctx, canonicalImage, types.ImagePullOptions{})
	if err != nil {
		return err
	}

	defer imageReader.Close()

	loadResponse, err := cli.ImageLoad(ctx, imageReader, false)
	if err != nil {
		return err
	}

	defer loadResponse.Body.Close()

	fmt.Printf("Stopping container %s\n", lc.Container.ID)
	err = cli.ContainerStop(ctx, lc.Container.ID, nil)
	if err != nil {
		return err
	}

	oldContainer, err := cli.ContainerInspect(ctx, lc.Container.ID)
	if err != nil {
		return err
	}

	name := oldContainer.Name
	tmpOldName := fmt.Sprintf("%s.lb.old", name)

	config := oldContainer.Config
	config.Image = image

	hostConfig := oldContainer.HostConfig
	hostConfig.VolumesFrom = []string{tmpOldName}

	l.Logger.Infof(`renaming container %s`, lc.Container.ID)
	err = cli.ContainerRename(ctx, lc.Container.ID, tmpOldName)
	if err != nil {
		return err
	}

	l.Logger.Infof("creating new container")
	new, err := cli.ContainerCreate(ctx, oldContainer.Config, hostConfig, &network.NetworkingConfig{
		EndpointsConfig: oldContainer.NetworkSettings.Networks,
	}, name)

	if err != nil {
		return err
	}

	l.Logger.Infof("starting new container id: %s", new.ID)
	err = cli.ContainerStart(ctx, new.ID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}

	l.Logger.Infof("removing old container")
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

