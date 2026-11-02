#!/usr/bin/env bash
set -euo pipefail

release_tag=${1:?Usage: release-build.sh VERSION OUTPUT_DIR}
output_dir=${2:?Usage: release-build.sh VERSION OUTPUT_DIR}
if [[ ! "$release_tag" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  echo "Expected a release version such as v0.1.0" >&2
  exit 1
fi

release_commit=$(git rev-parse --verify HEAD)

# Require a new directory so stale artifacts cannot enter a release.
mkdir "$output_dir"
output_dir=$(cd "$output_dir" && pwd)
build_dir=$(mktemp -d)
trap 'rm -rf "$build_dir"' EXIT

for target_os in linux darwin windows; do
  for target_arch in amd64 arm64; do
    archive_name="renodash_${release_tag}_${target_os}_${target_arch}"
    package_dir="$build_dir/$archive_name"
    mkdir "$package_dir"
    binary_name=renodash
    if [[ "$target_os" == windows ]]; then
      binary_name=renodash.exe
    fi
    CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" \
      go build -trimpath -ldflags "-X main.version=$release_tag -X main.commit=$release_commit" -o "$package_dir/$binary_name" ./cmd/renodash
    cp LICENSE README.md "$package_dir/"
    if [[ "$target_os" == windows ]]; then
      (cd "$package_dir" && zip -q "$output_dir/$archive_name.zip" "$binary_name" LICENSE README.md)
    else
      tar -czf "$output_dir/$archive_name.tar.gz" -C "$package_dir" "$binary_name" LICENSE README.md
    fi
  done
done

(cd "$output_dir" && sha256sum ./*.tar.gz ./*.zip > checksums.txt)
