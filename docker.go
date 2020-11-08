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

	fmt.Println("scanning running container labels")
	for _, container := range containers {
		fmt.Printf("- %s %s\n", container.ID[:10], container.Image)
		for k, v := range container.Labels {
			fmt.Printf("  - \"%s\": \"%s\"\n", k, v)
			if k == versioningModeLabel {
				mode := ParseVersioningMode(v)
				if mode == nil {
					fmt.Printf("Failed to parse '%s' as a versioning mode\n", v)
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
	fmt.Printf("Pulling image \"%s\"\n", canonicalImage)

	//containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	rc, err := cli.ImagePull(ctx, canonicalImage, types.ImagePullOptions{})
	if err != nil {
		return err
	}

	// TODO: does it still pull the image if i just close the reader?
	err = rc.Close()
	if err != nil {
		return err
	}

	fmt.Printf("Stopping container %s\n", lc.Container.ID)
	// TODO: hopefully this is blocking
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

	fmt.Printf("Renaming container %s\n", lc.Container.ID)
	err = cli.ContainerRename(ctx, lc.Container.ID, tmpOldName)
	if err != nil {
		return err
	}

	fmt.Printf("Creating new container\n")
	new, err := cli.ContainerCreate(ctx, oldContainer.Config, hostConfig, &network.NetworkingConfig{
		EndpointsConfig: oldContainer.NetworkSettings.Networks,
	}, name)

	if err != nil {
		return err
	}

	fmt.Printf("Starting new container id: %s\n", new.ID)
	err = cli.ContainerStart(ctx, new.ID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}

	fmt.Printf("Removing old container\n")
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