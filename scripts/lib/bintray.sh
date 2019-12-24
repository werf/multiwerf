BASE_NAME=multiwerf
#BINTRAY_AUTH=                        # bintray auth user:TOKEN
BINTRAY_SUBJECT=flant                # bintray organization
BINTRAY_REPO=multiwerf-experimental  # bintray repository
BINTRAY_PACKAGE=multiwerf            # bintray package in repository

bintray_create_version() {
  local VERSION=$1

PAYLOAD=$(cat <<- JSON
  {
     "name": "${VERSION}",
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
  local UPLOAD_FILE_PATH=$2
  local DESTINATION_PATH=$3

  curlResponse=$(mktemp)
  status=$(curl -s -w '%{http_code}' -o "$curlResponse" \
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

bintray_publish_files_in_version() {
  local VERSION=$1

  curlResponse=$(mktemp)
  status=$(curl -s -w '%{http_code}' -o "$curlResponse" \
      --request POST \
      --user "$BINTRAY_AUTH" \
      --header "Content-type: application/json" \
      "https://api.bintray.com/content/${BINTRAY_SUBJECT}/${BINTRAY_REPO}/${BINTRAY_PACKAGE}/${VERSION}/publish"
  )

  echo "Bintray publish files in version ${VERSION}: curl return status $status with response"
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
