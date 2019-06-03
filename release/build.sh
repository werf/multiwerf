#!/bin/bash
set -eo pipefail -o nounset

# build parameters
OS_ARCHS=(
  "linux-amd64"
  "darwin-amd64"
  #"windows-amd64"
)
BUILD_PACKAGE=github.com/flant/multiwerf/cmd/multiwerf
VERSION_VAR_NAME=github.com/flant/multiwerf/pkg/app.Version
RELEASE_BUILD_DIR=release/build

if [ -z "${1-}" ] ; then
  echo "Usage: $0 VERSION"
  echo
  exit 1
fi

VERSION=$1
BASE_NAME=${BUILD_PACKAGE##*/}
BUILD_DIR="${RELEASE_BUILD_DIR:?}/${VERSION:?}"

rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

for os_arch in "${OS_ARCHS[@]}"; do
  os=${os_arch%-*}
  arch=${os_arch#*-}
  outputFile="$BUILD_DIR/${BASE_NAME}-${os}-${arch}-${VERSION}"
  if [ "$os" == "windows" ] ; then
    outputFile="$outputFile.exe"
  fi

  echo "# Building ${BASE_NAME} $VERSION for $os $arch ..."

  GOOS=${os} GOARCH=${arch} CGO_ENABLED=0 \
  go build -ldflags="-s -w -X ${VERSION_VAR_NAME}=${VERSION} -X github.com/flant/multiwerf/pkg/app.OsArch=${os}-${arch}" \
           -o "${outputFile}" "${BUILD_PACKAGE}"
done

(
cd "$BUILD_DIR"
sha256sum "${BASE_NAME}"-* > SHA256SUMS
)

# save build date and commit
datetime="$(date +%d.%m.%Y\ %H:%M:%S)"
commit="$(git rev-parse HEAD)"
cat <<EOF > "${BUILD_DIR}/info.txt"
Build date: ${datetime}
Commit: ${commit}
EOF
