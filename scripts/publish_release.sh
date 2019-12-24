#!/usr/bin/env bash

set -e

for f in $(find scripts/lib -type f -name "*.sh"); do
  source $f
done

# publisher utility
# Build, create github release and upload go binaries to bintray.
main() {
  parse_args "$@" || (usage && exit 1)

  if [[ -z $BINTRAY_AUTH && -z $GITHUB_TOKEN ]] ; then
    echo "Warning! No bintray or github token specified."
    echo
  fi

  # check for git and curl
  check_git || (echo "$0: cannot find git command" && exit 2)
  check_curl || (echo "$0: cannot find curl command" && exit 2)

  TAG_LOCAL_SHA=$(git for-each-ref --format='%(objectname)' "refs/tags/$GIT_TAG")
  TAG_REMOTE_SHA=$(git ls-remote --tags $GIT_REMOTE "refs/tags/$GIT_TAG" | cut -f 1)

  if [ "x$TAG_LOCAL_SHA" != "x$TAG_REMOTE_SHA" ] ; then
    echo "CRITICAL: Tag $GIT_TAG should be pushed to $GIT_REMOTE before creating new release"
    exit 1
  fi

  # message for github release and bintray version description
  # change to *contents to get a commit message instead
  TAG_RELEASE_MESSAGE=$(git for-each-ref --format="%(contents)" "refs/tags/$GIT_TAG" | jq -R -s '.' )
  TAG_RELEASE_MESSAGE=${TAG_RELEASE_MESSAGE:="\"\""}

  repoMatchLine="$(echo -e "$TAG_RELEASE_MESSAGE" | grep -P '^<!-- repo: .* -->$' || true)"
  if [ "x$repoMatchLine" != "x" ] ; then
    repo=${repoMatchLine#"<!-- repo: "}
    repo=${repo%" -->"}
  fi

  if [ "x$repo" != "x" ] ; then
    BINTRAY_REPO="$repo"
  fi

  # There are 2 variants for a VERSION:
  # version for release without v prefix
  #VERSION=${GIT_TAG#v}
  # version for release equals git tag (preserve v prefix)
  VERSION=${GIT_TAG}

  # prevent export variables from build.sh
  (scripts/build_release.sh "$GIT_TAG") || (echo "$0: scripts/build_release.sh failed" && exit 2)

  echo "Publish version $VERSION from git tag $GIT_TAG"
  if [ -n "$BINTRAY_AUTH" ] ; then
    ( bintray_create_version "$VERSION" && echo "  Bintray: Version $VERSION created" ) || ( exit 1 )
  else
    echo "  Bintray: cannot create a version without token"
  fi
  if [ -n "$GITHUB_TOKEN" ] ; then
    if github_create_release ; then
      echo "  Github: Release $VERSION for tag $GIT_TAG created"
    else
      exit 1
    fi
    echo GITHUB_RELEASE_ID='"'"${GITHUB_RELEASE_ID}"'"'
  else
    echo "  Github: cannot create release without token"
  fi

  echo "Upload assets"
  if [ -n "$BINTRAY_AUTH" ] ; then
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
  else
    echo "  Bintray: cannot upload without token"
  fi

  if [ -n "$GITHUB_TOKEN" ] ; then
    echo "  Upload to github"
    (
     cd "$RELEASE_BUILD_DIR/$VERSION"
     for filename in "${BASE_NAME}"-* SHA256SUMS info.txt ; do
       echo "  - $filename"
       ( github_upload_asset_for_release "$filename") || ( exit 1 )
     done
    )
  else
    echo "  Github: cannot upload without token"
  fi
}

usage() {
printf " Usage: %s --tag <tagname> [--github-token TOKEN] [--bintray-token TOKEN]

    --no-prerelease
            This is final release, not a prerelease. Prerelease will be created by default.
            Can be changed manually later in the github UI.

    --tag
            Release is a tag based. Tag should be present if gh-token specified.

    --github-token TOKEN
            Write access token for github. No github actions if no token specified.

    --bintray-auth user:TOKEN
            User and token for upload to bintray.com. No bintray actions if no token specified.

    --help|-h
            Print help

" "$0"
}

parse_args() {
  while [ $# -gt 0 ]; do
    case "$1" in
      --tag)
        GIT_TAG="$2"
        shift
        ;;
      --github-token)
        GITHUB_TOKEN="$2"
        shift
        ;;
      --bintray-auth)
        BINTRAY_AUTH="$2"
        shift
        ;;
      --no-prerelease)
        NO_PRERELEASE="yes"
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

  [ -z "$GIT_TAG" ] && return 1 || return 0
}

# wait for full file download if executed as
# $ curl | sh
main "$@"
