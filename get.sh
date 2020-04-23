#!/usr/bin/env sh

# Invoking this script:
#
# curl https://raw.githubusercontent.com/flant/multiwerf/master/get.sh | sh
#
# Actions:
# - check os and arch
# - detect curl or wget
# - check bintary for latest available release of multiwerf binary
# - download multiwerf, check SHA256
# - making sure multiwerf is executable
# - print brief usage

set -e -o nounset

http_client="curl"

detect_downloader() {
  if tmp=$(curl --version 2>&1 >/dev/null) ; then return ; fi
  if tmp=$(wget --help 2>&1 >/dev/null) ; then http_client="wget" ; return ; fi
  echo "Cannot detect curl or wget. Install one of them and run again."
  exit 2
}

# download_file URL OUTPUT_FILE_PATH
download_file() {
  if [ "${http_client}" = "curl" ] ; then
    if ! curl -Ls "$1" -o "$2" ; then
      echo "curl error for file $1"
      return 1
    fi
    return
  fi
  if [ "${http_client}" = "wget" ] ; then
    if ! wget -q -O "$2" "$1" ; then
      echo "wget error for file $1"
      return 1
    fi
  fi
}

# get_location_header URL
get_location_header() {
  if [ "${http_client}" = "curl" ] ; then
    if ! curl -s "$1" -w "%{redirect_url}" ; then
      echo "curl error for $1"
      return 1
    fi
    return
  fi
  if [ "${http_client}" = "wget" ] ; then
    if ! wget -S -q -O - "$1" 2>&1 | grep -m 1 'Location:' | tr -d '\r\n' ; then
      echo "wget error for $1"
      return 1
    fi
  fi
}

check_os_arch() {
  supported="linux-amd64 darwin-amd64"

  if ! echo "${supported}" | tr ' ' '\n' | grep -q "${OS}-${ARCH}"; then
    cat <<EOF

${PROGRAM} installation is not currently supported on ${OS}-${ARCH}.

See https://github.com/flant/multiwerf for more information.

EOF
  fi
}

get_latest_version() {
  url="${1}"
  version="$(get_location_header "${url}" | sed 's%.*multiwerf/%%;s%/view.*%%')"

  if [ "x${version}" = "x" ]; then
    echo "There doesn't seem to be a version of ${PROGRAM} avaiable at ${url}." 1>&2
    return 1
  fi

  url_decode "${version}"
}

url_decode() {
  echo "$1" | sed 's@+@ @g;s@%@\\x@g' | xargs -0 printf '%b'
}

# emulate missing option --ignore-missing of sha256sum for alpine and centos
# use shasum on MacOS
sha256check() {
  BIN_FILE=$1
  SHA_FILE=$2
  SHA_SUM="${SHA_FILE}.sum"

  grep "$BIN_FILE" "$SHA_FILE" > "$SHA_SUM"

  sha_cmd="sha256sum"
  if [ "$OS" = "darwin" ] ; then
    sha_cmd="shasum -a 256"
  fi

  if ! $sha_cmd -c "${SHA_SUM}" ; then
    rm -f "${SHA_SUM}" "${SHA_FILE}" "${BIN_FILE}"
    return 1
  fi

  rm -f "${SHA_SUM}" "${SHA_FILE}"
}

PROGRAM="multiwerf"
OS="$(uname | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
BINTRAY_LATEST_VERSION_URL="https://bintray.com/flant/multiwerf/multiwerf/_latestVersion"
BINTRAY_DL_URL_BASE="https://dl.bintray.com/flant/multiwerf"

if [ "${ARCH}" = "x86_64" ] ; then
  ARCH="amd64"
fi

check_os_arch

detect_downloader

VERSION="$(get_latest_version "${BINTRAY_LATEST_VERSION_URL}")"
MULTIWERF_BIN_NAME="multiwerf-${OS}-${ARCH}-${VERSION}"

echo "Downloading ${MULTIWERF_BIN_NAME} from bintray..."
if ! download_file "${BINTRAY_DL_URL_BASE}/${VERSION}/${MULTIWERF_BIN_NAME}" "${MULTIWERF_BIN_NAME}"
then
  exit 2
fi

# check hash
echo "Checking hash sum..."
if ! download_file "${BINTRAY_DL_URL_BASE}/${VERSION}/SHA256SUMS" "${PROGRAM}.sha256sums"
then
  exit 2
fi

if ! sha256check "${MULTIWERF_BIN_NAME}" "${PROGRAM}.sha256sums"
then
  echo "${MULTIWERF_BIN_NAME} sha256 hash is not verified. Please download and check hash manually."
  exit 1
fi

mv "${MULTIWERF_BIN_NAME}" "${PROGRAM}"
chmod +x "${PROGRAM}"

cat <<EOF

${PROGRAM} is now available in your current directory.

To learn more, execute:

    $ ./${PROGRAM} help

To use latest werf, execute:

    $ source $(./${PROGRAM} use 1.1 stable --as-file)

EOF
