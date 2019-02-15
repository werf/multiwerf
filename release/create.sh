#!/bin/bash

set -eo pipefail -o nounset

if [ -z "${1-}" ] ; then
  echo "Usage: $0 VERSION"
  echo
  exit 1
fi

DIR="$(dirname "${0}")"
VERSION=${1}

TAG_TEMPLATE="$DIR/git_tag_template.md"

LATEST_TAG="$(git tag -l --sort=-taggerdate | head -n1)"
echo "latest tag is ${LATEST_TAG}"
CHANGELOG_TEXT="$(git log --pretty="%s" HEAD...${LATEST_TAG})"
if [[ -n $CHANGELOG_TEXT ]] ; then
  CHANGELOG_TEXT="$(echo "$CHANGELOG_TEXT" | grep -v '^Merge' | sed 's/^/- /')"
fi
echo "CHANGELOG_TEXT = ${CHANGELOG_TEXT}"
echo "envsubst"

envsubst < ${TAG_TEMPLATE} | git tag --annotate --file - --edit $VERSION

git push --tags
