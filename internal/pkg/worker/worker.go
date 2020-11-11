package worker

import (
	"hulthe.net/lookbuilding/internal/pkg/docker"
	l "hulthe.net/lookbuilding/internal/pkg/logging"
	"hulthe.net/lookbuilding/internal/pkg/registry"
	"hulthe.net/lookbuilding/internal/pkg/semver"

	"github.com/docker/docker/client"
)

var (
	triggerCh = make(chan struct{})
)

func TriggerScan() {
	triggerCh <- struct{}{}
}

func Worker() {
	l.Logger.Debugf("background worker starting")

	responseCh := make(chan struct{})

	workerRunning := false
	triggerWaiting := false

	for {
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
	l.Logger.Infof("starting scan")

	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	hub, err := registry.AnonymousClient()
	if err != nil {
		panic(err)
	}

	labeledContainers := docker.GetLabeledContainers(cli)

	l.Logger.Infof("found %d valid containers", len(labeledContainers))

	for _, lc := range labeledContainers {
		owner, repository, tag := lc.SplitImageParts()
		name := lc.GetName()
		imageName := docker.CombineImageParts(owner, repository, nil)

		if tag == nil {
			l.Logger.Errorf(`no tag specified for container "%s", ignoring`, name)
			continue
		}

		l.Logger.Infof(`container "%s" image="%s" mode=%s tag="%s"`, name, imageName, lc.Mode.Label(), *tag)

		repoTags, err := hub.GetRepoTags(owner, repository)
		if err != nil {
			panic(err)
		}

		l.Logger.Infof(`tags in registry for "%s": %d`, name, len(repoTags))
		for _, tag := range repoTags {
			svt := semver.ParseTagAsSemVer(tag.Name)
			l.Logger.Infof(`tag_name="%s" semver=%t`, tag.Name, svt != nil)
		}

		shouldUpdateTo := lc.Mode.ShouldUpdate(*tag, repoTags)
		if shouldUpdateTo != nil {
			l.Logger.Infof(`updating %s from %s to: %s`, name, *tag, shouldUpdateTo.Name)

			go func() {
				err = lc.UpdateTo(cli, *shouldUpdateTo)
				if err != nil {
					l.Logger.Error(err)
				}
			}()
		} else {
			l.Logger.Infof("no update available for container %s", name)
		}
	}
	l.Logger.Infof("all done")
}
