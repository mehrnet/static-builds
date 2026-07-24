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
radar-wg up   --config tunnel.conf --state /tmp/probe123.state
# ... run a normal tcp/http/icmp probe against a target inside AllowedIPs ...
radar-wg down --state /tmp/probe123.state
```

`up` exits as soon as the interface is configured and up; the tunnel
itself is a kernel interface, not something this process needs to stay
running to hold open. `down` is a separate, later invocation that reads
the interface name back from `--state` and deletes it.

## Config (`tunnel.conf`)

A normal `wg-quick(8)`-style config -- the exact same `.conf` you'd
already have from any WireGuard setup, no translation step needed:

```ini
[Interface]
PrivateKey = <base64, this node's own private key>
Address = 10.0.0.2/32

[Peer]
PublicKey = <base64, the peer's public key>
PresharedKey = <base64, optional>
Endpoint = 203.0.113.1:51820
AllowedIPs = 10.0.0.0/24
PersistentKeepalive = 25
```

Only `[Interface]`/`[Peer]` `PrivateKey`/`Address`/`PublicKey`/
`PresharedKey`/`Endpoint`/`AllowedIPs`/`PersistentKeepalive`/
`ListenPort`/`MTU` are read; anything else (`DNS`, `Table`,
`PostUp`/`PostDown`, ...) is ignored rather than rejected, since this
isn't a full `wg-quick` replacement -- just enough of the format to
configure one tunnel and one peer. Only a single `[Peer]` section is
supported; a config with more than one is rejected outright rather
than silently picking one.

`AllowedIPs` has **no implicit `0.0.0.0/0`** the way a real VPN client's
config normally would. Only the CIDRs listed get a route added, scoped to
this tunnel's own interface -- the node's own default route is never
touched. This is deliberate: this binary exists to let a probe test
reachability *through* a specific tunnel to a specific target, not to
become the node's general-purpose VPN egress. Point `AllowedIPs` (and
whatever you probe next) at just the target(s) you actually want to test.

A literal `AllowedIPs = 0.0.0.0/0` (or `::/0`) -- an ordinary full-tunnel
config, nothing exotic -- is handled the same way `wg-quick(8)` handles
it: split into two half-address-space routes (`0.0.0.0/1` + `128.0.0.0/1`,
or `::/1` + `8000::/1`) instead of added as a literal default route. Adding
the literal route would otherwise collide (`EEXIST`) with the node's own
already-existing default route; the split covers the identical address
space without touching it.
