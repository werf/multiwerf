[![Go Report Card](https://goreportcard.com/badge/github.com/werf/multiwerf)](https://goreportcard.com/report/github.com/werf/multiwerf)
[![Test coverage](https://api.codeclimate.com/v1/badges/fab032c3d6836d768af4/test_coverage)](https://codeclimate.com/github/werf/multiwerf/test_coverage)
[![Download from Github](https://img.shields.io/github/tag-date/werf/multiwerf.svg?logo=github&label=latest)](https://github.com/werf/multiwerf/releases/latest)

# ⛔️ DEPRECATED! This project repo is no longer being maintained. Use the new update manager by following [the instructions on the werf site](https://werf.io/installation.html). 

**multiwerf** is a self-updatable [werf](https://github.com/werf/werf) manager with the awareness of release channels, allowed stability levels. multiwerf follows werf [Backward Compatibility Promise](https://github.com/werf/werf#backward-compatibility-promise).

General usage of multiwerf is managing werf binaries and providing the actual binary for `MAJOR.MINOR` version and `CHANNEL` in the shell session.

## Contents

- [Installation](#installation)
- [Common Usage](#common-usage)
- [Commands](#commands)
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
curl -L https://raw.githubusercontent.com/werf/multiwerf/master/get.sh | bash
```

### Windows

Choose a release from [GitHub releases](https://github.com/werf/multiwerf/releases) and use one of the following approaches with the chosen binary URL.

#### PowerShell

```shell
$MULTIWERF_BIN_PATH = "C:\ProgramData\multiwerf\bin"
mkdir $MULTIWERF_BIN_PATH

Invoke-WebRequest -Uri https://storage.yandexcloud.net/multiwerf/targets/releases/v1.4.7/multiwerf-windows-amd64-v1.4.7.exe -OutFile $MULTIWERF_BIN_PATH\multiwerf.exe

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
bitsadmin.exe /transfer "multiwerf" https://storage.yandexcloud.net/multiwerf/targets/releases/v1.4.7/multiwerf-windows-amd64-v1.4.7.exe %MULTIWERF_BIN_PATH%\multiwerf.exe
setx /M PATH "%PATH%;%MULTIWERF_BIN_PATH%"

# after that open new cmd.exe session and start using multiwerf
```

## Common Usage

### Unix shell (sh, bash, zsh)

#### Add werf alias to the current shell session

```bash
. $(multiwerf use 1.1 stable --as-file)
```

#### Run command on terminal startup

```bash
echo '. $(multiwerf use 1.1 stable --as-file)' >> ~/.bashrc
```

#### CI usage tip

`source` with `Process Substitution` can lead to errors If multiwerf is used in shell scenarios without possibility to enter custom commands after execution, for example, in CI environments. The recommendation is to use `type` to ensure that multiwerf
is exist and executable:

```shell
type multiwerf && . $(multiwerf use 1.1 stable --as-file)
```

This command will print a message to stderr in case if multiwerf is not found, so diagnostic in CI environment should be simple. 

### Windows

#### PowerShell

##### Add werf alias to the current shell session

```shell
Invoke-Expression -Command "multiwerf use 1.1 stable --as-file --shell powershell" | Out-String -OutVariable WERF_USE_SCRIPT_PATH
. $WERF_USE_SCRIPT_PATH.Trim()
```

#### cmd.exe

##### Add werf alias to the current shell session

```shell
FOR /F "tokens=*" %g IN ('multiwerf use 1.1 stable --as-file --shell cmdexe') do (SET WERF_USE_SCRIPT_PATH=%g)
%WERF_USE_SCRIPT_PATH%
```

## Commands

- `multiwerf update <MAJOR.MINOR> [<CHANNEL>]`: Perform self-update and download the actual channel werf binary.

- `multiwerf use <MAJOR.MINOR> [<CHANNEL>]`: Generate the shell script that should be sourced to use the actual channel werf binary in the current shell session based on the local channel mapping.

- `multiwerf werf-path <MAJOR.MINOR> [<CHANNEL>]`: Print the actual channel werf binary path based on the local channel mapping..

- `multiwerf werf-exec <MAJOR.MINOR> [<CHANNEL>] [<WERF_ARGS>...]`: Exec the actual channel werf binary based on the local channel mapping.

The first positional argument is the version in the form of `MAJOR.MINOR`. `CHANNEL` is one of the following channels: alpha, beta, ea, stable, rock-solid. Read more about it in [Backward Compatibility Promise](https://github.com/werf/werf#backward-compatibility-promise) section.

multiwerf download werf binary to a directory like `$HOME/.multiwerf/VERSION/`. 
For example, the werf version `1.0.1-ea.3` for the user `gitlab-runner` will be stored as:

```
/home/gitlab-runner/.multiwerf
|-- 1.0.1-ea.3
|   |-- SHA256SUMS
|   |-- SHA256SUMS.sig
|   `-- werf-linux-amd64-1.0.1-ea.3
|
...
```

> `multiwerf update` checks for the latest version of multiwerf and performs self-update if it is needed. This can be disabled with `--self-update=no` flag. 

## Self-update

Before downloading the actual channel werf binary multiwerf performs self-update process. If the new version is available multiwerf downloads it and starts the new process with the same environment and arguments.

`--self-update=no` flag and `MULTIWERF_SELF_UPDATE=no` environment variable are available to turn off self-updates.

Self-update is disabled if `multiwerf` binary is not owned by user that runs it and if the binary file is not writable by owner. 

## License

Apache License 2.0, see [LICENSE](LICENSE)
