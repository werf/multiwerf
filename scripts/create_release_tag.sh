#!/bin/bash

set -e

for f in $(find scripts/lib -type f -name "*.sh"); do
  source $f
done

if [ -z "${1-}" ] ; then
  echo "Usage: $0 VERSION"
  echo
  exit 1
fi

DIR="$(dirname "${0}")"
CHANGELOG_TEXT=""
EXTRA_GIT_TAG_OPTS=""

if [ "$1" = ":experimental" ] ; then
  VERSION="v$(date --utc +%y.%m.%d-%H.%M.%S)"
  BINTRAY_REPO=multiwerf-experimental

  echo "Creating experimental release version $VERSION"

  CHANGELOG_TEXT="Experimental release."
else
  VERSION=$1
  BINTRAY_REPO=multiwerf

  LATEST_TAG="$(git tag -l --sort=-taggerdate | head -n1)"
  CHANGELOG_TEXT="$(git log --pretty="%s" HEAD...${LATEST_TAG})"
  if [[ -n $CHANGELOG_TEXT ]] ; then
    CHANGELOG_TEXT="$(echo "$CHANGELOG_TEXT" | grep -v '^Merge' | sed 's/^/- /')"
  fi
  EXTRA_GIT_TAG_OPTS="--edit"

  echo "Creating release version $VERSION"
fi

TAG_TEMPLATE="$DIR/git_tag_template.md"

BINTRAY_REPO="${BINTRAY_REPO}" VERSION="${VERSION}" CHANGELOG_TEXT="${CHANGELOG_TEXT}" envsubst < ${TAG_TEMPLATE} | git tag --annotate --file - $EXTRA_GIT_TAG_OPTS $VERSION

git push $GIT_ORIGIN --tags
