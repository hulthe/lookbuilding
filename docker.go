package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

type LabeledContainer struct {
	Container types.Container
	Mode      VersioningMode
}

// Extract the repository owner (if any), repository and tag (if any) from a docker image name
func (lc LabeledContainer) SplitImageParts() (*string, string, *string) {
	name := lc.Container.Image

	var repository string
	var owner *string
	var tag *string

	slashIndex := strings.Index(name, "/")
	if slashIndex >= 0 {
		tmp := name[:slashIndex]
		owner = &tmp
		name = name[slashIndex+1:]
	}

	colonIndex := strings.Index(name, ":")
	if colonIndex >= 0 {
		tmp := name[colonIndex+1:]
		tag = &tmp

		repository = name[:colonIndex]
	} else {
		repository = name
	}

	return owner, repository, tag
}

func (lc LabeledContainer) GetName() string {
	if len(lc.Container.Names) >= 0 {
		// trim prefixed "/"
		return lc.Container.Names[0][1:]
	} else {
		return lc.Container.ID[:10]
	}
}

func combineImageParts(owner *string, repository string, tag *string) string {
	image := repository
	if owner != nil {
		image = fmt.Sprintf("%s/%s", *owner, image)
	}
	if tag != nil {
		image = fmt.Sprintf("%s:%s", image, *tag)
	}

	return image
}

func getLabeledContainers(cli *client.Client) []LabeledContainer {
	out := make([]LabeledContainer, 0)

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	Logger.Infof("scanning running container labels")
	for _, container := range containers {
		Logger.Debugf("checking %s %s", container.ID[:10], container.Image)
		for k, v := range container.Labels {
			Logger.Debugf(`  - "%s": "%s"`, k, v)
			if k == versioningModeLabel {
				mode := ParseVersioningMode(v)
				if mode == nil {
					Logger.Errorf(`Failed to parse "%s" as a versioning mode`, v)
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

func (lc LabeledContainer) UpdateTo(cli *client.Client, tag Tag) error {
	ctx := context.Background()

	owner, repository, _ := lc.SplitImageParts()
	image := combineImageParts(owner, repository, &tag.Name)
	canonicalImage := fmt.Sprintf("docker.io/%s", image)
	Logger.Infof(`pulling image "%s"`, canonicalImage)

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

	Logger.Infof(`renaming container %s`, lc.Container.ID)
	err = cli.ContainerRename(ctx, lc.Container.ID, tmpOldName)
	if err != nil {
		return err
	}

	Logger.Infof("creating new container")
	new, err := cli.ContainerCreate(ctx, oldContainer.Config, hostConfig, &network.NetworkingConfig{
		EndpointsConfig: oldContainer.NetworkSettings.Networks,
	}, name)

	if err != nil {
		return err
	}

	Logger.Infof("starting new container id: %s", new.ID)
	err = cli.ContainerStart(ctx, new.ID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}

	Logger.Infof("removing old container")
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
