package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.zx2c4.com/wireguard/tun/netstack"
)

// Result is what `run` prints to stdout as its one line of JSON --
// radar-node's own module collector (writeout_json) reads exactly
// this, the same shape every other bespoke prober here (xray-run.sh,
// openvpn-test.sh) already produces.
type Result struct {
	LatencyMs float64 `json:"latency_ms"`
	HTTPCode  int     `json:"http_code"`
}

// run is the entire tunnel lifecycle in one process: bring the tunnel
// up, probe target THROUGH it, and tear it back down -- all before
// returning. This deliberately does NOT split across separate CLI
// invocations the way an even earlier "up" (bring up, exit) + external
// curl + "down" (tear down) design did (see git history) -- with a
// real kernel TUN device, that design didn't actually work: the
// WireGuard session (handshake, encrypt/decrypt) is Go goroutines
// running inside the process that brought the tunnel up, and doesn't
// outlive it.
//
// bringUp/probe below use wireguard-go's own tun/netstack (an
// in-process virtual network stack) rather than a real kernel
// interface at all now, which sidesteps that whole class of problem --
// see bringUp's own doc comment.
func run(cfg *Config, target string, budget time.Duration) (*Result, error) {
	tnet, teardown, err := bringUp(cfg)
	if err != nil {
		return nil, fmt.Errorf("bring up tunnel: %w", err)
	}
	defer teardown()

	return probe(tnet, target, budget)
}

// probe issues one HTTP GET through the tunnel -- tnet.DialContext
// (routed through the WireGuard session via the in-process netstack;
// see bringUp) is the only thing that makes this "through the tunnel"
// at all, everything else is an ordinary net/http request. Downloads
// and discards the response body, same as curl's own `-o /dev/null`
// did back when this shelled out to curl instead, and reports the
// same two fields curl's `-w '%{time_total} %{http_code}'` used to:
// total elapsed time and the final HTTP status.
//
// The full budget goes to this one request's own context -- unlike
// the old kernel-TUN design, bring-up/teardown here are both
// effectively instant (in-memory netstack setup/is teardown, no
// syscalls to a real kernel interface or routing table), so there's
// no meaningful setup cost to reserve headroom against.
func probe(tnet *netstack.Net, target string, budget time.Duration) (*Result, error) {
	url := target
	if !strings.Contains(url, "://") {
		url = "http://" + url
	}

	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}
				// netstack.CreateNetTUN was given no DNS servers (see
				// bringUp) -- it has no resolver of its own, so a
				// hostname target (not unusual: e.g. the connectivity-
				// check default target this codebase autofills
				// elsewhere is a hostname) has to be resolved before
				// tnet ever sees it. Resolved via the node's own normal
				// DNS, not through the tunnel: the tunnel's job here is
				// reachability to a specific already-known target, not
				// necessarily acting as this probe's DNS service too --
				// most AllowedIPs configs wouldn't even route to
				// whatever DNS server the peer's network expects.
				if net.ParseIP(host) == nil {
					resolved, err := net.DefaultResolver.LookupHost(ctx, host)
					if err != nil {
						return nil, fmt.Errorf("resolve %q: %w", host, err)
					}
					if len(resolved) == 0 {
						return nil, fmt.Errorf("no addresses found for %q", host)
					}
					host = resolved[0]
				}
				return tnet.DialContext(ctx, network, net.JoinHostPort(host, port))
			},
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request for %q: %w", url, err)
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	elapsed := time.Since(start)

	return &Result{LatencyMs: float64(elapsed) / float64(time.Millisecond), HTTPCode: resp.StatusCode}, nil
}
