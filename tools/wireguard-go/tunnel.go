package main

import (
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"strings"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun/netstack"
)

// stderrLogger is device.NewLogger's own logic, verbatim, except
// targeting os.Stderr instead of the library's hardcoded os.Stdout.
// run()'s final result is a single line of JSON on stdout that
// radar-node's collector parses directly -- any of wireguard-go's own
// internal log lines landing on that same stream (which
// device.NewLogger does by design, its own doc comment says exactly
// that) silently corrupts it, breaking collection even on an
// otherwise-successful probe. This exists so the device's *diagnostic*
// output never shares a stream with our *result* output.
func stderrLogger(level int, prepend string) *device.Logger {
	logger := &device.Logger{Verbosef: device.DiscardLogf, Errorf: device.DiscardLogf}
	logf := func(prefix string) func(string, ...any) {
		return log.New(os.Stderr, prefix+": "+prepend, log.Ldate|log.Ltime).Printf
	}
	if level >= device.LogLevelVerbose {
		logger.Verbosef = logf("DEBUG")
	}
	if level >= device.LogLevelError {
		logger.Errorf = logf("ERROR")
	}
	return logger
}

// bringUp creates and configures a userspace WireGuard tunnel entirely
// in-process, via wireguard-go's own tun/netstack (a gVisor-backed
// virtual network stack) rather than a real kernel TUN device. This
// replaced an earlier design built on a real kernel interface + a
// private routing table + an ip rule scoping traffic into it -- that
// approach needed CAP_NET_ADMIN, competed (however carefully) with the
// node's own kernel networking state, and turned out fragile in
// practice (DNS-hostname endpoints the UAPI can't resolve itself, a
// literal 0.0.0.0/0 route colliding with the node's own default route,
// teardown-ordering races against the device's own background
// goroutines, ...). None of that exists here: netstack is a private,
// in-memory network stack with no kernel footprint at all -- no TUN
// device, no interface name, no routes, no policy rules, no root
// required. This is the same approach Xray's own WireGuard outbound
// uses (see golang.zx2c4.com/wireguard/tun/netstack), not something
// bespoke to radar-wg.
//
// AllowedIPs enforcement is unaffected by any of this: it's the
// wireguard-go *device* layer (unchanged, same as the kernel-TUN
// design) that encrypts an outbound packet under whichever peer's
// AllowedIPs trie covers its destination, and drops it if none do --
// netstack only changes how packets get into and out of that layer.
//
// Returns the dialer probe() uses to actually reach target through
// the tunnel, and a teardown func -- see run()'s own doc comment for
// why bring-up, the probe, and teardown all happen within one single
// process's lifetime rather than split across separate invocations.
func bringUp(cfg *Config) (tnet *netstack.Net, teardown func(), err error) {
	localAddr, _, _ := strings.Cut(cfg.Address, "/") // Address may carry a CIDR mask (e.g. "10.0.0.2/32"); netstack wants the bare IP
	addr, err := netip.ParseAddr(localAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("parse address %q: %w", cfg.Address, err)
	}

	mtu := cfg.MTU
	if mtu <= 0 {
		mtu = device.DefaultMTU
	}
	tunDev, tnet, err := netstack.CreateNetTUN([]netip.Addr{addr}, nil, mtu)
	if err != nil {
		return nil, nil, fmt.Errorf("create netstack tun: %w", err)
	}

	logger := stderrLogger(device.LogLevelError, "")
	dev := device.NewDevice(tunDev, conn.NewDefaultBind(), logger)

	uapiConfig, err := buildUAPIConfig(cfg)
	if err != nil {
		dev.Close()
		return nil, nil, fmt.Errorf("build UAPI config: %w", err)
	}
	if err := dev.IpcSet(uapiConfig); err != nil {
		dev.Close()
		return nil, nil, fmt.Errorf("configure device: %w", err)
	}
	if err := dev.Up(); err != nil {
		dev.Close()
		return nil, nil, fmt.Errorf("bring device up: %w", err)
	}

	return tnet, dev.Close, nil
}

func buildUAPIConfig(cfg *Config) (string, error) {
	privHex, err := base64KeyToHex(cfg.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("private_key: %w", err)
	}
	pubHex, err := base64KeyToHex(cfg.PeerPublicKey)
	if err != nil {
		return "", fmt.Errorf("peer_public_key: %w", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "private_key=%s\n", privHex)
	if cfg.ListenPort > 0 {
		fmt.Fprintf(&b, "listen_port=%d\n", cfg.ListenPort)
	}
	fmt.Fprintf(&b, "public_key=%s\n", pubHex)
	if cfg.PeerPresharedKey != "" {
		pskHex, err := base64KeyToHex(cfg.PeerPresharedKey)
		if err != nil {
			return "", fmt.Errorf("peer_preshared_key: %w", err)
		}
		fmt.Fprintf(&b, "preshared_key=%s\n", pskHex)
	}
	resolvedEndpoint, err := resolveEndpoint(cfg.Endpoint)
	if err != nil {
		return "", fmt.Errorf("endpoint: %w", err)
	}
	fmt.Fprintf(&b, "endpoint=%s\n", resolvedEndpoint)
	for _, cidr := range cfg.AllowedIPs {
		fmt.Fprintf(&b, "allowed_ip=%s\n", cidr)
	}
	if cfg.PersistentKeepalive > 0 {
		fmt.Fprintf(&b, "persistent_keepalive_interval=%d\n", cfg.PersistentKeepalive)
	}
	return b.String(), nil
}

// resolveEndpoint turns a possibly-hostname Endpoint ("host:port" or
// "[ipv6]:port") into one with a literal IP, resolving via DNS if
// needed. The UAPI's own "endpoint=" key (dev.IpcSet) only accepts a
// literal address -- it has no resolver of its own, unlike wg-quick(8)
// or `wg setconf`, both of which resolve hostnames client-side before
// ever touching the kernel/UAPI. A bare wg-quick .conf commonly has a
// hostname here (that's the whole point of DDNS-backed WireGuard
// endpoints), so radar-wg has to do that same resolution itself or
// every such config fails with an opaque "unexpected character" parse
// error from the UAPI layer instead of a real DNS failure.
func resolveEndpoint(endpoint string) (string, error) {
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid endpoint %q: %w", endpoint, err)
	}
	if ip := net.ParseIP(host); ip != nil {
		return endpoint, nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return "", fmt.Errorf("resolve endpoint host %q: %w", host, err)
	}
	// Prefer an IPv4 result if one exists, matching wg-quick's own
	// resolver preference -- most WireGuard endpoints are IPv4-only,
	// and a stray AAAA record on a host that isn't actually reachable
	// over IPv6 shouldn't win by dumb luck of DNS answer ordering.
	chosen := ips[0]
	for _, ip := range ips {
		if ip4 := ip.To4(); ip4 != nil {
			chosen = ip4
			break
		}
	}
	return net.JoinHostPort(chosen.String(), port), nil
}
