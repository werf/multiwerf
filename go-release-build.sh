#!/bin/bash

version=$1

os_archs=(
#  "linux/386"
  "linux/amd64"
  "darwin/amd64"
#  "windows/386/.exe"
#  "windows/amd64/.exe"
)

package=github.com/flant/multiwerf/cmd/multiwerf
bin_base_name=multiwerf

build_dir=$(pwd)/release-build
rm -rf $build_dir
mkdir -p ${build_dir}

for os_arch in ${os_archs[@]}; do
  a=(${os_arch//\// })
  os=${a[0]}
  arch=${a[1]}
  ext=${a[2]}
  echo "Build for $os/$arch..."
  output=${build_dir}/${bin_base_name}-${os}-${arch}-${version}${ext}
  GOOS=${os} GOARCH=${arch} go build -ldflags="-s -w -X github.com/flant/multiwerf/pkg/app.Version=${version} -X github.com/flant/multiwerf/pkg/app.OsArch=${os}-${arch}" -o ${output} ${package}
done

$(
cd $build_dir
for i in ${bin_base_name}*
do
  sha256sum $i >> SHA256SUMS
done
)

# save date and commit
datetime=$(date +%d.%m.%Y\ %H:%M:%S)
commit=$(git rev-parse HEAD)
cat <<EOF > ${build_dir}/info.txt
Build date: ${datetime}
Commit: ${commit}
EOF