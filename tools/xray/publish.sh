#!/usr/bin/env bash
# Re-hosts XTLS/Xray-core's own official static release assets under this
# repo's naming convention (xray_<version>_<os>_<arch>.<ext>, matching
# radar-node's own goreleaser output) -- Xray-core already publishes fully
# static binaries per platform, so there's nothing to actually build here,
# just a consistent, versioned, checksum-verified re-host so install.sh has
# one uniform fetch shape across every bundled tool.
#
# Idempotent: this repo's own release tags ARE the "already done" record --
# if xray-<latest_upstream_tag> already exists here, this is a no-op. Safe
# to run on a schedule or by hand with no extra state to track.
set -euo pipefail

UPSTREAM_REPO="XTLS/Xray-core"
NAME="xray"

latest_tag=$(gh api "repos/$UPSTREAM_REPO/releases/latest" --jq .tag_name)
version="${latest_tag#v}"
our_tag="${NAME}-${latest_tag}"

if gh release view "$our_tag" >/dev/null 2>&1; then
  echo "Already published $our_tag -- nothing to do."
  exit 0
fi

echo "Publishing $our_tag (upstream $latest_tag)..."

workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT
cd "$workdir"

# Xray-core's own asset names -> our os_arch naming.
declare -A ASSET_MAP=(
  ["linux_amd64"]="Xray-linux-64.zip"
  ["linux_arm64"]="Xray-linux-arm64-v8a.zip"
  ["darwin_amd64"]="Xray-macos-64.zip"
  ["darwin_arm64"]="Xray-macos-arm64-v8a.zip"
  ["windows_amd64"]="Xray-windows-64.zip"
  ["windows_arm64"]="Xray-windows-arm64-v8a.zip"
)

assets=()
: > checksums.txt

for platform in "${!ASSET_MAP[@]}"; do
  upstream_asset="${ASSET_MAP[$platform]}"
  os="${platform%_*}"
  arch="${platform#*_}"

  url="https://github.com/$UPSTREAM_REPO/releases/download/$latest_tag/$upstream_asset"
  echo "Fetching $url"
  curl -fsSL -o upstream.zip "$url"

  extract_dir="extract_$platform"
  mkdir -p "$extract_dir"
  unzip -q upstream.zip -d "$extract_dir"

  bin_name="xray"
  [ "$os" = "windows" ] && bin_name="xray.exe"
  bin_path=$(find "$extract_dir" -type f -iname "$bin_name" | head -1)
  if [ -z "$bin_path" ]; then
    echo "ERROR: could not find $bin_name in $upstream_asset" >&2
    exit 1
  fi

  if [ "$os" = "windows" ]; then
    out="xray_${version}_${os}_${arch}.zip"
    (cd "$(dirname "$bin_path")" && zip -q "$workdir/$out" "$(basename "$bin_path")")
  else
    out="xray_${version}_${os}_${arch}.tar.gz"
    chmod +x "$bin_path"
    tar -C "$(dirname "$bin_path")" -czf "$out" "$(basename "$bin_path")"
  fi

  sha256sum "$out" >> checksums.txt
  assets+=("$out")
  rm -f upstream.zip
  rm -rf "$extract_dir"
done

gh release create "$our_tag" \
  --title "xray $latest_tag" \
  --notes "Re-hosted static build of [XTLS/Xray-core](https://github.com/$UPSTREAM_REPO/releases/tag/$latest_tag) ($latest_tag) -- unpacked from upstream's own official release assets and repackaged under this repo's naming convention. Not rebuilt from source; upstream already ships fully static binaries." \
  "${assets[@]}" checksums.txt

echo "Published $our_tag with ${#assets[@]} assets."
