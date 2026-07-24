# radar-wg

A small wrapper around upstream [wireguard-go](https://git.zx2c4.com/wireguard-go/)
that brings a userspace WireGuard tunnel up, probes a target through it, and
tears it back down -- all in one process, all in memory, via
[`tun/netstack`](https://pkg.go.dev/golang.zx2c4.com/wireguard/tun/netstack)
(a gVisor-backed virtual network stack), the same approach Xray's own
WireGuard outbound uses. No `wireguard-tools` (`wg`/`wg-quick`) needed on the
host, no kernel TUN device, no kernel routes or policy rules, and **no
`CAP_NET_ADMIN`/root required at all**. Built and published by
`.github/workflows/wireguard-go.yml`; see the repo root README for download
links.

An earlier version of this tool used a real kernel TUN device + a private
routing table + an `ip rule` to scope traffic into it. That turned out
fragile in practice for what's fundamentally a one-shot connectivity probe:
DNS-hostname endpoints the UAPI can't resolve itself, a literal `0.0.0.0/0`
route colliding with the node's own default route, teardown-ordering races
against the device's own background goroutines, all needing their own
fixes. None of that exists here -- `netstack` is a private, in-memory
network stack scoped to this one process, with no kernel footprint to
manage or race against at all.

## Usage

```sh
radar-wg run --config tunnel.conf --target 203.0.113.9:443 --timeout-ms 5000
```

One subcommand, one process, covering the whole lifecycle -- bring-up,
probe, teardown -- deliberately not split across separate CLI invocations.
wireguard-go here is a *userspace* implementation: the actual WireGuard
session (handshake, encrypt/decrypt) is Go goroutines running inside this
one process, not a kernel module, so it can't outlive the process the way
a kernel WireGuard interface would.

`run` prints one line of JSON to stdout: `{"latency_ms": ..., "http_code": ...}`,
the result of an HTTP GET against `--target`, dialed through the tunnel
(`tnet.DialContext`, not a real OS socket) -- not radar-wg's own concern
beyond that; everything else (interpreting the result, retries, scheduling)
is radar-node's.

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

`AllowedIPs` is enforced the normal WireGuard way, unchanged from a real
kernel interface: it's the device layer (not `netstack`) that encrypts an
outbound packet under whichever peer's `AllowedIPs` covers its
destination, and drops it if none do. A literal `AllowedIPs = 0.0.0.0/0`
(an ordinary full-tunnel config) works fine and needs no special handling.

## Concurrency and isolation

Each `radar-wg run` invocation gets its own private, in-memory `netstack`
instance -- there's no shared kernel state at all (no interface, no
routing table, no policy rule) for two concurrent invocations to collide
over, even with identical/overlapping `AllowedIPs` (e.g. two different
full-tunnel probes on the same node at once). Teardown is just stopping
this one process's own goroutines (`dev.Close()`) before `run` returns,
whether the probe succeeded or not -- there's nothing external left behind
to clean up either way.
