package controller

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// stepPreflight runs comprehensive system checks before bootstrap.
// Validates: user, OS, architecture, kernel, memory, disk, DNS,
// network connectivity, required commands, ports, and time sync.
func stepPreflight(cfg *Config) error {
	log.Println("preflight checks...")

	// ── User & Permissions ──
	if err := checkRoot(); err != nil {
		return err
	}

	// ── OS & Architecture ──
	if err := checkPlatform(); err != nil {
		return err
	}

	// ── System Resources ──
	if err := checkResources(cfg); err != nil {
		return err
	}

	// ── Required Commands ──
	if err := checkCommands(); err != nil {
		return err
	}

	// ── Network Connectivity ──
	if err := checkNetwork(); err != nil {
		return err
	}

	// ── DNS Resolution ──
	if err := checkDNS(cfg); err != nil {
		return err
	}

	// ── Port Availability ──
	if err := checkPorts(cfg); err != nil {
		return err
	}

	// ── Time Synchronization ──
	checkTimesync()

	// ── Create Directories ──
	if err := createDirectories(cfg); err != nil {
		return err
	}

	// ── Detect Public IP ──
	ip, err := detectPublicIP()
	if err != nil {
		log.Printf("  ⚠ could not detect public IP: %v, trying hostname\n", err)
		ip = "127.0.0.1"
	}
	cfg.NodePublicIP = ip
	log.Printf("  ✓ public IP: %s\n", ip)

	// ── Firewall ──
	if err := configureFirewall(cfg); err != nil {
		return fmt.Errorf("firewall: %w", err)
	}

	log.Println("  ✓ all preflight checks passed")
	return nil
}

// checkRoot verifies the process is running as root.
func checkRoot() error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("get current user: %w", err)
	}
	if currentUser.Uid != "0" {
		return fmt.Errorf("must run as root (current uid=%s)", currentUser.Uid)
	}
	log.Println("  ✓ running as root")
	return nil
}

// checkPlatform validates the OS, architecture, and kernel version.
func checkPlatform() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("unsupported OS: %s (linux required)", runtime.GOOS)
	}

	switch runtime.GOARCH {
	case "amd64", "arm64":
		// supported
	default:
		return fmt.Errorf("unsupported architecture: %s (amd64 or arm64 required)", runtime.GOARCH)
	}

	log.Printf("  ✓ platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	// Check kernel version (need >= 4.15 for modern networking)
	if out, err := exec.Command("uname", "-r").Output(); err == nil {
		kernel := strings.TrimSpace(string(out))
		log.Printf("  ✓ kernel: %s\n", kernel)
		parts := strings.SplitN(kernel, ".", 3)
		if len(parts) >= 2 {
			major, _ := strconv.Atoi(parts[0])
			minor, _ := strconv.Atoi(parts[1])
			if major < 4 || (major == 4 && minor < 15) {
				return fmt.Errorf("kernel %s too old (>= 4.15 required for eBPF and modern networking)", kernel)
			}
		}
	}

	// Detect distro
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		lines := strings.Split(string(data), "\n")
		var prettyName string
		var versionID string
		for _, line := range lines {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				prettyName = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
			}
			if strings.HasPrefix(line, "VERSION_ID=") {
				versionID = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
			}
		}
		if prettyName != "" {
			log.Printf("  ✓ distro: %s\n", prettyName)
		}

		// Warn on unsupported distros
		content := string(data)
		supported := strings.Contains(content, "ubuntu") || strings.Contains(content, "debian")
		if !supported {
			log.Printf("  ⚠ untested distro — Ubuntu 22.04+ or Debian 12+ recommended\n")
		}

		// Warn on old versions
		if versionID != "" {
			ver, _ := strconv.ParseFloat(versionID, 64)
			if strings.Contains(content, "ubuntu") && ver < 22.04 {
				log.Printf("  ⚠ Ubuntu %s is old — 22.04+ recommended\n", versionID)
			}
			if strings.Contains(content, "debian") && ver < 12 {
				log.Printf("  ⚠ Debian %s is old — 12+ recommended\n", versionID)
			}
		}
	}

	// Check systemd
	if _, err := exec.LookPath("systemctl"); err != nil {
		return fmt.Errorf("systemd not found — required for service management")
	}

	return nil
}

