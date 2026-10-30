#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo 'usage: ./setup/mise-install.sh [--dry-run|--apply]'
}

parse_mode() {
  case "${1:---dry-run}" in
    --dry-run|--apply) printf '%s\n' "${1:---dry-run}" ;;
    -h|--help) usage; exit 0 ;;
    *) usage >&2; exit 2 ;;
  esac
}

require_ci_pin() {
  MISE_VERSION="${MISE_VERSION:?MISE_VERSION is required}"
  MISE_SHA256_LINUX_X64="${MISE_SHA256_LINUX_X64:?MISE_SHA256_LINUX_X64 is required}"
  MISE_SHA256_LINUX_ARM64="${MISE_SHA256_LINUX_ARM64:?MISE_SHA256_LINUX_ARM64 is required}"
  MISE_SHA256_MACOS_X64="${MISE_SHA256_MACOS_X64:?MISE_SHA256_MACOS_X64 is required}"
  MISE_SHA256_MACOS_ARM64="${MISE_SHA256_MACOS_ARM64:?MISE_SHA256_MACOS_ARM64 is required}"
}

resolve_platform() {
  case "$(uname -s):$(uname -m)" in
    Linux:x86_64|Linux:amd64)
      platform=linux-x64
      archive_suffix=.tar.xz
      sha="$MISE_SHA256_LINUX_X64"
      ;;
    Linux:aarch64|Linux:arm64)
      platform=linux-arm64
      archive_suffix=.tar.xz
      sha="$MISE_SHA256_LINUX_ARM64"
      ;;
    Darwin:x86_64|Darwin:amd64)
      platform=macos-x64
      archive_suffix=.tar.xz
      sha="$MISE_SHA256_MACOS_X64"
      ;;
    Darwin:arm64|Darwin:aarch64)
      platform=macos-arm64
      archive_suffix=.tar.xz
      sha="$MISE_SHA256_MACOS_ARM64"
      ;;
    *)
      echo "error: unsupported platform: $(uname -s):$(uname -m)" >&2
      return 1
      ;;
  esac
}

validate_target() {
  case "$install_path" in
    /*) ;;
    *) echo "error: install path must be absolute" >&2; return 1 ;;
  esac

  if [ -e "$install_path" ] || [ -L "$install_path" ]; then
    echo "error: target exists: $install_path" >&2
    return 1
  fi
}

install_mise() {
  tmp_dir="$(mktemp -d)"
  trap 'rm -rf -- "$tmp_dir"' EXIT
  archive="$tmp_dir/mise$archive_suffix"
  extracted_dir="$tmp_dir/extracted"
  mkdir -p "$extracted_dir"

  curl -fsSL \
    -o "$archive" \
    "https://github.com/jdx/mise/releases/download/v${MISE_VERSION}/mise-v${MISE_VERSION}-${platform}${archive_suffix}"

  printf '%s  %s\n' "$sha" "$archive" |
    if command -v sha256sum >/dev/null; then sha256sum -c -; else shasum -a 256 -c -; fi

  tar -xJf "$archive" -C "$extracted_dir"
  install -d "$(dirname "$install_path")"
  install -m 755 "$(find "$extracted_dir" -type f -name mise -print -quit)" "$install_path"
}

main() {
  local mode
  if [ "${1:-}" = '-h' ] || [ "${1:-}" = '--help' ]; then
    usage
    return 0
  fi
  mode="$(parse_mode "${1:-}")"
  install_path="${MISE_BOOTSTRAP_TARGET:-$HOME/.local/bin/mise}"

  validate_target

  if [ "$mode" = '--dry-run' ]; then
    echo "plan: bootstrap_mise $install_path"
    return 0
  fi

  require_ci_pin
  resolve_platform
  install_mise
}

main "$@"
