package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Result is what `run` prints to stdout as its one line of JSON --
// radar-node's own module collector (writeout_json) reads exactly
// this, the same shape every other bespoke prober here (xray-run.sh,
// openvpn-test.sh, the old wireguard-test.sh) already produces.
type Result struct {
	LatencyMs float64 `json:"latency_ms"`
	HTTPCode  int     `json:"http_code"`
}

// run is the entire tunnel lifecycle in one process: bring the tunnel
// up, probe target THROUGH it, and tear it back down -- all before
// returning. This deliberately does NOT split across separate CLI
// invocations the way an earlier "up" (bring up, exit) + external curl
// + "down" (tear down) design did. That earlier design doesn't
// actually work: wireguard-go here is a *userspace* implementation,
// meaning the WireGuard session itself (handshake, encrypt/decrypt) is
// Go goroutines running inside this process, not a kernel module --
// once this process exits, that session is gone regardless of whether
// the kernel-level TUN interface is technically still present (which,
// without TUNSETPERSIST, it also isn't). A curl launched afterward as
// a separate process would at best find a dead, unresponsive
// interface, or at worst silently succeed by reaching a target that
// was *also* reachable over the node's own ordinary internet
// connection -- a false positive with no tunnel involved at all.
//
// budget is the probe's overall timeout (radar-node's own
// {{timeout_ms}}); only a fraction of it goes to curl's own --max-time
// (see probe), leaving headroom for tunnel bring-up above and teardown
// below to actually run to completion before the outer process itself
// gets killed for exceeding that same budget -- same class of race
// this repo already fixed once for xray (see xray-run.sh's own
// {{timeout_ms}}-proportional --max-time, and its comment on why a
// fixed --max-time raced the outer context).
func run(cfg *Config, target string, budget time.Duration) (*Result, error) {
	localAddr, teardown, err := bringUp(cfg)
	if err != nil {
		return nil, fmt.Errorf("bring up tunnel: %w", err)
	}
	defer teardown()

	return probe(target, localAddr, budget)
}

// probe runs curl through the tunnel: --interface binds its outgoing
// connection to the tunnel's own local address, which is exactly what
// the policy rule bringUp() added keys off of -- nothing about this
// curl invocation is otherwise WireGuard-aware, it's just an ordinary
// connectivity check whose traffic happens to get steered into the
// tunnel by that one bound source address.
func probe(target, localAddr string, budget time.Duration) (*Result, error) {
	// 70/30 split, not xray's 40/60 -- bringUp() above is typically
	// sub-100ms (TUN + UAPI + a handful of netlink calls, no external
	// process to wait on the way xray's SOCKS proxy startup needs),
	// so curl can safely get the larger share while still leaving
	// enough headroom for bringUp+teardown to finish inside budget.
	curlBudget := (budget * 7) / 10
	timeoutS := int(curlBudget.Seconds())
	if timeoutS < 1 {
		timeoutS = 1
	}
	cmd := exec.Command("curl",
		"--silent", "--interface", localAddr,
		"--max-time", strconv.Itoa(timeoutS),
		"-o", "/dev/null", "-w", "%{time_total} %{http_code}",
		target,
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("curl: %w", err)
	}
	fields := strings.Fields(string(out))
	if len(fields) != 2 {
		return nil, fmt.Errorf("curl: unexpected output %q", string(out))
	}
	timeTotal, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return nil, fmt.Errorf("curl: parse time_total %q: %w", fields[0], err)
	}
	httpCode, err := strconv.Atoi(fields[1])
	if err != nil {
		return nil, fmt.Errorf("curl: parse http_code %q: %w", fields[1], err)
	}
	// curl's %{time_total} is seconds, not milliseconds -- every native
	// Go check reports genuine ms (see radar-node's internal/probe/
	// latency()), so this converts before labeling the field
	// "latency_ms" rather than silently under-reporting by 1000x.
	return &Result{LatencyMs: timeTotal * 1000, HTTPCode: httpCode}, nil
}