// checkResources verifies minimum memory and disk space.
func checkResources(cfg *Config) error {
	// Memory check (minimum 512MB, recommended 1GB)
	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err == nil {
		totalMB := info.Totalram * uint64(info.Unit) / 1024 / 1024
		freeMB := (info.Freeram + info.Bufferram) * uint64(info.Unit) / 1024 / 1024
		log.Printf("  ✓ memory: %d MB total, %d MB available\n", totalMB, freeMB)

		if totalMB < 512 {
			return fmt.Errorf("insufficient memory: %d MB (minimum 512 MB)", totalMB)
		}
		if totalMB < 1024 {
			log.Printf("  ⚠ low memory: %d MB (1024 MB+ recommended for production)\n", totalMB)
		}
	}

	// Disk check on target directory (minimum 2GB free)
	targetDir := cfg.Home
	if targetDir == "" {
		targetDir = "/opt/kontango"
	}
	// Check the parent directory if target doesn't exist yet
	checkDir := targetDir
	for checkDir != "/" {
		if _, err := os.Stat(checkDir); err == nil {
			break
		}
		checkDir = filepath.Dir(checkDir)
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs(checkDir, &stat); err == nil {
		freeGB := stat.Bavail * uint64(stat.Bsize) / 1024 / 1024 / 1024
		totalGB := stat.Blocks * uint64(stat.Bsize) / 1024 / 1024 / 1024
		log.Printf("  ✓ disk: %d GB free / %d GB total on %s\n", freeGB, totalGB, checkDir)

		if freeGB < 2 {
			return fmt.Errorf("insufficient disk space: %d GB free (minimum 2 GB)", freeGB)
		}
		if freeGB < 5 {
			log.Printf("  ⚠ low disk: %d GB free (5 GB+ recommended)\n", freeGB)
		}
	}

	// CPU count
	cpus := runtime.NumCPU()
	log.Printf("  ✓ CPUs: %d\n", cpus)
	if cpus < 2 {
		log.Println("  ⚠ single CPU detected — 2+ recommended for production")
	}

	return nil
}

// checkCommands verifies all required system commands are available.
func checkCommands() error {
	required := []struct {
		cmd  string
		desc string
	}{
		{"curl", "HTTP client for downloads"},
		{"openssl", "TLS certificate generation"},
		{"tar", "archive extraction"},
		{"systemctl", "service management"},
		{"ip", "network configuration"},
	}

	optional := []struct {
		cmd  string
		desc string
	}{
		{"ufw", "firewall management"},
		{"jq", "JSON processing"},
		{"unzip", "archive extraction"},
	}

	for _, c := range required {
		if _, err := exec.LookPath(c.cmd); err != nil {
			return fmt.Errorf("required command not found: %s (%s)", c.cmd, c.desc)
		}
	}
	log.Println("  ✓ required commands: curl, openssl, tar, systemctl, ip")

	var missing []string
	for _, c := range optional {
		if _, err := exec.LookPath(c.cmd); err != nil {
			missing = append(missing, c.cmd)
		}
	}
	if len(missing) > 0 {
		log.Printf("  ⚠ optional commands missing: %s\n", strings.Join(missing, ", "))
	}

	return nil
}

// checkNetwork verifies outbound internet connectivity.
func checkNetwork() error {
	targets := []struct {
		host string
		desc string
	}{
		{"1.1.1.1:443", "internet (Cloudflare DNS)"},
		{"github.com:443", "GitHub (binary downloads)"},
	}

	for _, t := range targets {
		conn, err := net.DialTimeout("tcp", t.host, 5*time.Second)
		if err != nil {
			return fmt.Errorf("network unreachable: %s (%s) — %v", t.desc, t.host, err)
		}
		conn.Close()
	}
	log.Println("  ✓ outbound connectivity: internet and GitHub reachable")

	return nil
}

// checkDNS verifies DNS resolution for the configured domains.
func checkDNS(cfg *Config) error {
	if cfg.TestMode {
		log.Println("  ✓ DNS: skipped (test mode)")
		return nil
	}

	if cfg.Domain == "" {
		log.Println("  ✓ DNS: skipped (no domain configured)")
		return nil
	}

	// Check the main domain resolves
	domains := []string{cfg.Domain}
	if cfg.JoinDomain != "" {
		domains = append(domains, cfg.JoinDomain)
	}
	// Also check the node-specific FQDN
	nodeFQDN := cfg.Name + "." + cfg.Domain
	domains = append(domains, nodeFQDN)

	for _, domain := range domains {
		addrs, err := net.LookupHost(domain)
		if err != nil {
			log.Printf("  ⚠ DNS: %s does not resolve — %v\n", domain, err)
			log.Printf("    → ensure DNS A/CNAME records are configured before production use\n")
			continue
		}
		log.Printf("  ✓ DNS: %s → %s\n", domain, strings.Join(addrs, ", "))
	}

	// Verify DNS resolvers work
	if _, err := net.LookupHost("github.com"); err != nil {
		return fmt.Errorf("DNS resolution broken: cannot resolve github.com — %v", err)
	}

	return nil
}

// checkPorts verifies required ports are not already in use.
func checkPorts(cfg *Config) error {
	ports := []struct {
		port int
		desc string
	}{
		{80, "HTTP (Caddy)"},
		{443, "HTTPS (Caddy)"},
		{cfg.ZitiCtrlPort, "Ziti controller"},
		{cfg.ZitiEdgePort, "Ziti edge"},
		{cfg.BaoPort, "Bao API"},
		{cfg.SchmutzPort, "Schmutz enrollment"},
	}

	var conflicts []string
	for _, p := range ports {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", p.port))
		if err != nil {
			conflicts = append(conflicts, fmt.Sprintf("%d (%s)", p.port, p.desc))
			continue
		}
		ln.Close()
	}

	if len(conflicts) > 0 {
		log.Printf("  ⚠ ports in use: %s\n", strings.Join(conflicts, ", "))
		log.Println("    → these services may conflict with the bootstrap")
	} else {
		log.Println("  ✓ required ports available: 80, 443, " +
			fmt.Sprintf("%d, %d, %d, %d", cfg.ZitiCtrlPort, cfg.ZitiEdgePort, cfg.BaoPort, cfg.SchmutzPort))
	}

	return nil
}

// checkTimesync warns if the system clock appears to be off.
func checkTimesync() {
	// Check if NTP/chrony/systemd-timesyncd is active
	timesyncServices := []string{"systemd-timesyncd", "chrony", "ntp", "ntpd"}
	synced := false
	for _, svc := range timesyncServices {
		out, err := exec.Command("systemctl", "is-active", svc).Output()
		if err == nil && strings.TrimSpace(string(out)) == "active" {
			log.Printf("  ✓ time sync: %s active\n", svc)
			synced = true
			break
		}
	}
	if !synced {
		log.Println("  ⚠ no time sync service detected — TLS certificates require accurate clocks")
	}
}

// createDirectories creates the installation directory tree.
func createDirectories(cfg *Config) error {
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
	log.Printf("  ✓ directories created under %s\n", cfg.Home)
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
		// Extract IP or hostname from JoinLeader (format: "host:port" or "root@host:port")
		leader := cfg.JoinLeader
		if idx := strings.Index(leader, "@"); idx >= 0 {
			leader = leader[idx+1:]
		}
		// Strip port if present (format: "host:port")
		if idx := strings.Index(leader, ":"); idx >= 0 {
			leader = leader[:idx]
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

// generateTestPKI creates self-signed root CA and server cert for test mode
// This allows Bao to start with TLS before the full PKI is generated in step 5
func generateTestPKI(cfg *Config) error {
	// Create PKI directory
	if err := os.MkdirAll(cfg.PKIDir, 0755); err != nil {
		return fmt.Errorf("mkdir pki: %w", err)
	}

	// Generate self-signed root CA
	rootKey := filepath.Join(cfg.PKIDir, "root-ca.key")
	rootCert := filepath.Join(cfg.PKIDir, "root-ca.crt")

	// openssl genrsa -out root-ca.key 2048
	cmd := exec.Command("openssl", "genrsa", "-out", rootKey, "2048")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("genrsa root: %s", string(out))
	}

	// openssl req -x509 -new -nodes -key root-ca.key -days 3650 -out root-ca.crt -subj "/CN=test-ca"
	cmd = exec.Command("openssl", "req", "-x509", "-new", "-nodes",
		"-key", rootKey, "-days", "3650", "-out", rootCert,
		"-subj", "/CN=test-ca")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("req root: %s", string(out))
	}

	// Generate server key
	serverKey := filepath.Join(cfg.PKIDir, "server.key")
	cmd = exec.Command("openssl", "genrsa", "-out", serverKey, "2048")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("genrsa server: %s", string(out))
	}

	// Create CSR config with SAN for localhost
	csrConf := filepath.Join(cfg.PKIDir, "server.conf")
	csrConfContent := fmt.Sprintf(`[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
CN = localhost

[v3_req]
subjectAltName = DNS:localhost,DNS:127.0.0.1,DNS:%s
`, cfg.Name+"."+cfg.Domain)

	if err := os.WriteFile(csrConf, []byte(csrConfContent), 0600); err != nil {
		return fmt.Errorf("write csr conf: %w", err)
	}

	// Generate server CSR
	serverCsr := filepath.Join(cfg.PKIDir, "server.csr")
	cmd = exec.Command("openssl", "req", "-new", "-key", serverKey,
		"-out", serverCsr, "-config", csrConf)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("req server: %s", string(out))
	}

	// Sign server cert with root CA
	serverCert := filepath.Join(cfg.PKIDir, "server.crt")
	cmd = exec.Command("openssl", "x509", "-req", "-in", serverCsr,
		"-CA", rootCert, "-CAkey", rootKey, "-CAcreateserial",
		"-out", serverCert, "-days", "365",
		"-extensions", "v3_req", "-extfile", csrConf)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("x509 sign: %s", string(out))
	}

	// Create ca-bundle.pem (root cert for clients)
	bundlePath := filepath.Join(cfg.PKIDir, "ca-bundle.pem")
	bundleData, err := os.ReadFile(rootCert)
	if err != nil {
		return fmt.Errorf("read root cert: %w", err)
	}
	if err := os.WriteFile(bundlePath, bundleData, 0644); err != nil {
		return fmt.Errorf("write bundle: %w", err)
	}

	// Create signing-chain.crt (for Ziti, same as root in test mode)
	chainPath := filepath.Join(cfg.PKIDir, "signing-chain.crt")
	if err := os.WriteFile(chainPath, bundleData, 0644); err != nil {
		return fmt.Errorf("write chain: %w", err)
	}

	// Create intermediate.crt and intermediate.key (same as root for test mode)
	intCertPath := filepath.Join(cfg.PKIDir, "intermediate.crt")
	intKeyPath := filepath.Join(cfg.PKIDir, "intermediate.key")

	if err := os.WriteFile(intCertPath, bundleData, 0644); err != nil {
		return fmt.Errorf("write int cert: %w", err)
	}

	rootKeyData, err := os.ReadFile(rootKey)
	if err != nil {
		return fmt.Errorf("read root key: %w", err)
	}
	if err := os.WriteFile(intKeyPath, rootKeyData, 0600); err != nil {
		return fmt.Errorf("write int key: %w", err)
	}

	return nil
}
