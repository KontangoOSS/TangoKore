package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func cmdReset(args []string) {
	fs := flag.NewFlagSet("reset", flag.ExitOnError)
	confirm := fs.Bool("yes", false, "skip confirmation prompt")
	fs.Parse(args)

	if os.Getuid() != 0 {
		fmt.Fprintln(os.Stderr, "reset: must be run as root")
		os.Exit(1)
	}

	if !*confirm {
		fmt.Println("This will:")
		fmt.Println("  - Stop and disable all kontango services")
		fmt.Println("  - Remove systemd units")
		fmt.Println("  - Delete /opt/kontango/ (identity, binaries, configs)")
		fmt.Println("  - Remove the Ziti tunnel interface")
		fmt.Println()
		fmt.Print("Continue? [y/N] ")
		var answer string
		fmt.Scanln(&answer)
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			fmt.Println("aborted")
			return
		}
	}

	// Also clean up legacy tango-* services from previous installs.
	services := []string{
		"kontango-bao-agent",
		"kontango-caddy",
		"kontango-portal",
		"kontango-agent",
		"kontango-tunnel",
		// Legacy names
		"tango-bao-agent",
		"tango-caddy",
		"tango-portal",
		"tango-agent",
		"tango-tunnel",
	}

	fmt.Println("stopping services...")
	for _, svc := range services {
		exec.Command("systemctl", "stop", svc+".service").Run()
		exec.Command("systemctl", "disable", svc+".service").Run()
	}

	fmt.Println("removing systemd units...")
	for _, svc := range services {
		os.Remove("/etc/systemd/system/" + svc + ".service")
	}
	exec.Command("systemctl", "daemon-reload").Run()

	// macOS launchd cleanup
	launchdLabels := []string{
		"io.kontango.kontango-tunnel",
		"io.kontango.kontango-agent",
		"io.kontango.kontango-caddy",
		"io.tango.tango-tunnel",
		"io.tango.tango-agent",
		"io.tango.tango-caddy",
	}
	for _, label := range launchdLabels {
		plist := "/Library/LaunchDaemons/" + label + ".plist"
		if _, err := os.Stat(plist); err == nil {
			exec.Command("launchctl", "unload", plist).Run()
			os.Remove(plist)
		}
	}

	fmt.Println("removing /opt/kontango/...")
	os.RemoveAll("/opt/kontango")

	// Legacy path
	if _, err := os.Stat("/opt/tango"); err == nil {
		fmt.Println("removing /opt/tango/ (legacy)...")
		os.RemoveAll("/opt/tango")
	}

	// macOS path
	os.RemoveAll("/usr/local/kontango")
	os.RemoveAll("/usr/local/tango")

	// Windows path
	pd := os.Getenv("ProgramData")
	if pd != "" {
		os.RemoveAll(pd + "/kontango")
		os.RemoveAll(pd + "/tango")
	}

	fmt.Println("done. machine is clean.")
}
