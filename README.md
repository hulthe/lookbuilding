# lookbuilding

Scuffed lookalike of watchtower

This project will, when triggered by an http request,
scan and update all properly annotated containers.

Add one of the following labels to your container to have lookbuilding update it.
```
lookbuilding.mode = same_tag
lookbuilding.mode = semver_major
lookbuilding.mode = semver_minor
lookbuilding.mode = semver_patch
```

#### Environment variables:

- `LOOKBUILDING_ADDR` - set the http bind address, default "0.0.0.0:8000"