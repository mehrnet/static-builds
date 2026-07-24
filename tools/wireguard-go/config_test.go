package main

import (
	"strings"
	"testing"
)

func TestParseConfigBasic(t *testing.T) {
	conf := `[Interface]
PrivateKey = aGVsbG8td29ybGQtcHJpdmF0ZS1rZXk=
Address = 10.0.0.2/32
DNS = 1.1.1.1

[Peer]
PublicKey = cGVlci1wdWJsaWMta2V5LWhlcmU=
PresharedKey = cHNrLWhlcmU=
Endpoint = 203.0.113.5:51820
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
`
	cfg, err := parseConfig(strings.NewReader(conf))
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if cfg.PrivateKey != "aGVsbG8td29ybGQtcHJpdmF0ZS1rZXk=" {
		t.Errorf("PrivateKey = %q", cfg.PrivateKey)
	}
	if cfg.Address != "10.0.0.2/32" {
		t.Errorf("Address = %q", cfg.Address)
	}
	if len(cfg.DNS) != 1 || cfg.DNS[0] != "1.1.1.1" {
		t.Errorf("DNS = %v", cfg.DNS)
	}
	if cfg.PeerPublicKey != "cGVlci1wdWJsaWMta2V5LWhlcmU=" {
		t.Errorf("PeerPublicKey = %q", cfg.PeerPublicKey)
	}
	if cfg.PeerPresharedKey != "cHNrLWhlcmU=" {
		t.Errorf("PeerPresharedKey = %q", cfg.PeerPresharedKey)
	}
	if cfg.Endpoint != "203.0.113.5:51820" {
		t.Errorf("Endpoint = %q", cfg.Endpoint)
	}
	if len(cfg.AllowedIPs) != 2 || cfg.AllowedIPs[0] != "0.0.0.0/0" || cfg.AllowedIPs[1] != "::/0" {
		t.Errorf("AllowedIPs = %v", cfg.AllowedIPs)
	}
	if cfg.PersistentKeepalive != 25 {
		t.Errorf("PersistentKeepalive = %d", cfg.PersistentKeepalive)
	}
}

func TestParseConfigMissingRequired(t *testing.T) {
	conf := `[Interface]
Address = 10.0.0.2/32

[Peer]
Endpoint = 203.0.113.5:51820
AllowedIPs = 0.0.0.0/0
`
	if _, err := parseConfig(strings.NewReader(conf)); err == nil {
		t.Fatal("expected an error for a config missing PrivateKey/PublicKey")
	}
}

func TestParseConfigMissingAllowedIPs(t *testing.T) {
	conf := `[Interface]
PrivateKey = aGVsbG8td29ybGQtcHJpdmF0ZS1rZXk=
Address = 10.0.0.2/32

[Peer]
PublicKey = cGVlci1wdWJsaWMta2V5LWhlcmU=
Endpoint = 203.0.113.5:51820
`
	if _, err := parseConfig(strings.NewReader(conf)); err == nil {
		t.Fatal("expected an error for a config with no AllowedIPs")
	}
}

func TestParseConfigMultiplePeersRejected(t *testing.T) {
	conf := `[Interface]
PrivateKey = aGVsbG8td29ybGQtcHJpdmF0ZS1rZXk=
Address = 10.0.0.2/32

[Peer]
PublicKey = cGVlci1wdWJsaWMta2V5LWhlcmU=
Endpoint = 203.0.113.5:51820
AllowedIPs = 0.0.0.0/0

[Peer]
PublicKey = YW5vdGhlci1wZWVyLXB1YmxpYy1rZXk=
Endpoint = 203.0.113.6:51820
AllowedIPs = 10.0.0.0/24
`
	if _, err := parseConfig(strings.NewReader(conf)); err == nil {
		t.Fatal("expected an error for a config with more than one [Peer] section")
	}
}

func TestParseConfigAddressTakesOnlyFirst(t *testing.T) {
	conf := `[Interface]
PrivateKey = aGVsbG8td29ybGQtcHJpdmF0ZS1rZXk=
Address = 10.0.0.2/32, fd00::2/128

[Peer]
PublicKey = cGVlci1wdWJsaWMta2V5LWhlcmU=
Endpoint = 203.0.113.5:51820
AllowedIPs = 0.0.0.0/0
`
	cfg, err := parseConfig(strings.NewReader(conf))
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if cfg.Address != "10.0.0.2/32" {
		t.Errorf("Address = %q, want only the first entry", cfg.Address)
	}
}

func TestParseConfigDNSAbsentByDefault(t *testing.T) {
	conf := `[Interface]
PrivateKey = aGVsbG8td29ybGQtcHJpdmF0ZS1rZXk=
Address = 10.0.0.2/32

[Peer]
PublicKey = cGVlci1wdWJsaWMta2V5LWhlcmU=
Endpoint = 203.0.113.5:51820
AllowedIPs = 0.0.0.0/0
`
	cfg, err := parseConfig(strings.NewReader(conf))
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if len(cfg.DNS) != 0 {
		t.Errorf("DNS = %v, want empty when the .conf declares none", cfg.DNS)
	}
}

func TestParseConfigDNSMultiple(t *testing.T) {
	conf := `[Interface]
PrivateKey = aGVsbG8td29ybGQtcHJpdmF0ZS1rZXk=
Address = 10.0.0.2/32
DNS = 1.1.1.1, 8.8.8.8

[Peer]
PublicKey = cGVlci1wdWJsaWMta2V5LWhlcmU=
Endpoint = 203.0.113.5:51820
AllowedIPs = 0.0.0.0/0
`
	cfg, err := parseConfig(strings.NewReader(conf))
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if len(cfg.DNS) != 2 || cfg.DNS[0] != "1.1.1.1" || cfg.DNS[1] != "8.8.8.8" {
		t.Errorf("DNS = %v", cfg.DNS)
	}
}

func TestBase64KeyToHex(t *testing.T) {
	// 32 zero bytes, base64-encoded.
	if _, err := base64KeyToHex("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="); err != nil {
		t.Fatalf("base64KeyToHex: %v", err)
	}
	if _, err := base64KeyToHex("not-valid-base64!!"); err == nil {
		t.Fatal("expected an error for invalid base64")
	}
	if _, err := base64KeyToHex("aGVsbG8="); err == nil {
		t.Fatal("expected an error for a key that isn't 32 bytes")
	}
}
