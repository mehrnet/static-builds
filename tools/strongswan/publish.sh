#!/usr/bin/env bash
# Builds a portable monolithic strongSwan charon + swanctl tree from the latest
# upstream tag for linux/amd64 + linux/arm64.
set -euo pipefail

REPO="mehrnet/static-builds"
UPSTREAM_REPO="strongswan/strongswan"
NAME="strongswan"

tags=$(git ls-remote --tags --refs "https://github.com/$UPSTREAM_REPO")
latest_tag=$(printf '%s\n' "$tags" | sed 's#.*refs/tags/##' | grep -E '^[0-9]+\.[0-9]+(\.[0-9]+)?$' | sort -V | tail -1)
if [ -z "$latest_tag" ]; then
  echo "Could not determine latest $UPSTREAM_REPO tag." >&2
  exit 1
fi
version="$latest_tag"
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
    --build-arg "STRONGSWAN_VERSION=${version}" \
    --target export \
    --output "type=local,dest=${out_dir}" \
    tools/strongswan

  out="${NAME}_${version}_linux_${arch}.tar.gz"
  tar -C "$out_dir" -czf "$workdir/$out" bin lib

  (cd "$workdir" && sha256sum "$out" >> checksums.txt)
  assets+=("$workdir/$out")
done

gh release create "$our_tag" --repo "$REPO" \
  --title "strongswan $latest_tag" \
  --notes "Portable monolithic strongSwan $latest_tag build from [strongswan/strongswan](https://github.com/$UPSTREAM_REPO/tree/$latest_tag) against musl (Alpine). Ships charon, swanctl, libstrongswan, libcharon, libvici, musl loader and runtime libraries under bin/ and lib/. Host xfrm/netlink kernel support is still required for live IPsec." \
  "${assets[@]}" "$workdir/checksums.txt"

echo "Published $our_tag with ${#assets[@]} assets."
