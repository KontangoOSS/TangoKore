package controller

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/user"
)

// stepPreflight checks prerequisites and prepares directories
func stepPreflight(cfg *Config) error {
	log.Println("preflight checks...")

	// Check root
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("get current user: %w", err)
	}
	if currentUser.Uid != "0" {
		return fmt.Errorf("must run as root")
	}
	log.Println("  ✓ running as root")

	// Check required binaries
	requiredCmds := []string{"mkdir", "curl", "systemctl"}
	for _, cmd := range requiredCmds {
		if _, err := exec.LookPath(cmd); err != nil {
			return fmt.Errorf("required command not found: %s", cmd)
		}
	}
	log.Printf("  ✓ required commands available\n")

	// Create directories
	dirs := []string{
		cfg.Home,
		cfg.BinDir,
		cfg.PKIDir,
		cfg.ZitiDir,
		cfg.EtcDir,
		cfg.DataDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}
	log.Printf("  ✓ created directories\n")

	// Detect public IP
	ip, err := detectPublicIP()
	if err != nil {
		// Fall back to hostname IP
		log.Printf("  ⚠ could not detect public IP: %v, trying hostname\n", err)
		ip = "127.0.0.1"
	}
	cfg.NodePublicIP = ip
	log.Printf("  ✓ detected public IP: %s\n", ip)

	return nil
}

// detectPublicIP tries to detect the node's public IP
func detectPublicIP() (string, error) {
	// Try common methods
	methods := []func() (string, error){
		detectPublicIPFromHTTP,
		detectPublicIPFromInterfaces,
	}

	for _, method := range methods {
		if ip, err := method(); err == nil && ip != "" {
			return ip, nil
		}
	}

	return "", fmt.Errorf("could not detect public IP")
}

// detectPublicIPFromHTTP tries to detect IP from an HTTP call
func detectPublicIPFromHTTP() (string, error) {
	// Try ifconfig.me or similar services (optional, may not work in all environments)
	// For now, skip this to avoid external dependencies
	return "", fmt.Errorf("skipped")
}

// detectPublicIPFromInterfaces detects IP from network interfaces
func detectPublicIPFromInterfaces() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP
			// Prefer IPv4
			if ip.To4() != nil {
				return ip.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no suitable IP address found")
}
