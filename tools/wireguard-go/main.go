// radar-wg is a small, self-contained wrapper around the upstream
// wireguard-go userspace implementation: it brings up a WireGuard
// tunnel entirely through the UAPI + netlink (no dependency on
// wireguard-tools' `wg`/`wg-quick` being installed on the host at
// all), probes a target through it, and tears it back down -- all in
// one invocation. See this repo's README for the config format.
//
//	radar-wg run --config <path> --target <host:port> --timeout-ms <ms>
//
// Deliberately a single subcommand covering the whole lifecycle, not
// separate "bring up" / "tear down" calls around an external probe:
// wireguard-go here is a *userspace* implementation, so the actual
// WireGuard session (handshake, encrypt/decrypt) is Go goroutines
// running inside this process -- it doesn't outlive the process the
// way a kernel WireGuard interface would. See run.go's own comment for
// the fuller reasoning.
//
// Must run with CAP_NET_ADMIN (root, in practice) -- creating a TUN
// device and adding routes/rules all need it. There is no
// privilege-drop here; if that matters for your deployment, wrap this
// binary with your own sudo/capsh policy.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "run":
		fs := flag.NewFlagSet("run", flag.ExitOnError)
		configPath := fs.String("config", "", "path to a wg-quick(8)-style .conf (see README)")
		target := fs.String("target", "", "host:port to probe reachability of, through the tunnel")
		timeoutMs := fs.Int("timeout-ms", 5000, "overall timeout budget in milliseconds, for bring-up + probe + teardown combined")
		_ = fs.Parse(os.Args[2:])
		if *configPath == "" || *target == "" {
			fmt.Fprintln(os.Stderr, "run requires --config and --target")
			os.Exit(2)
		}
		cfg, err := loadConfig(*configPath)
		if err != nil {
			fail("run", err)
		}
		result, err := run(cfg, *target, time.Duration(*timeoutMs)*time.Millisecond)
		if err != nil {
			fail("run", err)
		}
		out, err := json.Marshal(result)
		if err != nil {
			fail("run", err)
		}
		fmt.Println(string(out))

	case "-h", "--help", "help":
		usage()

	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func fail(subcommand string, err error) {
	fmt.Fprintf(os.Stderr, "radar-wg %s: %v\n", subcommand, err)
	os.Exit(1)
}

func usage() {
	fmt.Fprintln(os.Stderr, `radar-wg -- bring a userspace WireGuard tunnel up, probe through it, and tear it down

Usage:
  radar-wg run --config <path> --target <host:port> --timeout-ms <ms>

See https://github.com/mehrnet/static-builds/tree/main/tools/wireguard-go for the config format.`)
}
