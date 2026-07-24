package main

import (
	"net"
	"reflect"
	"strings"
	"testing"
)

func TestSplitDefaultRouteIPv4(t *testing.T) {
	got := splitDefaultRoute("0.0.0.0/0")
	want := []string{"0.0.0.0/1", "128.0.0.0/1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSplitDefaultRouteIPv6(t *testing.T) {
	got := splitDefaultRoute("::/0")
	want := []string{"::/1", "8000::/1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSplitDefaultRoutePassthrough(t *testing.T) {
	got := splitDefaultRoute("10.0.0.0/24")
	want := []string{"10.0.0.0/24"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
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
