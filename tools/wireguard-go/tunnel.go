package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
)

// State is what `up` writes and `down` reads back -- the interface name
// is kernel-assigned-ish (we pick it, but uniquely per invocation, see
// randomIfaceName) precisely so concurrent probes never collide on one
// shared name the way a fixed "wg0" would.
type State struct {
	Interface string    `json:"interface"`
	CreatedAt time.Time `json:"created_at"`
}

// up brings a userspace WireGuard tunnel online: creates a TUN device,
// configures it via the UAPI (private key, one peer, allowed IPs),
// assigns Address to it, adds a route for each AllowedIPs entry scoped
// to this interface, and brings it up. No route replaces or competes
// with the node's own default route -- only the CIDRs the caller
// explicitly listed in allowed_ips get routed through the tunnel, so a
// misconfigured or malicious peer config can't silently redirect this
// node's own general traffic. That's a deliberate scoping-down from
// what a real wg-quick "0.0.0.0/0" tunnel does, matching this being a
// probe/connectivity test, not a general-purpose VPN client.
func up(cfg *Config, statePath string) error {
	ifaceName, err := randomIfaceName()
	if err != nil {
		return fmt.Errorf("generate interface name: %w", err)
	}

	mtu := cfg.MTU
	if mtu <= 0 {
		mtu = device.DefaultMTU
	}
	tunDev, err := tun.CreateTUN(ifaceName, mtu)
	if err != nil {
		return fmt.Errorf("create tun %s (needs CAP_NET_ADMIN/root): %w", ifaceName, err)
	}
	actualName, _ := tunDev.Name()

	logger := device.NewLogger(device.LogLevelError, fmt.Sprintf("(%s) ", actualName))
	dev := device.NewDevice(tunDev, conn.NewDefaultBind(), logger)

	uapiConfig, err := buildUAPIConfig(cfg)
	if err != nil {
		_ = dev.Down()
		dev.Close()
		return fmt.Errorf("build UAPI config: %w", err)
	}
	if err := dev.IpcSet(uapiConfig); err != nil {
		dev.Close()
		return fmt.Errorf("configure device: %w", err)
	}
	if err := dev.Up(); err != nil {
		dev.Close()
		return fmt.Errorf("bring device up: %w", err)
	}
	// The device itself now runs its own goroutines independent of this
	// process's lifetime expectations -- deliberately NOT calling
	// dev.Close() on the success path. `down` is what tears this back
	// out, by interface name, from a separate invocation of this same
	// binary (see main.go: up and down are two short-lived CLI calls
	// wrapping one long-lived kernel interface, not one long-running
	// process holding it open).

	link, err := netlink.LinkByName(actualName)
	if err != nil {
		return fmt.Errorf("find created link %s: %w", actualName, err)
	}
	addr, err := netlink.ParseAddr(cfg.Address)
	if err != nil {
		return fmt.Errorf("parse address %q: %w", cfg.Address, err)
	}
	if err := netlink.AddrAdd(link, addr); err != nil {
		return fmt.Errorf("assign address %s to %s: %w", cfg.Address, actualName, err)
	}
	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("set %s up: %w", actualName, err)
	}
	for _, cidr := range cfg.AllowedIPs {
		dst, err := netlink.ParseIPNet(cidr)
		if err != nil {
			return fmt.Errorf("parse allowed_ips entry %q: %w", cidr, err)
		}
		route := &netlink.Route{LinkIndex: link.Attrs().Index, Dst: dst}
		if err := netlink.RouteAdd(route); err != nil {
			return fmt.Errorf("add route %s via %s: %w", cidr, actualName, err)
		}
	}

	return writeState(statePath, &State{Interface: actualName, CreatedAt: time.Now().UTC()})
}

// down tears down exactly the interface `up` created -- reads the
// interface name back from statePath rather than taking it on the
// command line, so a caller (radar-node's module `teardown:` step)
// only ever needs to remember one path, the same one `prepare:` passed
// to `up`.
func down(statePath string) error {
	st, err := readState(statePath)
	if err != nil {
		return err
	}
	link, err := netlink.LinkByName(st.Interface)
	if err != nil {
		// Already gone (e.g. a previous down already ran, or the node
		// rebooted) -- not an error worth failing a teardown step over.
		return nil
	}
	if err := netlink.LinkDel(link); err != nil {
		return fmt.Errorf("delete link %s: %w", st.Interface, err)
	}
	_ = os.Remove(statePath)
	return nil
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
	fmt.Fprintf(&b, "endpoint=%s\n", cfg.Endpoint)
	for _, cidr := range cfg.AllowedIPs {
		fmt.Fprintf(&b, "allowed_ip=%s\n", cidr)
	}
	if cfg.PersistentKeepalive > 0 {
		fmt.Fprintf(&b, "persistent_keepalive_interval=%d\n", cfg.PersistentKeepalive)
	}
	return b.String(), nil
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
