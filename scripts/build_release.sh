#!/bin/bash
set -eo pipefail -o nounset

for f in $(find scripts/lib -type f -name "*.sh"); do
  source $f
done

# build parameters
OS_ARCHS=(
  "linux-amd64"
  "linux-arm64"
  "darwin-amd64"
  "darwin-arm64"
  "windows-amd64"
)
BUILD_PACKAGE=github.com/werf/multiwerf/cmd/multiwerf
VERSION_VAR_NAME=github.com/werf/multiwerf/pkg/app.Version

if [[ -z "${1-}" ]] ; then
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
  if [[ "$os" == "windows" ]] ; then
    outputFile="$outputFile.exe"
  fi

  echo "# Building ${BASE_NAME} $VERSION for $os $arch ..."

  GOOS=${os} GOARCH=${arch} CGO_ENABLED=0 \
  go build -ldflags="-s -w -X ${VERSION_VAR_NAME}=${VERSION} -X github.com/werf/multiwerf/pkg/app.OsArch=${os}-${arch}" \
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

LATEST_DIR="${RELEASE_BUILD_DIR:?}/latest"

rm -rf "${LATEST_DIR}"
cp -r "${BUILD_DIR}" "${LATEST_DIR}"

for os_arch in "${OS_ARCHS[@]}"; do
  os=${os_arch%-*}
  arch=${os_arch#*-}
  outputFile="$LATEST_DIR/${BASE_NAME}-${os}-${arch}-${VERSION}"
  outputFileLatest="$LATEST_DIR/${BASE_NAME}-${os}-${arch}-latest"
  if [[ "$os" == "windows" ]] ; then
    outputFile="$outputFile.exe"
  fi

  mv "${outputFile}" "${outputFileLatest}"
done

(
cd "${LATEST_DIR}"
sha256sum "${BASE_NAME}"-* > SHA256SUMS
)
