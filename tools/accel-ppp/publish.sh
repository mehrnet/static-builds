#!/usr/bin/env bash
# Builds a portable accel-ppp tree (daemon, CLI, plugins, musl loader and
# runtime libraries) from a pinned upstream tag for linux/amd64 + linux/arm64.
# The tree keeps bin/ and lib/ paths so the sheed-node bundle can install it
# under its prefix without rewriting rpaths.
set -euo pipefail

REPO="mehrnet/static-builds"
UPSTREAM_REPO="accel-ppp/accel-ppp"
NAME="accel-ppp"

tags=$(git ls-remote --tags --refs --sort=-v:refname "https://github.com/$UPSTREAM_REPO")
latest_tag=$(printf '%s\n' "$tags" | sed -n '1s#.*refs/tags/##p')
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
    --build-arg "ACCEL_PPP_VERSION=${version}" \
    --target export \
    --output "type=local,dest=${out_dir}" \
    tools/accel-ppp

  out="${NAME}_${version}_linux_${arch}.tar.gz"
  tar -C "$out_dir" -czf "$workdir/$out" bin lib

  (cd "$workdir" && sha256sum "$out" >> checksums.txt)
  assets+=("$workdir/$out")
done

gh release create "$our_tag" --repo "$REPO" \
  --title "accel-ppp $latest_tag" \
  --notes "Portable accel-ppp build from [accel-ppp/accel-ppp](https://github.com/$UPSTREAM_REPO/tree/$latest_tag) ($latest_tag) against musl (Alpine). Ships the daemon, accel-cmd, plugins, libtriton, musl loader and runtime libraries under bin/ and lib/. The wrapper sets ACCEL_PPP_MODULE_PATH; host kernel modules (l2tp_ppp, pppol2tp, ppp_generic, xfrm) are still required for live L2TP sessions." \
  "${assets[@]}" "$workdir/checksums.txt"

echo "Published $our_tag with ${#assets[@]} assets."
