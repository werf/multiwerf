#!/usr/bin/env bash
set -e

#BINTRAY_AUTH=         # bintray auth user:TOKEN
BINTRAY_SUBJECT=flant  # bintray organization
BINTRAY_REPO=werf      # bintray repository
BINTRAY_PACKAGE=werf   # bintray package in repository

#NO_PRERELEASE=        # This is not a pre release

#GITHUB_TOKEN=         # github API token
GITHUB_OWNER=flant     # github user/org
GITHUB_REPO=werf       # github repository

RELEASE_BUILD_DIR=release/build

#GIT_TAG=              # git tag value i.e. from $CIRCLE_TAG or $CI_COMMIT_TAG or $TRAVIS_TAG
GIT_REMOTE=origin      # can be changed to upstream with env

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

  # checkout tag if not in ci environment
  if [[ -z $CI ]] ; then
    git checkout -f "$GIT_TAG" || (echo "$0: git checkout '$GIT_TAG' error" && exit 2)
  fi

  # There are 2 variants for a VERSION:
  # version for release without v prefix
  #VERSION=${GIT_TAG#v}
  # version for release equals git tag (preserve v prefix)
  VERSION=${GIT_TAG}

  # prevent export variables from build.sh
  (release/build.sh "$GIT_TAG") || (echo "$0: build.sh failed" && exit 2)

  # message for github release and bintray version description
  # change to *contents to get a commit message instead
  TAG_RELEASE_MESSAGE=$(git for-each-ref --format="%(contents)" "refs/tags/$GIT_TAG" | jq -R -s '.' )
  TAG_RELEASE_MESSAGE=${TAG_RELEASE_MESSAGE:="\"\""}

  echo "Publish version $VERSION from git tag $GIT_TAG"
  if [ -n "$BINTRAY_AUTH" ] ; then
    ( bintray_create_version "$VERSION" && echo "Bintray: Version $VERSION created" ) || ( exit 1 )

    (
     cd $RELEASE_BUILD_DIR
     for filename in "${BASE_NAME}-"* SHA256SUMS info.txt ; do
       echo "Upload $filename"
       ( bintray_upload_file_into_version "$VERSION" "$filename" "$VERSION/$filename" ) || ( exit 1 )
     done
    )
  else
    echo "Bintray: cannot upload without token"
  fi

  if [ -n "$GITHUB_TOKEN" ] ; then
    ( github_create_release && echo "Github: Release for tag $GIT_TAG created" ) || ( exit 1 )
  else
    echo "Github: cannot create release without token"
  fi
}

bintray_create_version() {
  local VERSION=$1

PAYLOAD=$(cat <<- JSON
  {
     "name": "${VERSION}",
     "desc": ${TAG_RELEASE_MESSAGE},
     "vcs_tag": "${GIT_TAG}"
  }
JSON
)
  curlResponse=$(mktemp)
  status=$(curl -s -w '%{http_code}' -o "$curlResponse" \
      --request POST \
      --user "$BINTRAY_AUTH" \
      --header "Content-type: application/json" \
      --data "$PAYLOAD" \
      "https://api.bintray.com/packages/${BINTRAY_SUBJECT}/${BINTRAY_REPO}/${BINTRAY_PACKAGE}/versions"
  )

  echo "Bintray create version ${VERSION}: curl return status $status with response"
  cat "$curlResponse"
  echo
  rm "$curlResponse"

  ret=0
  if [ "x$(echo "$status" | cut -c1)" != "x2" ]
  then
    ret=1
  fi

  return $ret
}

# upload file to a package $BINTRAY_PACKAGE version $VERSION
bintray_upload_file_into_version() {
  local VERSION=$1
  local UPLOAD_FILE_PATH=$1
  local DESTINATION_PATH=$2

  curlResponse=$(mktemp)
  status=$(curl -s -w '%{http_code}' -o "$curlResponse" \
      --header "X-Bintray-Publish: 1" \
      --header "X-Bintray-Override: 1" \
      --header "X-Bintray-Package: $BINTRAY_PACKAGE" \
      --header "X-Bintray-Version: $VERSION" \
      --header "Content-type: application/binary" \
      --request PUT \
      --user "$BINTRAY_AUTH" \
      --upload-file "$UPLOAD_FILE_PATH" \
      "https://api.bintray.com/content/${BINTRAY_SUBJECT}/${BINTRAY_REPO}/$DESTINATION_PATH"
  )

  echo "Bintray upload $DESTINATION_PATH: curl return status $status with response"
  cat "$curlResponse"
  echo
  rm "$curlResponse"

  ret=0
  if [ "x$(echo "$status" | cut -c1)" != "x2" ]
  then
    ret=1
  else
    dlUrl="https://dl.bintray.com/${BINTRAY_SUBJECT}/${BINTRAY_REPO}/${DESTINATION_PATH}"
    echo "Bintray: $DESTINATION_PATH uploaded to ${dlUrl}"
  fi

  return $ret
}

github_create_release() {
  prerelease="true"
  if [[ "$NO_PRERELEASE" == "yes" ]] ; then
    prerelease="false"
    echo "# Creating release $GIT_TAG"
  else
    echo "# Creating pre-release $GIT_TAG"
  fi

  GHPAYLOAD=$(cat <<- JSON
{
  "tag_name": "$GIT_TAG",
  "name": "Werf $VERSION",
  "body": $TAG_RELEASE_MESSAGE,
  "draft": false,
  "prerelease": $prerelease
}
JSON
)

  curlResponse=$(mktemp)
  status=$(curl -s -w '%{http_code}' -o "$curlResponse" \
      --request POST \
      --header "Authorization: token $GITHUB_TOKEN" \
      --header "Accept: application/vnd.github.v3+json" \
      --data "$GHPAYLOAD" \
      "https://api.github.com/repos/$GITHUB_OWNER/$GITHUB_REPO/releases"
  )

  echo "Github create release: curl return status $status with response"
  cat "$curlResponse"
  echo
  rm "$curlResponse"

  ret=0
  if [ "x$(echo "$status" | cut -c1)" != "x2" ]
  then
    ret=1
  fi

  return $ret
}

check_git() {
  type git > /dev/null 2>&1 || return 1
}

check_curl() {
  type curl > /dev/null 2>&1 || return 1
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
