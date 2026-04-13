package agent

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ConfigureTangoDNS sets up the system DNS resolver to forward all queries
// for the overlay domain (.tango) to the Ziti tunnel DNS at 100.64.0.1.
//
// Rule: if it's .tango, it resolves via Ziti. No exceptions.
//
// On Linux with systemd-resolved: creates a drop-in config
// On Linux without systemd: prepends to /etc/resolv.conf
func ConfigureTangoDNS(overlayDomain string) error {
	if overlayDomain == "" {
		overlayDomain = "tango"
	}

	if runtime.GOOS != "linux" {
		return fmt.Errorf("DNS configuration not implemented for %s", runtime.GOOS)
	}

	if isSystemdResolved() {
		return configureDNSSystemdResolved(overlayDomain)
	}

	return configureDNSResolvConf(overlayDomain)
}

func configureDNSSystemdResolved(overlayDomain string) error {
	dir := "/etc/systemd/resolved.conf.d"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	config := fmt.Sprintf("[Resolve]\nDNS=100.64.0.1\nDomains=~%s\n", overlayDomain)

	path := fmt.Sprintf("%s/ziti-%s.conf", dir, overlayDomain)
	if err := os.WriteFile(path, []byte(config), 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	slog.Info("configured systemd-resolved for overlay DNS", "domain", overlayDomain, "dns", "100.64.0.1")

	if err := exec.Command("systemctl", "restart", "systemd-resolved").Run(); err != nil {
		slog.Warn("could not restart systemd-resolved", "error", err)
	}

	return nil
}

func configureDNSResolvConf(overlayDomain string) error {
	content, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return fmt.Errorf("read resolv.conf: %w", err)
	}

	if strings.Contains(string(content), "100.64.0.1") {
		slog.Info("resolv.conf already has Ziti DNS")
		return nil
	}

	newContent := fmt.Sprintf("nameserver 100.64.0.1\nsearch %s\n%s", overlayDomain, string(content))
	if err := os.WriteFile("/etc/resolv.conf", []byte(newContent), 0644); err != nil {
		return fmt.Errorf("write resolv.conf: %w", err)
	}

	slog.Info("configured resolv.conf for overlay DNS", "domain", overlayDomain, "dns", "100.64.0.1")
	return nil
}

func isSystemdResolved() bool {
	out, err := exec.Command("systemctl", "is-active", "systemd-resolved").Output()
	return err == nil && strings.TrimSpace(string(out)) == "active"
}

// ConfigureTproxyRoutes sets up ip rule and route table for tproxy mode.
func ConfigureTproxyRoutes() error {
	exec.Command("ip", "rule", "add", "fwmark", "1", "lookup", "100").Run()
	exec.Command("ip", "route", "add", "local", "0.0.0.0/0", "dev", "lo", "table", "100").Run()
	slog.Info("tproxy routing rules configured")
	return nil
}
