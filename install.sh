#!/usr/bin/env bash

# Invoking this script:
#
# curl https://raw.githubusercontent.com/flant/multiwerf/master/install.sh | sh
#
# - download latest version of multiwerf file
# - making sure multiwerf is executable
# - explain what was done
#

set -eo pipefail -o nounset

function check_os_arch {
  local supported="linux-amd64 darwin-amd64"

  if ! echo "${supported}" | tr ' ' '\n' | grep -q "${OS}-${ARCH}"; then
    cat <<EOF

${PROGRAM} installation is not currently supported on ${OS}-${ARCH}.

See https://github.com/flant/multiwerf for more information.

EOF
  fi
}

function get_latest_version {
  local url="${1}"
  local version
  version="$(curl -sI "${url}" | grep "Location:" | sed -n 's%.*multiwerf/%%;s%/view.*%%;s%\r%%;p' )"

  if [ -z "${version}" ]; then
    echo "There doesn't seem to be a version of ${PROGRAM} avaiable at ${url}." 1>&2
    return 1
  fi

  url_decode "${version}"
}

function url_decode {
  local url_encoded="${1//+/ }"
  printf '%b' "${url_encoded//%/\\x}"
}

PROGRAM="multiwerf"
OS="$(uname | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
BINTRAY_LATEST_VERSION_URL="https://bintray.com/flant/multiwerf/multiwerf/_latestVersion"
BINTRAY_DL_URL_BASE="https://dl.bintray.com/flant/multiwerf"

if [ "${ARCH}" == "x86_64" ]; then
  ARCH="amd64"
fi

check_os_arch

VERSION="$(get_latest_version "${BINTRAY_LATEST_VERSION_URL}")"
MULTIWERF_BIN_NAME="multiwerf-${OS}-${ARCH}-${VERSION}"
echo "Downloading ${MULTIWERF_BIN_NAME} from bintray..."
curl -Ls "${BINTRAY_DL_URL_BASE}/${VERSION}/${MULTIWERF_BIN_NAME}" -o "${MULTIWERF_BIN_NAME}"

# check hash
curl -Ls "${BINTRAY_DL_URL_BASE}/${VERSION}/SHA256SUMS" -o "${PROGRAM}.sha256sums"
if ! (sha256sum -c --ignore-missing ${PROGRAM}.sha256sums) ; then
  echo "${MULTIWERF_BIN_NAME} sha256 hash is not verified. Please download and check hash manually."
  rm "${PROGRAM}.sha256sums"
  rm "${MULTIWERF_BIN_NAME}"
  exit 1
fi

rm "${PROGRAM}.sha256sums"
mv "${MULTIWERF_BIN_NAME}" "${PROGRAM}"
chmod +x "${PROGRAM}"

cat <<EOF

${PROGRAM} is now available in your current directory.

To learn more, execute:

    $ ./${PROGRAM}

EOF