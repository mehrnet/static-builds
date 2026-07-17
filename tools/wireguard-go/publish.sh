#!/usr/bin/env bash
# Builds radar-wg (this directory's own Go source, vendoring upstream
# wireguard-go) for linux/amd64 + linux/arm64 and publishes it under
# this repo's naming convention. Linux-only isn't a scoping choice here
# the way it is for openvpn -- github.com/vishvananda/netlink only
# targets Linux at all, so there's no macOS/Windows build to skip.
#
# Tracks upstream golang.zx2c4.com/wireguard's latest commit (it has no
# regular tagged releases), not a release tag -- "the version" for this
# tool is really "which commit of wireguard-go is vendored right now",
# recorded via `go list -m` after `go get -u` pulls the latest.
#
# Idempotent the same way the other two tools' publish.sh are: this
# repo's own release tags are the "already built this" record.
set -euo pipefail

REPO="mehrnet/static-builds"

NAME="wireguard-go"
cd tools/wireguard-go

go get -u golang.zx2c4.com/wireguard@latest
go mod tidy

commit_info=$(go list -m -f '{{.Version}}' golang.zx2c4.com/wireguard)
# Pseudo-version form is v0.0.0-<timestamp>-<12-char-commit>; the trailing
# segment is what we actually want as a short, stable version label.
short_sha="${commit_info##*-}"
our_tag="${NAME}-${short_sha}"

if gh release view "$our_tag" --repo "$REPO" >/dev/null 2>&1; then
  echo "Already published $our_tag -- nothing to do."
  # Still worth committing a go.mod/go.sum bump if `go get -u` moved
  # them but this exact commit was already released under a different
  # invocation's tag somehow -- in practice this should be a no-op diff.
  exit 0
fi

echo "Building/publishing $our_tag (wireguard-go @ $commit_info)..."

workdir=$(mktemp -d)
trap 'rm -rf "$workdir"' EXIT

assets=()
: > "$workdir/checksums.txt"

for arch in amd64 arm64; do
  out_bin="$workdir/radar-wg"
  CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath -ldflags="-s -w" -o "$out_bin" .

  out="radar-wg_${short_sha}_linux_${arch}.tar.gz"
  chmod +x "$out_bin"
  tar -C "$workdir" -czf "$workdir/$out" radar-wg
  rm -f "$out_bin"

  (cd "$workdir" && sha256sum "$out" >> checksums.txt)
  assets+=("$workdir/$out")
done

gh release create "$our_tag" --repo "$REPO" \
  --title "wireguard-go wrapper @ $short_sha" \
  --notes "Static build of this repo's own radar-wg (tools/wireguard-go), vendoring [golang.zx2c4.com/wireguard]($commit_info) at commit $short_sha. Brings a userspace WireGuard tunnel up/down via the UAPI + netlink, no wireguard-tools required. linux/amd64 + linux/arm64 only (netlink is Linux-only)." \
  "${assets[@]}" "$workdir/checksums.txt"

echo "Published $our_tag with ${#assets[@]} assets."

# Commit the go.mod/go.sum bump from `go get -u` so the repo's own
# history reflects exactly which wireguard-go commit this release built
# against -- the calling workflow does the actual git commit/push (same
# as it already does for the README), this just stages the files.
cd - >/dev/null
git add tools/wireguard-go/go.mod tools/wireguard-go/go.sum
