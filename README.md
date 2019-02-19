[![CircleCI](https://circleci.com/gh/flant/multiwerf/tree/master.svg?style=svg)](https://circleci.com/gh/flant/multiwerf/tree/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/flant/multiwerf)](https://goreportcard.com/report/github.com/flant/multiwerf)
[![Download from Github](https://img.shields.io/github/tag-date/flant/multiwerf.svg?logo=github&label=latest)](https://github.com/flant/multiwerf/releases/latest)
[![Download from Bintray mirror](https://api.bintray.com/packages/flant/multiwerf/multiwerf/images/download.svg)](https://bintray.com/flant/multiwerf/multiwerf/_latestVersion)

# multiwerf
Self-updatable version manager of [werf](https://github.com/flant/werf) binaries with awareness of release channels.

### Contents

- [Quick start](#quick-start)
- [Commands](#commands)
- [Werf versioning](#werf-versioning)
- [Installation and self-update](#installation-and-self-update)
- [License](#license)


## Quick start
 
### Install

The simplest way is to get latest version of multiwerf to current directory with get.sh script:

```
curl -L https://raw.githubusercontent.com/flant/multiwerf/master/get.sh | bash
```

Also you can manually download a binary for your platform from [github releases](https://github.com/flant/multiwerf/releases) or from [bintray mirror](https://bintray.com/flant/multiwerf/multiwerf/_latestVersion).

It is recommended to install multiwerf with enabled self updates as described in [Installation and update](#installation-and-update).

### Usage

General usage of `multiwerf` is to download a werf binary and setup a `werf` function for the shell.

```
$ source <(multiwerf use 1.0 alpha)
Detect version v1.0.0-alpha.17 as latest for channel 1.0/alpha
werf 1.0/alpha updated to v1.0.0-alpha.17
$ werf version
v1.0.0-alpha.17
```

This command will download the latest version of `werf` from `1.0/alpha` channel into ~/.multiwerf/<version> directory and setup a shell function to run this version.

More on compatibility of werf channels in [werf README](https://github.com/flant/werf#backward-compatibility-promise).


## Commands

- `multiwerf use MAJOR.MINOR CHANNEL` — check for latest version in channel in MAJOR.MINOR branch and return a script for use with `source`
- `multiwerf update MAJOR.MINOR CHANNEL` — update binary to the latest version of channel in MAJOR.MINOR branch

First positional argument is in form of MAJOR.MINOR. CHANNEL is one of: alpha, beta, rc, ea, stable. More on this in [werf versioning](#werf-versioning).

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

`use` command also have `--update=no` flag to prevent version checking and use only locally available versions from ~/.multiwerf.

`use` and `update` commands are check for latest version of multiwerf and self update a multiwerf binary if needed. This can be disabled with `--self-update=no` flag. 


## Werf versioning

Werf binary releases are follow a [Semantic Versioning](https://semver.org/) and [Backward Compatibility Promise](https://github.com/flant/werf#backward-compatibility-promise), so `multiwerf` makes this assumptions:

- each werf release version has a form of `MAJOR.MINOR.PATCH-PRERELEASE+METADATA`
- PATCH can be increased directly (1.0.1 → 1.0.2)
- PATCH can be increased with prereleases (1.0.1 → 1.0.2-alpha → 1.0.2-alpha.1 → 1.0.2-rc.1 → 1.0.2)
- prefix of a PRERELEASE determines a CHANNEL
- version without PRERELEASE part is a version for `stable` channel
- version with PRERELEASE part should be a version for `alpha`, `beta`, `rc` or `ea` channels
- versions from unknown channels are ignored
- METADATA parts are not sorted (by semver spec)

### Channels

#### stable

`stable` can be ommited: `multiwerf use 1.1 stable` or `multiwerf use 1.1` is equivalent commands.

`multiwerf` will check for the latest PATCH for passed MAJOR.MINOR.

#### alpha, beta, rc, ea

`multiwerf use 1.0 alpha`, `multiwerf update 1.1 rc`

`multiwerf` will check for the latest version from requested channel. If there is version from equal or more stable channel then this version will be used.
If requested channel is `beta`, but `ea` is available, then binary will be updated from `ea`.

For example, let assume that repository contains these versions:

```
2.0.2
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
This command will download and run version `2.1.2` — the latest available patch release from `stable` channel for 2.1 minor branch

```
multiwerf use 2.1 alpha
multiwerf use 2.1 rc
```
These commands will ignore 2.1.1-alpha and 2.1.1-rc patch releases and download more stable version `2.1.2`.

```
multiwerf use 3.0 alpha
```
This command will download `3.0.1-beta.2` because there is availabe release in more stable channel: beta.

```
multiwerf use 3.0 rc
```
This command will download `3.0.0` because the latest patch release 3.0.1 have no versions from rc channel.

```
multiwerf use 2.0
```
This command will download `2.0.12+build.2018.12` because metadata is ignored.


## Installation and self-update

Before checking for new versions of werf in use and update commands, multiwerf checks for self new versions. If new version is available in bintray.com/flant/multiwerf/multiwerf, multiwerf download it and  start a new proccess with the same arguments and environment as current.

`--self-update=no` flag and `MULTIWERF_SELF_UPDATE=no` environment variable are available to turn off self updates.

Self update is disabled if `multiwerf` binary isn't owned by user that run it and if file is not writable by owner.

There are 2 recommended ways to install multiwerf:

1. Put multiwerf into $HOME/bin directory. This is a best scenario for gitlab-runner setup or for local development. In this case multiwerf will check for new version no more than every day and new versions of werf will be checked no more than every hour.
2. Put multiwerf into /usr/local/bin directory and set root as owner. This setup requires to define a cronjob for user root with command `multiwerf update 1.0`. In this case users cannot update multiwerf but self-update is working.

### Running multiwerf in CI

If multiwerf is used in shell scenarios without possibility to enter custom commands after execution, for example, in CI environments,
then `source` with `Process Substitution` can lead to errors. The recommendation is to use `type` to ensure that multiwerf
is exists and is executable:

```
type multiwerf && source <(multiwerf use 1.0 alpha)
```

This command will print a message to stderr in case if multiwerf is not found, so diagnostic in CI environment should be simple. 

### Offline tips

`multiwerf` can be used in offline scenarios.

1. set MULTIWERF_UPDATE=no and MULTIWERF_SELF_UPDATE=no environment variables to prevent http requests or use `--self-update=no --update=no` flags
2. put desired binary file and SHA256SUMS file into `~/.multiwerf/<version> directory`
3. `source <(multiwerf use ...)` will not make any online request and consider locally available version as latest


## License

Apache License 2.0, see [LICENSE](LICENSE).
