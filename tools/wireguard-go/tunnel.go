package main

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
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

// bringUp creates and configures a userspace WireGuard tunnel: a TUN
// device, the UAPI session (private key, one peer, allowed IPs),
// Address assigned to the interface, and routes for every AllowedIPs
// entry in a routing table private to this one invocation (randomID)
// -- never the main table -- reached only via a matching "from Address
// lookup Table" policy rule (RuleAdd) scoped to this tunnel's own
// assigned Address. That's deliberate, and gets two things a plain
// main-table route can't:
//
//  1. Isolation from the node's own default route: nothing needs a
//     0.0.0.0/0-vs-existing-default-route special case (no split-route
//     trick) since this table is never consulted for the node's own
//     general traffic -- only for traffic actually sourced from this
//     tunnel's own address (which only run()'s own `curl --interface`
//     invocation uses).
//  2. Isolation between concurrent tunnels on the same node: two
//     probes with identical/overlapping AllowedIPs (e.g. both a full
//     0.0.0.0/0 tunnel) each get their own table + rule, so neither's
//     route-add can collide (EEXIST) with the other's, and traffic
//     from one tunnel's address can never get silently routed through
//     a different, unrelated tunnel the way two same-prefix main-table
//     routes could.
//
// Returns a teardown func that undoes exactly what this call did, and
// nothing else -- see run()'s own doc comment for why bring-up, the
// actual probe, and this teardown all have to happen within one
// single process's lifetime rather than split across separate CLI
// invocations.
func bringUp(cfg *Config) (localAddr string, teardown func(), err error) {
	ifaceName, err := randomIfaceName()
	if err != nil {
		return "", nil, fmt.Errorf("generate interface name: %w", err)
	}

	mtu := cfg.MTU
	if mtu <= 0 {
		mtu = device.DefaultMTU
	}
	tunDev, err := tun.CreateTUN(ifaceName, mtu)
	if err != nil {
		return "", nil, fmt.Errorf("create tun %s (needs CAP_NET_ADMIN/root): %w", ifaceName, err)
	}
	actualName, _ := tunDev.Name()

	logger := stderrLogger(device.LogLevelError, fmt.Sprintf("(%s) ", actualName))
	dev := device.NewDevice(tunDev, conn.NewDefaultBind(), logger)

	uapiConfig, err := buildUAPIConfig(cfg)
	if err != nil {
		dev.Close()
		return "", nil, fmt.Errorf("build UAPI config: %w", err)
	}
	if err := dev.IpcSet(uapiConfig); err != nil {
		dev.Close()
		return "", nil, fmt.Errorf("configure device: %w", err)
	}
	if err := dev.Up(); err != nil {
		dev.Close()
		return "", nil, fmt.Errorf("bring device up: %w", err)
	}

	link, err := netlink.LinkByName(actualName)
	if err != nil {
		dev.Close()
		return "", nil, fmt.Errorf("find created link %s: %w", actualName, err)
	}
	addr, err := netlink.ParseAddr(cfg.Address)
	if err != nil {
		dev.Close()
		return "", nil, fmt.Errorf("parse address %q: %w", cfg.Address, err)
	}
	if err := netlink.AddrAdd(link, addr); err != nil {
		dev.Close()
		return "", nil, fmt.Errorf("assign address %s to %s: %w", cfg.Address, actualName, err)
	}
	if err := netlink.LinkSetUp(link); err != nil {
		dev.Close()
		return "", nil, fmt.Errorf("set %s up: %w", actualName, err)
	}

	table, err := randomID(10000, 1<<28)
	if err != nil {
		dev.Close()
		return "", nil, fmt.Errorf("generate routing table id: %w", err)
	}
	for _, cidr := range cfg.AllowedIPs {
		dst, err := netlink.ParseIPNet(cidr)
		if err != nil {
			dev.Close()
			return "", nil, fmt.Errorf("parse allowed_ips entry %q: %w", cidr, err)
		}
		route := &netlink.Route{LinkIndex: link.Attrs().Index, Dst: dst, Table: table}
		if err := netlink.RouteAdd(route); err != nil {
			dev.Close()
			return "", nil, fmt.Errorf("add route %s via %s (table %d): %w", cidr, actualName, table, err)
		}
	}

	// Below "local" (0) and well clear of "main" (32766)/"default"
	// (32767) so this rule is always consulted before the node's
	// normal routing, for the one specific source address it matches.
	priority, err := randomID(10000, 20000)
	if err != nil {
		dev.Close()
		return "", nil, fmt.Errorf("generate rule priority: %w", err)
	}
	// Always a single host (/32 or /128), regardless of whatever mask
	// Address itself declared (a wg-quick .conf's Address commonly
	// carries a wider subnet mask than /32, e.g. "10.0.0.2/24") -- this
	// rule's job is to catch traffic sourced from this one tunnel's own
	// address specifically, not an entire subnet.
	ruleSrc := hostOnlyNet(addr.IPNet)
	rule := netlink.NewRule()
	rule.Src = ruleSrc
	rule.Table = table
	rule.Priority = priority
	if err := netlink.RuleAdd(rule); err != nil {
		dev.Close()
		return "", nil, fmt.Errorf("add policy rule (from %s table %d): %w", ruleSrc, table, err)
	}

	teardown = func() {
		// dev.Close() first, always -- it stops every one of the
		// device's own background goroutines (packet read/write loop,
		// handshake timers, periodic MTU checks, ...), which otherwise
		// keep running against a TUN device that's about to disappear
		// out from under them. Tearing down netlink state first left
		// them tripping over a device that had already vanished
		// ("Failed to load updated MTU of device: ... no such device",
		// "read /dev/net/tun: not pollable") -- confusing at best.
		//
		// It also means the interface itself is typically already gone
		// by the time this returns: without TUNSETPERSIST (deliberately
		// never set -- this tunnel has no business outliving this one
		// process), closing the TUN fd that created it tears the kernel
		// interface down as a side effect. So the LinkDel below almost
		// always finds nothing left to delete -- expected, not logged;
		// it only exists as a defensive fallback for whatever edge case
		// leaves the interface behind anyway.
		dev.Close()
		if err := netlink.RuleDel(rule); err != nil {
			fmt.Fprintf(os.Stderr, "radar-wg: delete policy rule (table %d): %v\n", table, err)
		}
		if err := netlink.LinkDel(link); err != nil && !errors.Is(err, unix.ENODEV) {
			fmt.Fprintf(os.Stderr, "radar-wg: delete link %s: %v\n", actualName, err)
		}
	}
	return ruleSrc.IP.String(), teardown, nil
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

// hostOnlyNet narrows ipnet down to a single-host network (its IP with
// a /32 or /128 mask, IPv4 vs IPv6 chosen by which form the IP parses
// as), discarding whatever wider mask it may have originally carried.
func hostOnlyNet(ipnet *net.IPNet) *net.IPNet {
	if ip4 := ipnet.IP.To4(); ip4 != nil {
		return &net.IPNet{IP: ip4, Mask: net.CIDRMask(32, 32)}
	}
	return &net.IPNet{IP: ipnet.IP, Mask: net.CIDRMask(128, 128)}
}

// randomID returns a random int in [min, max) via crypto/rand, used
// for both the private routing table id and the policy rule priority
// `up` generates fresh per invocation -- see its own doc comment for
// why each invocation needs its own unique pick of both rather than a
// shared fixed value.
func randomID(min, max uint32) (int, error) {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return 0, err
	}
	return int(min + binary.BigEndian.Uint32(buf)%(max-min)), nil
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

// randomIfaceName picks a short, unique-enough name so N probes can
// each bring up their own tunnel concurrently without colliding on a
// shared fixed name like "wg0" -- Linux interface names are capped at
// 15 bytes, hence the short hex suffix rather than a full random ID.
func randomIfaceName() (string, error) {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "rwg" + hex.EncodeToString(buf), nil
}
