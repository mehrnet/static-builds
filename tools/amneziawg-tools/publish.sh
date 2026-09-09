#!/usr/bin/env bash
# Builds the AmneziaWG userspace tools (awg + awg-quick) statically from
# source (see Dockerfile) for linux/amd64 + linux/arm64, and publishes
# them under this repo's naming convention. Same shape as
# tools/openvpn/publish.sh -- musl/Alpine via docker buildx, this repo's
# own release tags as the "already built this version" record.
#
# Upstream ships date-based tags (v3.1.YYYYMMDD), no GitHub "releases",
# so the version is the newest tag by version sort.
set -euo pipefail

REPO="mehrnet/static-builds"
UPSTREAM_REPO="amnezia-vpn/amneziawg-tools"
NAME="amneziawg-tools"

latest_tag=$(git ls-remote --tags --refs --sort=-v:refname \
  "https://github.com/$UPSTREAM_REPO" | head -1 | sed 's#.*refs/tags/##')
if [ -z "$latest_tag" ]; then
  echo "Could not determine latest $UPSTREAM_REPO tag." >&2
  exit 1
fi
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
    --build-arg "AWG_TOOLS_VERSION=${version}" \
    --target export \
    --output "type=local,dest=${out_dir}" \
    tools/amneziawg-tools

  out="amneziawg-tools_${version}_linux_${arch}.tar.gz"
  chmod +x "$out_dir/awg" "$out_dir/awg-quick"
  tar -C "$out_dir" -czf "$workdir/$out" awg awg-quick

  (cd "$workdir" && sha256sum "$out" >> checksums.txt)
  assets+=("$workdir/$out")
done

gh release create "$our_tag" --repo "$REPO" \
  --title "amneziawg-tools $latest_tag" \
  --notes "Static build of [amnezia-vpn/amneziawg-tools](https://github.com/$UPSTREAM_REPO/tree/$latest_tag) ($latest_tag) against musl (Alpine): the \`awg(8)\` CLI plus the \`awg-quick(8)\` script. \`awg-quick\` auto-falls back to \`amneziawg-go\` (sibling tool) when the kernel module is absent. linux/amd64 + linux/arm64." \
  "${assets[@]}" "$workdir/checksums.txt"

echo "Published $our_tag with ${#assets[@]} assets."
