[![CircleCI](https://circleci.com/gh/flant/multiwerf/tree/master.svg?style=svg)](https://circleci.com/gh/flant/multiwerf/tree/master)
[![Download](https://api.bintray.com/packages/flant/multiwerf/multiwerf/images/download.svg)](https://bintray.com/flant/multiwerf/multiwerf/_latestVersion)

# multiwerf
Tool to (auto)update a binary `werf` release to the latest available version

## Commands

- `multiwerf use MAJOR.MINOR CHANNEL` — check for latest version of multiwerf, self update in background if needed, check for latest version in MAJOR.MINOR series and return a script for use with `source`
- `multiwerf update MAJOR.MINOR CHANNEL` — update binary to the latest version in MAJOR.MINOR series

First positional argument is in form of MAJOR.MINOR. More on this in Versioning.

CHANNEL is one of: alpha, beta, rc, stable

Binaries are downloaded to a directory `$HOME/.multiwerf/VERSION/`. For example, version `1.0.1-alpha.3` of `werf` binaries for user `gitlab-runner` will be stored as

```
/home/gitlab-runner/.multiwerf
|-- 1.0.1-alpha.3
|   |-- SHA256SUMS
|   |-- SHA256SUMS.sig
|   `-- werf-linux-amd64-1.0.1-alpha.3
|
...
```

## Versioning

Binary releases should follow a [semver](https://semver.org/) versioning, so version has this form:

```
MAJOR.MINOR.PATCH-PRERELEASE+METADATA
```

`multiwerf` makes this assumptions:

- PRERELEASE determines a CHANNEL
- PATCH can be increased directly (1.0.1 → 1.0.2)
- PATCH can be increased with prereleases (1.0.1 → 1.0.2-alpha → 1.0.2-alpha.1 → 1.0.2-rc.1 → 1.0.2)
- version without PRERELEASE part is a version for `stable` channel
- version with PRERELEASE part should be a version for `alpha`, `beta` or `rc` channels
- METADATA parts are not sorted (by semver spec)

## Channels

### stable

`stable` can be ommited: `multiwerf use 1.1 stable` or `multiwerf use 1.1` is equivalent commands.

`multiwerf` will check for the latest PATCH for passed MAJOR.MINOR.

### alpha, beta, rc

`multiwerf use 1.0 alpha`, `multiwerf use 1.1 rc`

`multiwerf` will check for the latest prerelease version. If there is version with equal or more stable prerelease then it will be used.
If the latest version is stable then binary will be updated to stable.

For example, let assume that repository contains these versions:

```
2.0.2
2.0.12+build.2018.11
2.0.12+build.2018.12
2.1.0
2.1.1-alpha.1
2.1.1-rc.1
2.1.2
3.0.0
3.0.1-alpha.1
3.0.1-alpha.1
3.0.1-beta.2
```

```
multiwerf use 2.1
```
This command will download and run version `2.1.2` — the latest available stable for MAJOR=2, MINOR=1

```
multiwerf use 2.1 alpha
multiwerf use 2.1 rc
```
These commands will ignore 2.1.1-alpha and 2.1.1-rc prereleases and download version `2.1.2` because the latest version for 2.1 is 2.1.2.

```
multiwerf use 3.0 alpha
```

This command will download `3.0.1-beta.2` because there is availabe release in more stable channel: beta.

```
multiwerf use 3.0 rc
```
This command will download `3.0.0` because the latest PATCH 3.0.1 have no versions with rc prerelease.

```
multiwerf use 2.0
```
This command will download `2.0.12+build.2018.12` because equal MAJOR.MINOR.PATCH are are sorted by metadata.

## Download settings

`multiwerf` depends on some external information:

- version list
- an url of repository
- a directories structure of the repository

The first version of `multiwerf` hardcode this information at complile time and support only bintray API and download url.

## Some thoughts on release cycle

- `stable` channel is used for most critical environments with tight SLA
- `rc` for environments with normal SLA
- `beta` for environments where downtime is acceptable, i.e., dev, test, some kind of stages
- `alpha` for bleeding edge environments to give a try for fixes and new features

## Installation and update

TODO self-update

TODO ...

Download to home directory, run...
