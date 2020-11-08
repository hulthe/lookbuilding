package main

import (
	"fmt"
	"net/http"

	"github.com/docker/docker/client"
)

func checkForUpdates() {
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	hub, err := anonymousClient()
	if err != nil {
		panic(err)
	}

	labeledContainers := getLabeledContainers(cli)
	fmt.Println()

	for _, lc := range labeledContainers {
		owner, repository, tag := lc.SplitImageParts()

		fmt.Printf("container image: %s\n", combineImageParts(owner, repository, nil))
		fmt.Printf("  versioning: %+v\n", lc.Mode.Label())
		fmt.Printf("  id: %s\n", lc.Container.ImageID)

		if tag != nil {
			fmt.Printf("  current tag: %s\n", *tag)
		} else {
			fmt.Printf("  no current tag, skipping\n")
			continue
		}

		repoTags, err := getDockerRepoTags(hub, owner, repository)
		if err != nil {
			panic(err)
		}

		fmt.Println("  tags in registry:")
		for _, tag := range repoTags {
			fmt.Printf("  - \"%s\" %s\n", tag.Name, tag.Digest)
			svt := parseTagAsSemVer(tag.Name)
			if svt != nil {
				fmt.Printf("    semver: true\n")
			}
		}

		shouldUpdateTo := lc.Mode.ShouldUpdate(*tag, repoTags)
		if shouldUpdateTo != nil {
			fmt.Printf("  updating to: %s\n", shouldUpdateTo.Name)
			err = lc.UpdateTo(cli, *shouldUpdateTo)
			if err != nil {
				panic(err)
			}
		} else {
			fmt.Println("  no update available")
		}
	}
	fmt.Println("all done")
}

func main() {
	addr := "0.0.0.0:8000"

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		go checkForUpdates()
		fmt.Fprintf(w, "OK")
	})

	fs := http.FileServer(http.Dir("static/"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	fmt.Printf("Listening on %s\n", addr)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		panic(err)
	}
}
