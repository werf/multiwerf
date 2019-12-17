[![CircleCI](https://circleci.com/gh/flant/multiwerf/tree/master.svg?style=svg)](https://circleci.com/gh/flant/multiwerf/tree/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/flant/multiwerf)](https://goreportcard.com/report/github.com/flant/multiwerf)
[![Download from Github](https://img.shields.io/github/tag-date/flant/multiwerf.svg?logo=github&label=latest)](https://github.com/flant/multiwerf/releases/latest)
[![Download from Bintray mirror](https://api.bintray.com/packages/flant/multiwerf/multiwerf/images/download.svg)](https://bintray.com/flant/multiwerf/multiwerf/_latestVersion)

**multiwerf** is a self-updatable [werf](https://github.com/flant/werf) manager with the awareness of release channels, allowed stability levels. multiwerf follows werf [Backward Compatibility Promise](https://github.com/flant/werf#backward-compatibility-promise).

General usage of multiwerf is managing werf binaries and providing the actual binary for `MAJOR.MINOR` version and `CHANNEL` in the shell session.

## Contents

- [Installation](#installation)
- [Common Usage](#common-usage)
- [Commands](#commands)
- [Self-update](#self-update)
- [Offline Usage](#offline-usage)
- [License](#license)

## Installation

### Unix shell (sh, bash, zsh)

```bash
# add ~/bin into PATH
export PATH=$PATH:$HOME/bin
echo 'export PATH=$PATH:$HOME/bin' >> ~/.bashrc

# install multiwerf into ~/bin directory
mkdir -p ~/bin
cd ~/bin
curl -L https://raw.githubusercontent.com/flant/multiwerf/master/get.sh | bash
```

### Windows

Choose a release from [GitHub releases](https://github.com/flant/multiwerf/releases) or [bintray mirror](https://bintray.com/flant/multiwerf/multiwerf/_latestVersion) and use one of the following approaches with the chosen binary URL.  

#### PowerShell

```shell
$MULTIWERF_BIN_PATH = "C:\ProgramData\multiwerf\bin"
mkdir $MULTIWERF_BIN_PATH

Invoke-WebRequest -Uri https://flant.bintray.com/multiwerf/v1.0.16/multiwerf-windows-amd64-v1.0.16.exe -OutFile $MULTIWERF_BIN_PATH\multiwerf.exe

[Environment]::SetEnvironmentVariable(
    "Path",
    [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::Machine) + "$MULTIWERF_BIN_PATH",
    [EnvironmentVariableTarget]::Machine)

$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
```

#### cmd.exe (run as Administrator)

```shell
set MULTIWERF_BIN_PATH="C:\ProgramData\multiwerf\bin"
mkdir %MULTIWERF_BIN_PATH%
bitsadmin.exe /transfer "multiwerf" https://flant.bintray.com/multiwerf/v1.0.16/multiwerf-windows-amd64-v1.0.16.exe %MULTIWERF_BIN_PATH%\multiwerf.exe
setx /M PATH "%PATH%;%MULTIWERF_BIN_PATH%"

# after that open new cmd.exe session and start using multiwerf
```

## Common Usage

### Unix shell (sh, bash, zsh)

#### Add werf alias to the current shell session

```bash
. $(multiwerf use 1.0 stable --as-file)
```

#### Run command on terminal startup

```bash
echo '. $(multiwerf use 1.0 stable --as-file)' >> ~/.bashrc
```

#### CI usage tip

`source` with `Process Substitution` can lead to errors If multiwerf is used in shell scenarios without possibility to enter custom commands after execution, for example, in CI environments. The recommendation is to use `type` to ensure that multiwerf
is exist and executable:

```shell
type multiwerf && . $(multiwerf use 1.0 stable --as-file)
```

This command will print a message to stderr in case if multiwerf is not found, so diagnostic in CI environment should be simple. 

### Windows

#### PowerShell

##### Add werf alias to the current shell session

```shell
Invoke-Expression -Command "multiwerf use 1.0 stable --as-file --shell powershell" | Out-String -OutVariable WERF_USE_SCRIPT_PATH
. $WERF_USE_SCRIPT_PATH.Trim()
```

#### cmd.exe

##### Add werf alias to the current shell session

```shell
FOR /F "tokens=*" %g IN ('multiwerf use 1.0 stable --as-file --shell cmdexe') do (SET WERF_USE_SCRIPT_PATH=%g)
%WERF_USE_SCRIPT_PATH%
```

## Commands

- `multiwerf update <MAJOR.MINOR> [<CHANNEL>]`: Perform self-update and download the actual werf binary.

- `multiwerf use <MAJOR.MINOR> [<CHANNEL>]`: Print the script that should be sourced to use the actual werf binary in the current shell session.

- `multiwerf werf-path <MAJOR.MINOR> [<CHANNEL>]`: Print the actual werf binary path (based on local werf binaries).

- `multiwerf werf-exec <MAJOR.MINOR> [<CHANNEL>] [<WERF_ARGS>...]`: Exec the actual werf binary (based on local werf binaries).

The first positional argument is the version in the form of `MAJOR.MINOR`. `CHANNEL` is one of the following channels: alpha, beta, ea, stable, rock-solid. More on this in [werf versioning](#werf-versioning).

multiwerf downloads binaries to a directory `$HOME/.multiwerf/VERSION/`. For example, the werf version `1.0.1-ea.3` for user `gitlab-runner` will be stored as:

```
/home/gitlab-runner/.multiwerf
|-- 1.0.1-ea.3
|   |-- SHA256SUMS
|   |-- SHA256SUMS.sig
|   `-- werf-linux-amd64-1.0.1-ea.3
|
...
```

> `multiwerf use` command also has `--update=no` flag to prevent version checking and use only locally available versions from ~/.multiwerf.

> `multiwerf use` and `multiwerf update` commands check for the latest version of multiwerf and perform self-update if it is needed. This can be disabled with `--self-update=no` flag. 

## Self-update

Before checking for new versions of werf in use and update commands multiwerf checks for self new versions. If a new version is available in `bintray.com/flant/multiwerf/multiwerf` multiwerf downloads it and starts a new process with the same arguments and environment as current.

`--self-update=no` flag and `MULTIWERF_SELF_UPDATE=no` environment variable are available to turn off self updates.

Self-update is disabled if `multiwerf` binary isn't owned by user that runs it and if file is not writable by owner.

There are 2 recommended ways to install multiwerf:

1. Put multiwerf into `$HOME/bin` directory. This is a best scenario for gitlab-runner setup or for local development. In this case multiwerf will check for new version no more than every day and new versions of werf will be checked no more than every hour.
2. Put multiwerf into `/usr/local/bin` directory and set root as owner. This setup requires to define a cronjob for user root with command `multiwerf update 1.0`. In this case users cannot update multiwerf but self-update is working.

### Update Delays

Checking for the latest multiwerf and werf versions are delayed to prevent excessive traffic.

Self-update is delayed to check for new multiwerf version not earlier than 24 hours after the last check for `use` and `update` command.

werf updates are delayed to check for the latest version not earlier than 1 hour after the last check for `use` command. 

## License

Apache License 2.0, see [LICENSE](LICENSE)
