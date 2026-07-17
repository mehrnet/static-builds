# radar-wg

A small wrapper around upstream [wireguard-go](https://git.zx2c4.com/wireguard-go/)
that brings a userspace WireGuard tunnel up and down entirely through the
UAPI and `netlink` -- no `wireguard-tools` (`wg`/`wg-quick`) needed on the
host at all. Built and published by `.github/workflows/wireguard-go.yml`;
see the repo root README for download links.

Requires `CAP_NET_ADMIN` (root, in practice) -- creating a TUN device and
adding routes both need it.

## Usage

```sh
radar-wg up   --config tunnel.json --state /tmp/probe123.state
# ... run a normal tcp/http/icmp probe against a target inside allowed_ips ...
radar-wg down --state /tmp/probe123.state
```

`up` exits as soon as the interface is configured and up; the tunnel
itself is a kernel interface, not something this process needs to stay
running to hold open. `down` is a separate, later invocation that reads
the interface name back from `--state` and deletes it.

## Config (`tunnel.json`)

```jsonc
{
  "private_key": "<base64, this node's own private key>",
  "address": "10.0.0.2/32",
  "peer_public_key": "<base64, the peer's public key>",
  "peer_preshared_key": "<base64, optional>",
  "endpoint": "203.0.113.1:51820",
  "allowed_ips": ["10.0.0.0/24"],
  "listen_port": 0,
  "persistent_keepalive": 25,
  "mtu": 1420
}
```

Keys are the same base64 encoding any `wg-quick` config or `wg genkey`
output already uses -- converted to the hex the UAPI wire format actually
requires internally, not something you need to convert yourself.

`allowed_ips` has **no implicit `0.0.0.0/0`** the way a real VPN client's
config normally would. Only the CIDRs listed get a route added, scoped to
this tunnel's own interface -- the node's own default route is never
touched. This is deliberate: this binary exists to let a probe test
reachability *through* a specific tunnel to a specific target, not to
become the node's general-purpose VPN egress. Point `allowed_ips` (and
whatever you probe next) at just the target(s) you actually want to test.
