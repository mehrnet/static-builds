package main

import (
	"net"
	"strings"
	"testing"
)

func TestHostOnlyNetIPv4NarrowsWiderMask(t *testing.T) {
	_, ipnet, err := net.ParseCIDR("10.0.0.2/24")
	if err != nil {
		t.Fatalf("ParseCIDR: %v", err)
	}
	ipnet.IP = net.ParseIP("10.0.0.2") // ParseCIDR masks IP to the network address; restore the host bits
	got := hostOnlyNet(ipnet)
	if got.String() != "10.0.0.2/32" {
		t.Errorf("got %s, want 10.0.0.2/32", got)
	}
}

func TestHostOnlyNetIPv6(t *testing.T) {
	ipnet := &net.IPNet{IP: net.ParseIP("fd00::2"), Mask: net.CIDRMask(64, 128)}
	got := hostOnlyNet(ipnet)
	if got.String() != "fd00::2/128" {
		t.Errorf("got %s, want fd00::2/128", got)
	}
}

func TestRandomIDWithinRange(t *testing.T) {
	for i := 0; i < 1000; i++ {
		got, err := randomID(10000, 20000)
		if err != nil {
			t.Fatalf("randomID: %v", err)
		}
		if got < 10000 || got >= 20000 {
			t.Fatalf("randomID returned %d, want in [10000, 20000)", got)
		}
	}
}

func TestRandomIDVaries(t *testing.T) {
	seen := map[int]bool{}
	for i := 0; i < 20; i++ {
		got, err := randomID(0, 1<<28)
		if err != nil {
			t.Fatalf("randomID: %v", err)
		}
		seen[got] = true
	}
	if len(seen) < 15 {
		t.Errorf("only %d distinct values out of 20 draws -- randomID doesn't look random", len(seen))
	}
}

func TestResolveEndpointLiteralIPv4(t *testing.T) {
	got, err := resolveEndpoint("203.0.113.5:51820")
	if err != nil {
		t.Fatalf("resolveEndpoint: %v", err)
	}
	if got != "203.0.113.5:51820" {
		t.Errorf("got %q, want unchanged literal IP", got)
	}
}

func TestResolveEndpointLiteralIPv6(t *testing.T) {
	got, err := resolveEndpoint("[2001:db8::1]:51820")
	if err != nil {
		t.Fatalf("resolveEndpoint: %v", err)
	}
	if got != "[2001:db8::1]:51820" {
		t.Errorf("got %q, want unchanged literal IP", got)
	}
}

func TestResolveEndpointMissingPort(t *testing.T) {
	if _, err := resolveEndpoint("dash.example.com"); err == nil {
		t.Fatal("expected an error for an endpoint with no port")
	}
}

func TestResolveEndpointHostname(t *testing.T) {
	// "localhost" resolves without a real network call on essentially
	// every system (either /etc/hosts or the stub resolver's builtin),
	// so this exercises the actual DNS-resolution path without making
	// the test suite depend on outbound network access to a real host.
	got, err := resolveEndpoint("localhost:51820")
	if err != nil {
		t.Fatalf("resolveEndpoint: %v", err)
	}
	host, port, err := net.SplitHostPort(got)
	if err != nil {
		t.Fatalf("resolveEndpoint returned an unparseable result %q: %v", got, err)
	}
	if port != "51820" {
		t.Errorf("port = %q, want 51820", port)
	}
	if net.ParseIP(host) == nil {
		t.Errorf("host = %q, want a literal IP", host)
	}
	if strings.Contains(got, "localhost") {
		t.Errorf("resolveEndpoint didn't actually resolve: %q", got)
	}
}
