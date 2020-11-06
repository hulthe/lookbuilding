package main

import (
	"context"
	"encoding/json"
	"net/http"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const (
	versioningModeLabel = "lookbuilding.mode"
)

type VersioningMode string

const (
	SemVerAny VersioningMode = "SemVerAny"
)

type labeledContainer struct {
	container types.Container
	mode      VersioningMode
}

type tag struct {
	Creator             int     `json:"creator"`
	ID                  int     `json:"id"`
	LastUpdated         string  `json:"last_updated"`
	LastUpdater         int     `json:"lastUpdater"`
	LastUpdaterUsername string  `json:"lastUpdaterUsername"`
	Name                string  `json:"name"`
	Repository          int     `json:"repository"`
	FullSize            int     `json:"full_size"`
	V2                  bool    `json:"v2"`
	TagStatus           string  `json:"tag_status"`
	TagLastPulled       string  `json:"tag_last_pulled"`
	TagLastPushed       string  `json:"tag_last_pushed"`
	Images              []image `json:images`
}

type image struct {
	Architecture string `json:architecture`
	Features     string `json:features`
	Digest       string `json:digest`
	OS           string `json:linux`
	OSFeatures   string `json:os_features`
	Size         int    `json:size`
	Status       string `json:status`
	LastPulled   string `json:last_pulled`
	LastPushed   string `json:last_pushed`
	//"variant":null,
	//"os_version":null,
}

// Extract the repository owner (if any), repository and tag (if any) from a docker image name
func getImageParts(name string) (*string, string, *string) {
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

func getDockerRepoTags(maybe_owner *string, repository string) []tag {
	type dockerPollResponse struct {
		Count   int   `json:"count"`
		Results []tag `json:"results"`
	}

	owner := "_"
	if maybe_owner != nil {
		owner = *maybe_owner
	}

	url := fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/%s/tags", owner, repository)
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	var data dockerPollResponse
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		panic(err)
	}

	return data.Results
}

func getLabeledContainers(cli *client.Client) []labeledContainer {
	out := make([]labeledContainer, 0)

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		fmt.Printf("%s %s\n", container.ID[:10], container.Image)
		for k, v := range container.Labels {
			if k == versioningModeLabel {
				lc := labeledContainer{
					container,
					SemVerAny,
				}

				out = append(out, lc)
				continue;
			}
			fmt.Printf("  - %s: %s\n", k, v)
		}
	}

	return out
}

func main() {
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	labeledContainers := getLabeledContainers(cli)
	for _, lc := range labeledContainers {
		owner, repository, tag := getImageParts(lc.container.Image)

		if owner != nil {
			fmt.Printf("container image: %s/%s\n", *owner, repository)
		} else {
			fmt.Printf("container image: _/%s\n", repository)
		}

		fmt.Printf("  id: %s\n", lc.container.ImageID)

		if tag != nil {
			fmt.Printf("  tag: %s\n", *tag)
		} else {
			fmt.Printf("  no tag\n")
		}

		repoTags := getDockerRepoTags(owner, repository)

		fmt.Println("## tags in registry ##")
		for _, tag := range repoTags {
			fmt.Printf("  tag: %s\n", tag.Name)
		}
		fmt.Println("##")
	}
}
