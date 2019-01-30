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

VERSION="$VERSION" envsubst < ${TAG_TEMPLATE} | git tag --annotate --file - --edit $VERSION

git push --tags
