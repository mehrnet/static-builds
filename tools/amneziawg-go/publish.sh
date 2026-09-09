#!/usr/bin/env bash
# Builds the stock upstream amneziawg-go userspace daemon (not a wrapper,
# unlike tools/wireguard-go's radar-wg) statically for linux/amd64 +
# linux/arm64, and publishes it under this repo's naming convention.
#
# amneziawg-go is a fork of wireguard-go, so like it: pure Go, CGO-free,
# and a plain `CGO_ENABLED=0 go build` yields a fully static binary that
# runs on anything from Debian 9 to Ubuntu 26.04 with no libc dependency.
# The GOARCH=arm64 build is a native Go cross-compile -- no QEMU needed,
# which is why this tool's workflow doesn't set up buildx the way
# amneziawg-tools' does.
#
# Upstream ships date-based tags (v3.1.YYYYMMDD), no GitHub "releases",
# so the version is the newest tag by version sort. Idempotent the same
# way the other tools' publish.sh are: this repo's own release tag is the
# "already built this version" record.
set -euo pipefail

REPO="mehrnet/static-builds"
UPSTREAM_REPO="amnezia-vpn/amneziawg-go"
NAME="amneziawg-go"

# Capture ls-remote fully before slicing: piping it straight into `head`
# lets head close the pipe early and kill git with SIGPIPE, which under
# `set -o pipefail` fails the whole script (exit 141).
tags=$(git ls-remote --tags --refs --sort=-v:refname "https://github.com/$UPSTREAM_REPO")
latest_tag=$(printf '%s\n' "$tags" | sed -n '1s#.*refs/tags/##p')
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

src="$workdir/src"
git clone --depth 1 --branch "$latest_tag" "https://github.com/$UPSTREAM_REPO" "$src"

assets=()
: > "$workdir/checksums.txt"

for arch in amd64 arm64; do
  out_bin="$workdir/amneziawg-go"
  ( cd "$src" && CGO_ENABLED=0 GOOS=linux GOARCH="$arch" \
      go build -trimpath -ldflags="-s -w" -o "$out_bin" . )

  out="amneziawg-go_${version}_linux_${arch}.tar.gz"
  chmod +x "$out_bin"
  tar -C "$workdir" -czf "$workdir/$out" amneziawg-go
  rm -f "$out_bin"

  (cd "$workdir" && sha256sum "$out" >> checksums.txt)
  assets+=("$workdir/$out")
done

gh release create "$our_tag" --repo "$REPO" \
  --title "amneziawg-go $latest_tag" \
  --notes "Static build of [amnezia-vpn/amneziawg-go](https://github.com/$UPSTREAM_REPO/tree/$latest_tag) ($latest_tag), the userspace AmneziaWG daemon. CGO-free, fully static, no kernel module required. Pair with amneziawg-tools: \`awg-quick\` auto-falls back to \`amneziawg-go\` when the kernel module is absent. linux/amd64 + linux/arm64." \
  "${assets[@]}" "$workdir/checksums.txt"

echo "Published $our_tag with ${#assets[@]} assets."
