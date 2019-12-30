#!/usr/bin/env bash

set -e

for f in $(find scripts/lib -type f -name "*.sh"); do
  source $f
done

# publisher utility
# Build, create github release and upload go binaries to bintray.
main() {
  parse_args "$@" || (usage && exit 1)

  if [[ -z $PUBLISH_BINTRAY_AUTH ]] ; then
    echo "PUBLISH_BINTRAY_AUTH token required!" >&2
    exit 1
  fi

  # check for git and curl
  check_git || (echo "$0: cannot find git command" && exit 2)
  check_curl || (echo "$0: cannot find curl command" && exit 2)

  VERSION="v$(date --utc +%y.%m.%d-%H.%M.%S)"

  # prevent export variables from build.sh
  (scripts/build_release.sh "$VERSION") || (echo "$0: scripts/build_release.sh failed" && exit 2)

  echo "Publish version $VERSION"
  ( bintray_create_version "$VERSION" && echo "  Bintray: Version $VERSION created" ) || ( exit 1 )

  echo "Upload assets"
  echo "  Upload to bintray"
  (
   cd "$RELEASE_BUILD_DIR/$VERSION"
   for filename in "${BASE_NAME}"-* SHA256SUMS info.txt ; do
     echo "  - $filename"
     ( bintray_upload_file_into_version "$VERSION" "$filename" "$VERSION/$filename" ) || ( exit 1 )
   done
  )
  echo "  Publish files"
  ( bintray_publish_files_in_version "$VERSION" ) || ( exit 1 )
}

usage() {
printf " Usage: %s [--bintray-token TOKEN]
    --bintray-auth user:TOKEN
            User and token for upload to bintray.com. No bintray actions if no token specified.

    --help|-h
            Print help

" "$0"
}

parse_args() {
  while [ $# -gt 0 ]; do
    case "$1" in
      --github-token)
        PUBLISH_GITHUB_TOKEN="$2"
        shift
        ;;
      --bintray-auth)
        PUBLISH_BINTRAY_AUTH="$2"
        shift
        ;;
      --help|-h)
        return 1
        ;;
      --*)
        echo "Illegal option $1"
        return 1
        ;;
    esac
    shift $(( $# > 0 ? 1 : 0 ))
  done
}

# wait for full file download if executed as
# $ curl | sh
main "$@"
