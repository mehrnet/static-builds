# radar-wg

A small wrapper around upstream [wireguard-go](https://git.zx2c4.com/wireguard-go/)
that brings a userspace WireGuard tunnel up, probes a target through it, and
tears it back down -- entirely through the UAPI and `netlink`, no
`wireguard-tools` (`wg`/`wg-quick`) needed on the host at all. Built and
published by `.github/workflows/wireguard-go.yml`; see the repo root README
for download links.

Requires `CAP_NET_ADMIN` (root, in practice) -- creating a TUN device and
adding routes/rules all need it.

## Usage

```sh
radar-wg run --config tunnel.conf --target 203.0.113.9:443 --timeout-ms 5000
```

One subcommand, one process, covering the whole lifecycle -- bring-up,
probe, teardown -- deliberately not split across separate CLI invocations.
wireguard-go here is a *userspace* implementation: the actual WireGuard
session (handshake, encrypt/decrypt) is Go goroutines running inside this
one process, not a kernel module, so it can't outlive the process the way
a kernel WireGuard interface would. An earlier "bring up and exit, run a
probe as a separate process, tear down as a third invocation" design never
actually worked for exactly that reason -- by the time the separate probe
process ran, the tunnel it depended on was already gone.

`run` prints one line of JSON to stdout: `{"latency_ms": ..., "http_code": ...}`,
the result of a `curl` request against `--target`, sourced from the
tunnel's own assigned address so it actually goes through the tunnel (see
"Concurrency and isolation" below) -- not radar-wg's own concern beyond
that; everything else (interpreting the result, retries, scheduling) is
radar-node's.

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
config normally would, and (unlike a real VPN client) never touches the
node's own default route or general traffic at all -- see "Concurrency and
isolation" below for how. A literal `AllowedIPs = 0.0.0.0/0` (an ordinary
full-tunnel config, nothing exotic) works fine and needs no special
handling on your end.

## Concurrency and isolation

Every route `run` adds goes into a routing table generated fresh for that
one invocation, reached only through a policy rule (`ip rule`) matching
traffic sourced from that invocation's own tunnel-assigned `Address` --
never the main table. `run`'s own `curl` probe binds to that same address
(`--interface`) so its traffic is exactly what matches. Two things fall
out of this:

- **The node's own default route is never touched or competed with.**
  Nothing but this one tunnel's own probe traffic is ever routed through
  it, so a literal `AllowedIPs = 0.0.0.0/0` needs no split-route trick or
  other special case the way it otherwise would.
- **Concurrent `radar-wg run` invocations on the same node never collide,**
  even with identical/overlapping `AllowedIPs` (e.g. two different
  full-tunnel probes at once) -- each gets its own table and rule, so
  neither's route-add can hit `EEXIST` against the other's, and one
  tunnel's traffic can never end up silently routed through a different,
  unrelated tunnel.

Teardown reverses all of it (link, routes, and the policy rule) before
`run` returns, whether the probe succeeded or not.
