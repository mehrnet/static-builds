package main

import (
	"bufio"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Config is the internal representation loadConfig produces from a
// wg-quick(8)-style .conf -- radar-wg deliberately speaks the exact
// same file format wg-quick does (not a bespoke JSON shape, not the
// raw UAPI wire format) so anyone with an existing WireGuard setup can
// point --config at the same .conf they already have, no translation
// step required either by hand or in radar's own UI (see openvpn.yaml
// for the equivalent choice on the OpenVPN side, which has always
// worked this way).
type Config struct {
	PrivateKey          string
	Address             string // e.g. "10.0.0.2/32" -- assigned to the tunnel interface
	ListenPort          int    // 0 = let the kernel pick an ephemeral port
	MTU                 int    // 0 = device default (1420)
	PeerPublicKey       string
	PeerPresharedKey    string
	Endpoint            string // "host:port"
	AllowedIPs          []string
	PersistentKeepalive int
}

// loadConfig reads and parses a wg-quick(8)-style .conf: [Interface]
// and [Peer] sections, "Key = Value" pairs, case-insensitive keys
// (same laxness wg-quick itself has). Only a single peer is supported
// -- radar-wg is a connectivity probe against one endpoint, not a
// general-purpose multi-peer client.
func loadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	defer f.Close()
	return parseConfig(f)
}

func parseConfig(r io.Reader) (*Config, error) {
	cfg := &Config{}
	section := ""
	sawPeer := false
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			if section == "peer" {
				if sawPeer {
					return nil, fmt.Errorf("config has more than one [Peer] section -- radar-wg only supports a single peer")
				}
				sawPeer = true
			}
			continue
		}
		eq := strings.Index(line, "=")
		if eq == -1 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:eq]))
		value := strings.TrimSpace(line[eq+1:])
		switch section {
		case "interface":
			switch key {
			case "privatekey":
				cfg.PrivateKey = value
			case "address":
				// wg-quick allows a comma-separated list of addresses;
				// radar-wg assigns a single address to the tunnel, so
				// only the first is used.
				cfg.Address = strings.TrimSpace(strings.SplitN(value, ",", 2)[0])
			case "listenport":
				cfg.ListenPort, _ = strconv.Atoi(value)
			case "mtu":
				cfg.MTU, _ = strconv.Atoi(value)
			}
		case "peer":
			switch key {
			case "publickey":
				cfg.PeerPublicKey = value
			case "presharedkey":
				cfg.PeerPresharedKey = value
			case "endpoint":
				cfg.Endpoint = value
			case "allowedips":
				for _, cidr := range strings.Split(value, ",") {
					cidr = strings.TrimSpace(cidr)
					if cidr != "" {
						cfg.AllowedIPs = append(cfg.AllowedIPs, cidr)
					}
				}
			case "persistentkeepalive":
				cfg.PersistentKeepalive, _ = strconv.Atoi(value)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	if cfg.PrivateKey == "" || cfg.PeerPublicKey == "" || cfg.Endpoint == "" || cfg.Address == "" {
		return nil, fmt.Errorf("config missing one of: PrivateKey, Address, [Peer] PublicKey, Endpoint")
	}
	if len(cfg.AllowedIPs) == 0 {
		return nil, fmt.Errorf("config needs at least one AllowedIPs entry -- there is no implicit 0.0.0.0/0 default here (see README: only the CIDRs listed get a route through the tunnel, on purpose)")
	}
	return cfg, nil
}

// base64KeyToHex converts a standard WireGuard base64 key (as found in
// any wg-quick .conf or `wg genkey` output) to the hex encoding the
// device.IpcSet UAPI wire format actually requires. Both are just
// encodings of the same 32 raw bytes.
func base64KeyToHex(b64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", fmt.Errorf("invalid base64 key: %w", err)
	}
	if len(raw) != 32 {
		return "", fmt.Errorf("key decodes to %d bytes, want 32", len(raw))
	}
	return hex.EncodeToString(raw), nil
}
