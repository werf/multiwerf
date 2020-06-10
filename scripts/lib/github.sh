#PUBLISH_GITHUB_TOKEN=         # github API token
GITHUB_OWNER=werf              # github user/org
GITHUB_REPO=multiwerf          # github repository
GITHUB_RELEASE_ID=

github_create_release() {
  echo "# Creating release $GIT_TAG"

  local GHPAYLOAD=$(cat <<- JSON
{
  "tag_name": "$GIT_TAG",
  "name": "Multiwerf $VERSION",
  "body": $TAG_RELEASE_MESSAGE,
  "draft": false,
  "prerelease": false
}
JSON
)

  local curlResponse=$(mktemp)
  local status=$(curl -s -w '%{http_code}' -o "$curlResponse" \
      --request POST \
      --header "Authorization: token $PUBLISH_GITHUB_TOKEN" \
      --header "Accept: application/vnd.github.v3+json" \
      --data "$GHPAYLOAD" \
      "https://api.github.com/repos/$GITHUB_OWNER/$GITHUB_REPO/releases"
  )

  echo "Github create release: curl return status $status with response"
  cat "$curlResponse"
  echo

  local ret=0
  if [ "x$(echo "$status" | cut -c1)" != "x2" ]
  then
    ret=1
  else
    GITHUB_RELEASE_ID=$(cat "$curlResponse" | jq '.id')
  fi

  rm "$curlResponse"

  return $ret
}

# upload file to a package $BINTRAY_PACKAGE version $VERSION
github_upload_asset_for_release() {
  local FILENAME=$1

  curlResponse=$(mktemp)
  status=$(curl -s -w '%{http_code}' -L -o "$curlResponse" \
      --header "Authorization: token $PUBLISH_GITHUB_TOKEN" \
      --header "Accept: application/vnd.github.v3+json" \
      --header "Content-type: application/binary" \
      --request POST \
      --upload-file "$FILENAME" \
      "https://uploads.github.com/repos/$GITHUB_OWNER/$GITHUB_REPO/releases/$GITHUB_RELEASE_ID/assets?name=$FILENAME"
  )

  echo "Github upload $FILENAME: curl return status $status with response"
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

