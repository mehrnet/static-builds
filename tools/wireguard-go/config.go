package main

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
)

// Config is the on-disk shape radar-node's wireguard module passes via
// its {{params_json}} temp file -- deliberately close to the fields a
// wg-quick .conf already has (base64 keys, CIDR strings), not the raw
// UAPI wire format, since that's what anyone hand-authoring a probe's
// params already has lying around from their existing WireGuard setup.
type Config struct {
	PrivateKey          string   `json:"private_key"`
	Address             string   `json:"address"`               // e.g. "10.0.0.2/32" -- assigned to the tunnel interface
	ListenPort          int      `json:"listen_port,omitempty"` // 0 = let the kernel pick an ephemeral port
	MTU                 int      `json:"mtu,omitempty"`         // 0 = device default (1420)
	PeerPublicKey       string   `json:"peer_public_key"`
	PeerPresharedKey    string   `json:"peer_preshared_key,omitempty"`
	Endpoint            string   `json:"endpoint"` // "host:port"
	AllowedIPs          []string `json:"allowed_ips"`
	PersistentKeepalive int      `json:"persistent_keepalive,omitempty"`
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.PrivateKey == "" || cfg.PeerPublicKey == "" || cfg.Endpoint == "" || cfg.Address == "" {
		return nil, fmt.Errorf("config missing one of: private_key, address, peer_public_key, endpoint")
	}
	if len(cfg.AllowedIPs) == 0 {
		return nil, fmt.Errorf("config needs at least one allowed_ips entry -- there is no implicit 0.0.0.0/0 default here (see README: only the CIDRs listed get a route through the tunnel, on purpose)")
	}
	return &cfg, nil
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
