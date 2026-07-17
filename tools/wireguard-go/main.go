// radar-wg is a small, self-contained wrapper around the upstream
// wireguard-go userspace implementation: it creates and configures a
// WireGuard tunnel entirely through the UAPI + netlink, with no
// dependency on wireguard-tools (`wg`/`wg-quick`) being installed on
// the host at all. Meant to be driven by radar-node's module system as
// a prepare/teardown pair around an existing prober (tcp/http/icmp)
// pointed at a target reachable through the tunnel -- see this repo's
// README for the module YAML shape.
//
// Two subcommands, both short-lived:
//
//	radar-wg up   --config <path> --state <path>
//	radar-wg down --state <path>
//
// `up` does the real work and exits immediately once the interface is
// configured and up -- the tunnel itself is a kernel interface backed
// by this process's own goroutines running as a detached child (see
// tunnel.go's own comment on why `up` never calls dev.Close() on its
// success path). `down` is a separate, later invocation that reads
// back the interface name `up` wrote to --state and deletes it.
//
// Must run with CAP_NET_ADMIN (root, in practice) -- creating a TUN
// device and adding routes both need it. There is no privilege-drop
// here; if that matters for your deployment, wrap this binary with
// your own sudo/capsh policy.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "up":
		fs := flag.NewFlagSet("up", flag.ExitOnError)
		configPath := fs.String("config", "", "path to a JSON tunnel config (see README)")
		statePath := fs.String("state", "", "path to write interface state for a later `down`")
		_ = fs.Parse(os.Args[2:])
		if *configPath == "" || *statePath == "" {
			fmt.Fprintln(os.Stderr, "up requires --config and --state")
			os.Exit(2)
		}
		cfg, err := loadConfig(*configPath)
		if err != nil {
			fail("up", err)
		}
		if err := up(cfg, *statePath); err != nil {
			fail("up", err)
		}

	case "down":
		fs := flag.NewFlagSet("down", flag.ExitOnError)
		statePath := fs.String("state", "", "path written by a previous `up`")
		_ = fs.Parse(os.Args[2:])
		if *statePath == "" {
			fmt.Fprintln(os.Stderr, "down requires --state")
			os.Exit(2)
		}
		if err := down(*statePath); err != nil {
			fail("down", err)
		}

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
	fmt.Fprintln(os.Stderr, `radar-wg -- bring a userspace WireGuard tunnel up/down without wireguard-tools

Usage:
  radar-wg up   --config <path> --state <path>
  radar-wg down --state <path>

See https://github.com/mehrnet/static-builds/tree/main/tools/wireguard-go for the config format.`)
}
