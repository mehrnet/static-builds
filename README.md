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

- **Latest version:** `v26.7.28`
- **Last updated:** 2026-08-17T16:33:10Z
- **Release:** https://github.com/mehrnet/static-builds/releases/tag/xray-v26.7.28

| Platform | Download |
|---|---|
| darwin/amd64 | [xray_26.7.28_darwin_amd64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/xray-v26.7.28/xray_26.7.28_darwin_amd64.tar.gz) |
| darwin/arm64 | [xray_26.7.28_darwin_arm64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/xray-v26.7.28/xray_26.7.28_darwin_arm64.tar.gz) |
| linux/amd64 | [xray_26.7.28_linux_amd64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/xray-v26.7.28/xray_26.7.28_linux_amd64.tar.gz) |
| linux/arm64 | [xray_26.7.28_linux_arm64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/xray-v26.7.28/xray_26.7.28_linux_arm64.tar.gz) |
| windows/amd64 | [xray_26.7.28_windows_amd64.zip](https://github.com/mehrnet/static-builds/releases/download/xray-v26.7.28/xray_26.7.28_windows_amd64.zip) |
| windows/arm64 | [xray_26.7.28_windows_arm64.zip](https://github.com/mehrnet/static-builds/releases/download/xray-v26.7.28/xray_26.7.28_windows_arm64.zip) |

### openvpn

Static build of [OpenVPN/openvpn](https://github.com/OpenVPN/openvpn) against musl (Alpine), management interface and plugin loading disabled. linux/amd64 + linux/arm64 only.

- **Latest version:** `v2.7.7`
- **Last updated:** 2026-08-18T03:56:08Z
- **Release:** https://github.com/mehrnet/static-builds/releases/tag/openvpn-v2.7.7

| Platform | Download |
|---|---|
| linux/amd64 | [openvpn_2.7.7_linux_amd64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/openvpn-v2.7.7/openvpn_2.7.7_linux_amd64.tar.gz) |
| linux/arm64 | [openvpn_2.7.7_linux_arm64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/openvpn-v2.7.7/openvpn_2.7.7_linux_arm64.tar.gz) |

### wireguard-go

This repo's own `radar-wg` wrapper (`tools/wireguard-go`), vendoring upstream [wireguard-go](https://git.zx2c4.com/wireguard-go/) -- brings a userspace WireGuard tunnel up/down via the UAPI + netlink, no `wireguard-tools` required. linux/amd64 + linux/arm64 only.

- **Latest version:** `2631ce99a06f`
- **Last updated:** 2026-09-09T07:50:05Z
- **Release:** https://github.com/mehrnet/static-builds/releases/tag/wireguard-go-2631ce99a06f

| Platform | Download |
|---|---|
| linux/amd64 | [radar-wg_2631ce99a06f_linux_amd64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/wireguard-go-2631ce99a06f/radar-wg_2631ce99a06f_linux_amd64.tar.gz) |
| linux/arm64 | [radar-wg_2631ce99a06f_linux_arm64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/wireguard-go-2631ce99a06f/radar-wg_2631ce99a06f_linux_arm64.tar.gz) |

### amneziawg-go

Static build of upstream [amnezia-vpn/amneziawg-go](https://github.com/amnezia-vpn/amneziawg-go), the userspace AmneziaWG daemon (a wireguard-go fork with DPI-evading obfuscation). CGO-free, fully static, no kernel module required. linux/amd64 + linux/arm64.

- **Latest version:** `v3.1.20260828`
- **Last updated:** 2026-09-09T07:47:13Z
- **Release:** https://github.com/mehrnet/static-builds/releases/tag/amneziawg-go-v3.1.20260828

| Platform | Download |
|---|---|
| linux/amd64 | [amneziawg-go_3.1.20260828_linux_amd64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/amneziawg-go-v3.1.20260828/amneziawg-go_3.1.20260828_linux_amd64.tar.gz) |
| linux/arm64 | [amneziawg-go_3.1.20260828_linux_arm64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/amneziawg-go-v3.1.20260828/amneziawg-go_3.1.20260828_linux_arm64.tar.gz) |

### amneziawg-tools

Static build of [amnezia-vpn/amneziawg-tools](https://github.com/amnezia-vpn/amneziawg-tools) against musl (Alpine): the `awg(8)` CLI plus `awg-quick(8)`, which auto-falls back to `amneziawg-go` when the kernel module is absent. linux/amd64 + linux/arm64.

- **Latest version:** `v3.1.20260812`
- **Last updated:** 2026-09-09T07:48:01Z
- **Release:** https://github.com/mehrnet/static-builds/releases/tag/amneziawg-tools-v3.1.20260812

| Platform | Download |
|---|---|
| linux/amd64 | [amneziawg-tools_3.1.20260812_linux_amd64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/amneziawg-tools-v3.1.20260812/amneziawg-tools_3.1.20260812_linux_amd64.tar.gz) |
| linux/arm64 | [amneziawg-tools_3.1.20260812_linux_arm64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/amneziawg-tools-v3.1.20260812/amneziawg-tools_3.1.20260812_linux_arm64.tar.gz) |

