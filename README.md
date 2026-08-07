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

### xray

Re-hosted static build of [XTLS/Xray-core](https://github.com/XTLS/Xray-core), unpacked from upstream's own official release assets -- not rebuilt from source.

- **Latest version:** `v26.3.27`
- **Last updated:** 2026-07-17T10:53:43Z
- **Release:** https://github.com/mehrnet/static-builds/releases/tag/xray-v26.3.27

| Platform | Download |
|---|---|
| darwin/amd64 | [xray_26.3.27_darwin_amd64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/xray-v26.3.27/xray_26.3.27_darwin_amd64.tar.gz) |
| darwin/arm64 | [xray_26.3.27_darwin_arm64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/xray-v26.3.27/xray_26.3.27_darwin_arm64.tar.gz) |
| linux/amd64 | [xray_26.3.27_linux_amd64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/xray-v26.3.27/xray_26.3.27_linux_amd64.tar.gz) |
| linux/arm64 | [xray_26.3.27_linux_arm64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/xray-v26.3.27/xray_26.3.27_linux_arm64.tar.gz) |
| windows/amd64 | [xray_26.3.27_windows_amd64.zip](https://github.com/mehrnet/static-builds/releases/download/xray-v26.3.27/xray_26.3.27_windows_amd64.zip) |
| windows/arm64 | [xray_26.3.27_windows_arm64.zip](https://github.com/mehrnet/static-builds/releases/download/xray-v26.3.27/xray_26.3.27_windows_arm64.zip) |

### openvpn

Static build of [OpenVPN/openvpn](https://github.com/OpenVPN/openvpn) against musl (Alpine), management interface and plugin loading disabled. linux/amd64 + linux/arm64 only.

- **Latest version:** `v2.7.6`
- **Last updated:** 2026-07-25T05:51:42Z
- **Release:** https://github.com/mehrnet/static-builds/releases/tag/openvpn-v2.7.6

| Platform | Download |
|---|---|
| linux/amd64 | [openvpn_2.7.6_linux_amd64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/openvpn-v2.7.6/openvpn_2.7.6_linux_amd64.tar.gz) |
| linux/arm64 | [openvpn_2.7.6_linux_arm64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/openvpn-v2.7.6/openvpn_2.7.6_linux_arm64.tar.gz) |

### wireguard-go

This repo's own `radar-wg` wrapper (`tools/wireguard-go`), vendoring upstream [wireguard-go](https://git.zx2c4.com/wireguard-go/) -- brings a userspace WireGuard tunnel up/down via the UAPI + netlink, no `wireguard-tools` required. linux/amd64 + linux/arm64 only.

- **Latest version:** `ecfc5a8d5446-r10`
- **Last updated:** 2026-07-24T15:22:01Z
- **Release:** https://github.com/mehrnet/static-builds/releases/tag/wireguard-go-ecfc5a8d5446-r10

| Platform | Download |
|---|---|
| linux/amd64 | [radar-wg_ecfc5a8d5446-r10_linux_amd64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/wireguard-go-ecfc5a8d5446-r10/radar-wg_ecfc5a8d5446-r10_linux_amd64.tar.gz) |
| linux/arm64 | [radar-wg_ecfc5a8d5446-r10_linux_arm64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/wireguard-go-ecfc5a8d5446-r10/radar-wg_ecfc5a8d5446-r10_linux_arm64.tar.gz) |

