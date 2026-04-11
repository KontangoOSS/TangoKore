package controller

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/user"
	"strings"
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

	// Configure firewall
	if err := configureFirewall(cfg); err != nil {
		return fmt.Errorf("firewall: %w", err)
	}

	return nil
}

// configureFirewall sets up UFW rules for the controller node.
// Allows: SSH(22), HTTP(80), HTTPS(443), Ziti router (edge+link) from anywhere.
// Restricts: Ziti controller, Bao API, Bao raft to cluster IPs + localhost only.
func configureFirewall(cfg *Config) error {
	// Ensure UFW is installed
	if _, err := exec.LookPath("ufw"); err != nil {
		return fmt.Errorf("ufw not found: %w", err)
	}

	// Get current rules to avoid duplicates
	out, err := exec.Command("ufw", "status").CombinedOutput()
	if err != nil {
		return fmt.Errorf("ufw status: %w", err)
	}
	currentRules := string(out)

	// Helper to run ufw commands only if rule doesn't already exist
	ufwAllow := func(rule string, args ...string) error {
		// Check if this rule already exists in current output
		if strings.Contains(currentRules, rule) {
			return nil
		}
		cmdArgs := append([]string{"allow"}, args...)
		cmd := exec.Command("ufw", cmdArgs...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("ufw allow %s: %s", rule, string(out))
		}
		return nil
	}

	ufwDelete := func(args ...string) {
		cmdArgs := append([]string{"--force", "delete", "allow"}, args...)
		exec.Command("ufw", cmdArgs...).Run() // ignore errors, rule may not exist
	}

	// Enable UFW if not active
	if !strings.Contains(currentRules, "Status: active") {
		cmd := exec.Command("ufw", "--force", "enable")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("ufw enable: %s", string(out))
		}
		log.Println("  ✓ firewall enabled")
	}

	// Set default deny incoming
	exec.Command("ufw", "default", "deny", "incoming").Run()
	exec.Command("ufw", "default", "allow", "outgoing").Run()

	// --- Public ports (allow from anywhere) ---
	publicPorts := []struct {
		port string
		name string
	}{
		{"22/tcp", "ssh"},
		{"80/tcp", "http"},
		{"443/tcp", "https"},
		{fmt.Sprintf("%d/tcp", cfg.ZitiEdgePort), "ziti-edge"},
		{fmt.Sprintf("%d/tcp", cfg.ZitiLinkPort), "ziti-link"},
	}

	for _, p := range publicPorts {
		if err := ufwAllow(p.port, p.port, "comment", p.name); err != nil {
			return err
		}
	}

	// --- Cluster-only ports (restrict to cluster IPs + localhost) ---
	clusterPorts := []struct {
		ports string
		name  string
	}{
		{fmt.Sprintf("%d/tcp", cfg.ZitiCtrlPort), "ziti-ctrl-cluster"},
		{fmt.Sprintf("%d,%d/tcp", cfg.BaoPort, cfg.BaoRaftPort), "bao-cluster"},
	}

	// Remove any wide-open rules for cluster ports
	ufwDelete(fmt.Sprintf("%d/tcp", cfg.ZitiCtrlPort))
	ufwDelete(fmt.Sprintf("%d/tcp", cfg.BaoPort))
	ufwDelete(fmt.Sprintf("%d/tcp", cfg.BaoRaftPort))

	// Determine cluster peer IPs — for now use the node's own IP.
	// In join mode the leader IP is also a peer.
	// Additional peers are added post-install when the cluster forms.
	clusterIPs := []string{"127.0.0.1", cfg.NodePublicIP}
	if cfg.JoinMode && cfg.JoinLeader != "" {
		// Extract IP or hostname from JoinLeader (format: "root@host")
		leader := cfg.JoinLeader
		if idx := strings.Index(leader, "@"); idx >= 0 {
			leader = leader[idx+1:]
		}
		// Resolve to IP if it's a hostname
		if addrs, err := net.LookupHost(leader); err == nil && len(addrs) > 0 {
			clusterIPs = append(clusterIPs, addrs[0])
		} else {
			clusterIPs = append(clusterIPs, leader)
		}
	}

	for _, cp := range clusterPorts {
		for _, ip := range clusterIPs {
			if err := ufwAllow(
				fmt.Sprintf("%s from %s", cp.ports, ip),
				"from", ip, "to", "any", "port", strings.TrimSuffix(cp.ports, "/tcp"), "proto", "tcp", "comment", cp.name,
			); err != nil {
				return err
			}
		}
	}

	log.Println("  ✓ firewall configured")
	log.Printf("    public: 22, 80, 443, %d, %d\n", cfg.ZitiEdgePort, cfg.ZitiLinkPort)
	log.Printf("    cluster-only: %d, %d, %d\n", cfg.ZitiCtrlPort, cfg.BaoPort, cfg.BaoRaftPort)
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
