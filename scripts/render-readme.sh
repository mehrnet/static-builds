#!/usr/bin/env bash
# Regenerates README.md from scratch by querying this repo's own
# releases API -- no separate state file to keep in sync, since a
# release's own tag/name/assets/publishedAt already say everything the
# table needs. Safe to run after any workflow (or none), on a schedule
# or by hand: if nothing changed upstream, the output is byte-identical
# and the calling workflow's own `git diff --cached --quiet` skips the
# commit.
set -euo pipefail

cd "$(dirname "$0")/.."

declare -A DESCRIPTIONS=(
  ["xray"]="Re-hosted static build of [XTLS/Xray-core](https://github.com/XTLS/Xray-core), unpacked from upstream's own official release assets -- not rebuilt from source."
  ["openvpn"]="Static build of [OpenVPN/openvpn](https://github.com/OpenVPN/openvpn) against musl (Alpine), management interface and plugin loading disabled. linux/amd64 + linux/arm64 only."
  ["wireguard-go"]="This repo's own \`radar-wg\` wrapper (\`tools/wireguard-go\`), vendoring upstream [wireguard-go](https://git.zx2c4.com/wireguard-go/) -- brings a userspace WireGuard tunnel up/down via the UAPI + netlink, no \`wireguard-tools\` required. linux/amd64 + linux/arm64 only."
)
# Ordered explicitly (bash associative arrays have no stable order).
TOOLS=(xray openvpn wireguard-go)

all_releases_json=$(gh release list --limit 200 --json tagName,name,createdAt,isDraft,isPrerelease)

render_tool() {
  local tool="$1"
  local prefix="${tool}-"

  local latest
  latest=$(echo "$all_releases_json" | jq -c --arg prefix "$prefix" '
    [.[] | select(.isDraft == false and (.tagName | startswith($prefix)))]
    | sort_by(.createdAt) | last // empty
  ')

  echo "### ${tool}"
  echo
  echo "${DESCRIPTIONS[$tool]}"
  echo

  if [ -z "$latest" ] || [ "$latest" = "null" ]; then
    echo "_Not built yet._"
    echo
    return
  fi

  local tag created
  tag=$(echo "$latest" | jq -r '.tagName')
  created=$(echo "$latest" | jq -r '.createdAt')
  local version="${tag#"$prefix"}"

  echo "- **Latest version:** \`${version}\`"
  echo "- **Last updated:** ${created}"
  echo "- **Release:** https://github.com/mehrnet/static-builds/releases/tag/${tag}"
  echo
  echo "| Platform | Download |"
  echo "|---|---|"

  gh release view "$tag" --json assets \
    --jq '.assets[] | select(.name | endswith("checksums.txt") | not) | [.name, .url] | @tsv' |
    sort |
    while IFS=$'\t' read -r name url; do
      # Derive a human platform label from our own <name>_<version>_<os>_<arch>.<ext>
      # convention -- anchored from the *end* of the filename (not a
      # leading ${tool}_ prefix), since the binary a tool ships isn't
      # always named the same as the tool itself (wireguard-go's own
      # binary is radar-wg, for instance).
      local platform
      platform=$(echo "$name" | sed -E 's/^.*_([a-z]+)_([a-z0-9]+)\.(tar\.gz|zip)$/\1\/\2/')
      echo "| ${platform} | [${name}](${url}) |"
    done
  echo
}

{
  cat <<'HEADER'
# static-builds

Statically-built third-party binaries for [radar-node](https://github.com/mehrnet/radar-node)'s
optional prober modules (xray, OpenVPN, WireGuard) -- fetched by
`install.sh`'s `--install-xray` / `--install-openvpn` / `--install-wireguard`
flags, never bundled into radar-node's own binary.

Each tool is checked against its own upstream daily (see
`.github/workflows/`) and re-published here under a consistent
`<name>_<version>_<os>_<arch>.<ext>` naming convention with a
`checksums.txt` alongside every release, regardless of whether upstream
itself ships one. Every workflow can also be triggered by hand from the
Actions tab.

This file is regenerated automatically after every workflow run
(`scripts/render-readme.sh`) -- edits here don't persist.

## Tools

HEADER

  for tool in "${TOOLS[@]}"; do
    render_tool "$tool"
  done
} > README.md

echo "README.md regenerated."
