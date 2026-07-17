#!/usr/bin/env bash
# Builds OpenVPN statically from source (see Dockerfile) for linux/amd64
# and linux/arm64, and publishes it under this repo's naming convention.
# Linux-only for now -- that's the realistic radar-node deployment target
# (VPS boxes), and a static musl build doesn't translate to macOS/Windows
# the way this Dockerfile is set up; those can be added later as their own
# build path if actually needed.
#
# Idempotent the same way tools/xray/publish.sh is: this repo's own release
# tags are the "already built this version" record.
set -euo pipefail

REPO="mehrnet/static-builds"

UPSTREAM_REPO="OpenVPN/openvpn"
NAME="openvpn"

latest_tag=$(gh api "repos/$UPSTREAM_REPO/releases/latest" --jq .tag_name)
version="${latest_tag#v}"
our_tag="${NAME}-${latest_tag}"

if gh release view "$our_tag" --repo "$REPO" >/dev/null 2>&1; then
  echo "Already published $our_tag -- nothing to do."
  exit 0
fi

echo "Building/publishing $our_tag (upstream $latest_tag)..."

workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT

assets=()
: > "$workdir/checksums.txt"

for arch in amd64 arm64; do
  out_dir="$workdir/out_$arch"
  mkdir -p "$out_dir"

  docker buildx build \
    --platform "linux/$arch" \
    --build-arg "OPENVPN_VERSION=${version}" \
    --target export \
    --output "type=local,dest=${out_dir}" \
    tools/openvpn

  out="openvpn_${version}_linux_${arch}.tar.gz"
  chmod +x "$out_dir/openvpn"
  tar -C "$out_dir" -czf "$workdir/$out" openvpn

  (cd "$workdir" && sha256sum "$out" >> checksums.txt)
  assets+=("$workdir/$out")
done

gh release create "$our_tag" --repo "$REPO" \
  --title "openvpn $latest_tag" \
  --notes "Static build of [OpenVPN/openvpn](https://github.com/$UPSTREAM_REPO/releases/tag/$latest_tag) ($latest_tag) against musl (Alpine), management interface and plugin loading disabled. linux/amd64 + linux/arm64 only." \
  "${assets[@]}" "$workdir/checksums.txt"

echo "Published $our_tag with ${#assets[@]} assets."
