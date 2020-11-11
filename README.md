# lookbuilding
_WIP_

Scuffed lookalike of watchtower

This project will, when triggered by an http request, scan the
docker registry and update all properly annotated containers.

Add one of the following labels to your container to have lookbuilding update it.
```sh
# only update to newer images with the same tag
lookbuilding.mode = same_tag

# only update to images with a newer semver version tag
lookbuilding.mode = semver_major

# only update to images with a newer semver version tag
# but with the same major version
lookbuilding.mode = semver_minor

# only update to images with a newer semver version tag
# but with the same major & minor version
lookbuilding.mode = semver_patch
```

#### Environment variables:

- `LOOKBUILDING_ADDR` - set the http bind address, default "0.0.0.0:8000"