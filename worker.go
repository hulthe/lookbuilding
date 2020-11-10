package main

import (
	"github.com/docker/docker/client"
	"hulthe.net/lookbuilding/internal/pkg/semver"
)

var (
	triggerCh = make(chan struct{})
)

func TriggerScan() {
	triggerCh <- struct{}{}
}

func Worker() {
	Logger.Debugf("background worker starting")

	responseCh := make(chan struct{})

	workerRunning := false
	triggerWaiting := false

	for true {
		select {
		case _ = <-triggerCh:
			if workerRunning {
				triggerWaiting = true
			} else {
				workerRunning = true
				go func() {
					checkAndDoUpdate()
					responseCh <- struct{}{}
				}()
			}
		case _ = <-responseCh:
			if triggerWaiting {
				triggerWaiting = false
				go func() {
					checkAndDoUpdate()
					responseCh <- struct{}{}
				}()
			} else {
				workerRunning = false
			}
		}
	}
}

func checkAndDoUpdate() {
	Logger.Infof("starting scan")

	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	hub, err := anonymousClient()
	if err != nil {
		panic(err)
	}

	labeledContainers := getLabeledContainers(cli)

	Logger.Infof("found %d valid containers", len(labeledContainers))

	for _, lc := range labeledContainers {
		owner, repository, tag := lc.SplitImageParts()
		name := lc.GetName()
		imageName := combineImageParts(owner, repository, nil)

		if tag == nil {
			Logger.Errorf(`no tag specified for container "%s", ignoring`, name)
			continue
		}

		Logger.Infof(`container "%s" image="%s" mode=%s tag="%s"`, name, imageName, lc.Mode.Label(), *tag)

		repoTags, err := getDockerRepoTags(hub, owner, repository)
		if err != nil {
			panic(err)
		}

		Logger.Infof(`tags in registry for "%s": %d`, name, len(repoTags))
		for _, tag := range repoTags {
			svt := semver.ParseTagAsSemVer(tag.Name)
			Logger.Infof(`tag_name="%s" semver=%t digest=%s`, tag.Name, svt != nil, tag.Digest)
		}

		shouldUpdateTo := lc.Mode.ShouldUpdate(*tag, repoTags)
		if shouldUpdateTo != nil {
			Logger.Infof(`updating %s from %s to: %s`, name, *tag, shouldUpdateTo.Name)
			err = lc.UpdateTo(cli, *shouldUpdateTo)
			if err != nil {
				panic(err)
			}
		} else {
			Logger.Infof("no update available for container %s", name)
		}
	}
	Logger.Infof("all done")
}
