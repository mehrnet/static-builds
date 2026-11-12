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

- **Latest version:** `v26.11.9`
- **Last updated:** 2026-11-06T11:29:37Z
- **Release:** https://github.com/mehrnet/static-builds/releases/tag/xray-v26.11.9

| Platform | Download |
|---|---|
| darwin/amd64 | [xray_26.11.9_darwin_amd64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/xray-v26.11.9/xray_26.11.9_darwin_amd64.tar.gz) |
| darwin/arm64 | [xray_26.11.9_darwin_arm64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/xray-v26.11.9/xray_26.11.9_darwin_arm64.tar.gz) |
| linux/amd64 | [xray_26.11.9_linux_amd64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/xray-v26.11.9/xray_26.11.9_linux_amd64.tar.gz) |
| linux/arm | [xray_26.11.9_linux_arm.tar.gz](https://github.com/mehrnet/static-builds/releases/download/xray-v26.11.9/xray_26.11.9_linux_arm.tar.gz) |
| linux/arm64 | [xray_26.11.9_linux_arm64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/xray-v26.11.9/xray_26.11.9_linux_arm64.tar.gz) |
| windows/amd64 | [xray_26.11.9_windows_amd64.zip](https://github.com/mehrnet/static-builds/releases/download/xray-v26.11.9/xray_26.11.9_windows_amd64.zip) |
| windows/arm64 | [xray_26.11.9_windows_arm64.zip](https://github.com/mehrnet/static-builds/releases/download/xray-v26.11.9/xray_26.11.9_windows_arm64.zip) |

### openvpn

Static build of [OpenVPN/openvpn](https://github.com/OpenVPN/openvpn) against musl (Alpine), management interface and plugin loading disabled. linux/amd64 + linux/arm64 only.

- **Latest version:** `v2.7.9`
- **Last updated:** 2026-11-01T10:23:22Z
- **Release:** https://github.com/mehrnet/static-builds/releases/tag/openvpn-v2.7.9

| Platform | Download |
|---|---|
| linux/amd64 | [openvpn_2.7.9_linux_amd64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/openvpn-v2.7.9/openvpn_2.7.9_linux_amd64.tar.gz) |
| linux/arm64 | [openvpn_2.7.9_linux_arm64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/openvpn-v2.7.9/openvpn_2.7.9_linux_arm64.tar.gz) |

### wireguard-go

This repo's own `radar-wg` wrapper (`tools/wireguard-go`), vendoring upstream [wireguard-go](https://git.zx2c4.com/wireguard-go/) -- brings a userspace WireGuard tunnel up/down via the UAPI + netlink, no `wireguard-tools` required. linux/amd64 + linux/arm64 only.

- **Latest version:** `b3bc704ef0ee`
- **Last updated:** 2026-10-23T08:28:43Z
- **Release:** https://github.com/mehrnet/static-builds/releases/tag/wireguard-go-b3bc704ef0ee

| Platform | Download |
|---|---|
| linux/amd64 | [radar-wg_b3bc704ef0ee_linux_amd64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/wireguard-go-b3bc704ef0ee/radar-wg_b3bc704ef0ee_linux_amd64.tar.gz) |
| linux/arm64 | [radar-wg_b3bc704ef0ee_linux_arm64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/wireguard-go-b3bc704ef0ee/radar-wg_b3bc704ef0ee_linux_arm64.tar.gz) |

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

### accel-ppp

Portable accel-ppp build from [accel-ppp/accel-ppp](https://github.com/accel-ppp/accel-ppp) against musl (Alpine): daemon, accel-cmd, plugins, libtriton, musl loader and runtime libraries under bin/ and lib/. Host L2TP/PPP/xfrm kernel modules are still required for live sessions.

- **Latest version:** `1.15.0`
- **Last updated:** 2026-11-12T13:13:24Z
- **Release:** https://github.com/mehrnet/static-builds/releases/tag/accel-ppp-1.15.0

| Platform | Download |
|---|---|
| linux/amd64 | [accel-ppp_1.15.0_linux_amd64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/accel-ppp-1.15.0/accel-ppp_1.15.0_linux_amd64.tar.gz) |
| linux/arm64 | [accel-ppp_1.15.0_linux_arm64.tar.gz](https://github.com/mehrnet/static-builds/releases/download/accel-ppp-1.15.0/accel-ppp_1.15.0_linux_arm64.tar.gz) |

