package enroll

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// LinuxPlatform handles registration on Linux (x86_64, arm64, arm).
type LinuxPlatform struct{}

func (p *LinuxPlatform) Preflight() []string {
	var missing []string
	if os.Getuid() != 0 {
		missing = append(missing, "root access (run with sudo)")
	}
	if err := os.MkdirAll("/opt/kontango/.preflight", 0755); err != nil {
		missing = append(missing, "write access to /opt/kontango")
	} else {
		os.Remove("/opt/kontango/.preflight")
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		missing = append(missing, "systemd (systemctl not found)")
	}
	if _, err := net.DialTimeout("tcp", "1.1.1.1:443", 5*time.Second); err != nil {
		missing = append(missing, "network access (cannot reach internet)")
	}
	return missing
}

func (p *LinuxPlatform) KontangoDir() string { return "/opt/kontango" }
func (p *LinuxPlatform) ZitiBinaryPath() string {
	return filepath.Join(p.KontangoDir(), "bin", "ziti")
}
func (p *LinuxPlatform) IdentityPath() string { return filepath.Join(p.KontangoDir(), "identity.json") }
func (p *LinuxPlatform) EnsureDir() error {
	return os.MkdirAll(filepath.Join(p.KontangoDir(), "bin"), 0755)
}

func (p *LinuxPlatform) ZitiDownloadURL(version string) string {
	return fmt.Sprintf("https://github.com/openziti/ziti/releases/download/v%s/ziti-%s-%s-%s.tar.gz",
		version, zitiOSString(), zitiArchString(), version)
}

func (p *LinuxPlatform) InstallZiti(version string) error {
	if _, err := os.Stat(p.ZitiBinaryPath()); err == nil {
		return nil
	}
	url := p.ZitiDownloadURL(version)
	return downloadAndExtractTarGz(url, filepath.Join(p.KontangoDir(), "bin"), "ziti")
}

func (p *LinuxPlatform) InstallAgent(baseURL, identityFile string) error {
	agentBin := filepath.Join(p.KontangoDir(), "bin", "kontango")
	if _, err := os.Stat(agentBin); err != nil {
		name := "kontango-linux-" + zitiArchString()
		if err := downloadBinary(baseURL+"/download/"+name, agentBin); err != nil {
			return fmt.Errorf("download kontango agent: %w", err)
		}
	}
	unit := fmt.Sprintf(`[Unit]
Description=Kontango Machine Agent
After=network-online.target kontango-tunnel.service
Wants=network-online.target
BindsTo=kontango-tunnel.service

[Service]
Type=simple
ExecStart=%s agent -identity %s
Restart=on-failure
RestartSec=10s
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
`, agentBin, identityFile)

	if err := os.WriteFile("/etc/systemd/system/kontango-agent.service", []byte(unit), 0644); err != nil {
		return err
	}
	exec.Command("systemctl", "daemon-reload").Run()
	if err := exec.Command("systemctl", "enable", "kontango-agent.service").Run(); err != nil {
		return err
	}
	return exec.Command("systemctl", "start", "kontango-agent.service").Run()
}

func (p *LinuxPlatform) InstallCaddy(baseURL, identityFile string) error {
	caddyBin := filepath.Join(p.KontangoDir(), "bin", "caddy")
	if _, err := os.Stat(caddyBin); err != nil {
		name := "caddy-linux-" + zitiArchString()
		if err := downloadBinary(baseURL+"/download/"+name, caddyBin); err != nil {
			return fmt.Errorf("download caddy: %w", err)
		}
	}

	caddyfile := filepath.Join(p.KontangoDir(), "Caddyfile")
	if _, err := os.Stat(caddyfile); err != nil {
		// Write a base Caddyfile only if one doesn't exist yet.
		// Apps are added by writing additional site blocks here.
		content := fmt.Sprintf(`# Kontango egress proxy — Ziti overlay transport
# Add site blocks below to expose local services onto the mesh.
# Example:
#   myapp.kontango {
#       reverse_proxy localhost:8080 {
#           transport ziti {
#               identity %s
#           }
#       }
#   }
`, identityFile)
		if err := os.WriteFile(caddyfile, []byte(content), 0644); err != nil {
			return fmt.Errorf("write Caddyfile: %w", err)
		}
	}

	unit := fmt.Sprintf(`[Unit]
Description=Kontango Caddy Egress
After=network-online.target kontango-tunnel.service
Wants=network-online.target

[Service]
Type=notify
ExecStart=%s run --config %s --adapter caddyfile
ExecReload=%s reload --config %s --adapter caddyfile
Restart=on-failure
RestartSec=5s
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
`, caddyBin, caddyfile, caddyBin, caddyfile)

	if err := os.WriteFile("/etc/systemd/system/kontango-caddy.service", []byte(unit), 0644); err != nil {
		return err
	}
	exec.Command("systemctl", "daemon-reload").Run()
	if err := exec.Command("systemctl", "enable", "kontango-caddy.service").Run(); err != nil {
		return err
	}
	return exec.Command("systemctl", "start", "kontango-caddy.service").Run()
}

func (p *LinuxPlatform) InstallService(identityFile string) error {
	unit := fmt.Sprintf(`[Unit]
Description=Kontango Root Tunnel
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s tunnel host -i %s -v
Restart=on-failure
RestartSec=5s
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
`, p.ZitiBinaryPath(), identityFile)

	if err := os.WriteFile("/etc/systemd/system/kontango-tunnel.service", []byte(unit), 0644); err != nil {
		return err
	}
	exec.Command("systemctl", "daemon-reload").Run()
	return exec.Command("systemctl", "enable", "kontango-tunnel.service").Run()
}

func (p *LinuxPlatform) StartService() error {
	return exec.Command("systemctl", "start", "kontango-tunnel.service").Run()
}
func (p *LinuxPlatform) StopService() error {
	return exec.Command("systemctl", "stop", "kontango-tunnel.service").Run()
}
func (p *LinuxPlatform) ServiceStatus() string {
	out, _ := exec.Command("systemctl", "is-active", "kontango-tunnel.service").Output()
	return strings.TrimSpace(string(out))
}

func (p *LinuxPlatform) WaitForTunnel(timeout time.Duration) error {
	zitiBin := p.ZitiBinaryPath()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if p.ServiceStatus() != "active" {
			time.Sleep(2 * time.Second)
			continue
		}
		// Use the v2 ziti agent IPC to check tunnel health
		out, err := exec.Command(zitiBin, "agent", "stats",
			"--app-type", "tunnel", "--timeout", "3s").Output()
		if err == nil && len(out) > 0 {
			var stats map[string]interface{}
			if json.Unmarshal(out, &stats) == nil {
				return nil // agent responded — tunnel is connected
			}
			// Non-JSON response is also fine — it means the agent is alive
			if len(out) > 10 {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("tunnel did not become healthy within %s", timeout)
}

// DarwinPlatform handles registration on macOS (Intel + Apple Silicon).
type DarwinPlatform struct{}

func (p *DarwinPlatform) Preflight() []string {
	var missing []string
	if os.Getuid() != 0 {
		missing = append(missing, "root access (run with sudo)")
	}
	if err := os.MkdirAll("/usr/local/kontango/.preflight", 0755); err != nil {
		missing = append(missing, "write access to /usr/local/kontango")
	} else {
		os.Remove("/usr/local/kontango/.preflight")
	}
	if _, err := exec.LookPath("launchctl"); err != nil {
		missing = append(missing, "launchctl (not found)")
	}
	if _, err := net.DialTimeout("tcp", "1.1.1.1:443", 5*time.Second); err != nil {
		missing = append(missing, "network access (cannot reach internet)")
	}
	return missing
}

func (p *DarwinPlatform) KontangoDir() string       { return "/usr/local/kontango" }
func (p *DarwinPlatform) ZitiBinaryPath() string { return filepath.Join(p.KontangoDir(), "bin", "ziti") }
func (p *DarwinPlatform) IdentityPath() string   { return filepath.Join(p.KontangoDir(), "identity.json") }
func (p *DarwinPlatform) EnsureDir() error {
	return os.MkdirAll(filepath.Join(p.KontangoDir(), "bin"), 0755)
}

func (p *DarwinPlatform) ZitiDownloadURL(version string) string {
	return fmt.Sprintf("https://github.com/openziti/ziti/releases/download/v%s/ziti-%s-%s-%s.tar.gz",
		version, zitiOSString(), zitiArchString(), version)
}

func (p *DarwinPlatform) InstallAgent(baseURL, identityFile string) error {
	agentBin := filepath.Join(p.KontangoDir(), "bin", "kontango")
	if _, err := os.Stat(agentBin); err != nil {
		name := "kontango-darwin-" + zitiArchString()
		if err := downloadBinary(baseURL+"/download/"+name, agentBin); err != nil {
			return fmt.Errorf("download kontango agent: %w", err)
		}
	}
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key><string>io.kontango.kontango-agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string><string>agent</string>
        <string>-identity</string><string>%s</string>
    </array>
    <key>RunAtLoad</key><true/>
    <key>KeepAlive</key><true/>
    <key>StandardOutPath</key><string>%s/kontango-agent.log</string>
    <key>StandardErrorPath</key><string>%s/kontango-agent.log</string>
</dict>
</plist>`, agentBin, identityFile, p.KontangoDir(), p.KontangoDir())
	const plistPath = "/Library/LaunchDaemons/io.kontango.kontango-agent.plist"
	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		return err
	}
	return exec.Command("launchctl", "load", plistPath).Run()
}

func (p *DarwinPlatform) InstallCaddy(baseURL, identityFile string) error {
	caddyBin := filepath.Join(p.KontangoDir(), "bin", "caddy")
	if _, err := os.Stat(caddyBin); err != nil {
		name := "caddy-darwin-" + zitiArchString()
		if err := downloadBinary(baseURL+"/download/"+name, caddyBin); err != nil {
			return fmt.Errorf("download caddy: %w", err)
		}
	}

	caddyfile := filepath.Join(p.KontangoDir(), "Caddyfile")
	if _, err := os.Stat(caddyfile); err != nil {
		content := fmt.Sprintf("# Kontango egress proxy\n# Add site blocks to expose local services onto the mesh.\n# identity: %s\n", identityFile)
		if err := os.WriteFile(caddyfile, []byte(content), 0644); err != nil {
			return fmt.Errorf("write Caddyfile: %w", err)
		}
	}

	const plistPath = "/Library/LaunchDaemons/io.kontango.kontango-caddy.plist"
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key><string>io.kontango.kontango-caddy</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string><string>run</string>
        <string>--config</string><string>%s</string>
        <string>--adapter</string><string>caddyfile</string>
    </array>
    <key>RunAtLoad</key><true/>
    <key>KeepAlive</key><true/>
    <key>StandardOutPath</key><string>%s/kontango-caddy.log</string>
    <key>StandardErrorPath</key><string>%s/kontango-caddy.log</string>
</dict>
</plist>`, caddyBin, caddyfile, p.KontangoDir(), p.KontangoDir())
	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		return err
	}
	return exec.Command("launchctl", "load", plistPath).Run()
}

func (p *DarwinPlatform) InstallZiti(version string) error {
	if _, err := os.Stat(p.ZitiBinaryPath()); err == nil {
		return nil
	}
	url := p.ZitiDownloadURL(version)
	return downloadAndExtractTarGz(url, filepath.Join(p.KontangoDir(), "bin"), "ziti")
}

const darwinPlistPath = "/Library/LaunchDaemons/io.kontango.kontango-tunnel.plist"

func (p *DarwinPlatform) InstallService(identityFile string) error {
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key><string>io.kontango.kontango-tunnel</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string><string>tunnel</string><string>host</string>
        <string>-i</string><string>%s</string><string>-v</string>
    </array>
    <key>RunAtLoad</key><true/>
    <key>KeepAlive</key><true/>
    <key>StandardOutPath</key><string>%s/kontango-tunnel.log</string>
    <key>StandardErrorPath</key><string>%s/kontango-tunnel.log</string>
</dict>
</plist>`, p.ZitiBinaryPath(), identityFile, p.KontangoDir(), p.KontangoDir())
	return os.WriteFile(darwinPlistPath, []byte(plist), 0644)
}

func (p *DarwinPlatform) StartService() error {
	return exec.Command("launchctl", "load", darwinPlistPath).Run()
}
func (p *DarwinPlatform) StopService() error {
	return exec.Command("launchctl", "unload", darwinPlistPath).Run()
}
func (p *DarwinPlatform) ServiceStatus() string {
	if err := exec.Command("launchctl", "list", "io.kontango.kontango-tunnel").Run(); err != nil {
		return "not running"
	}
	return "running"
}

func (p *DarwinPlatform) WaitForTunnel(timeout time.Duration) error {
	zitiBin := p.ZitiBinaryPath()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if p.ServiceStatus() != "running" {
			time.Sleep(2 * time.Second)
			continue
		}
		out, err := exec.Command(zitiBin, "agent", "stats",
			"--app-type", "tunnel", "--timeout", "3s").Output()
		if err == nil && len(out) > 10 {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("tunnel did not become healthy within %s", timeout)
}

// WindowsPlatform handles registration on Windows.
type WindowsPlatform struct{}

func (p *WindowsPlatform) Preflight() []string {
	var missing []string
	kontangoDir := p.KontangoDir()
	if err := os.MkdirAll(filepath.Join(kontangoDir, ".preflight"), 0755); err != nil {
		missing = append(missing, "admin access (cannot write to "+kontangoDir+")")
	} else {
		os.Remove(filepath.Join(kontangoDir, ".preflight"))
	}
	if _, err := exec.LookPath("sc.exe"); err != nil {
		missing = append(missing, "sc.exe (not found)")
	}
	if _, err := net.DialTimeout("tcp", "1.1.1.1:443", 5*time.Second); err != nil {
		missing = append(missing, "network access (cannot reach internet)")
	}
	return missing
}

func (p *WindowsPlatform) KontangoDir() string {
	pd := os.Getenv("ProgramData")
	if pd == "" {
		pd = `C:\ProgramData`
	}
	return filepath.Join(pd, "kontango")
}
func (p *WindowsPlatform) ZitiBinaryPath() string {
	return filepath.Join(p.KontangoDir(), "bin", "ziti.exe")
}
func (p *WindowsPlatform) IdentityPath() string { return filepath.Join(p.KontangoDir(), "identity.json") }
func (p *WindowsPlatform) EnsureDir() error {
	return os.MkdirAll(filepath.Join(p.KontangoDir(), "bin"), 0755)
}

func (p *WindowsPlatform) ZitiDownloadURL(version string) string {
	return fmt.Sprintf("https://github.com/openziti/ziti/releases/download/v%s/ziti-windows-%s-%s.zip",
		version, zitiArchString(), version)
}

func (p *WindowsPlatform) InstallAgent(baseURL, identityFile string) error {
	agentBin := filepath.Join(p.KontangoDir(), "bin", "kontango.exe")
	if _, err := os.Stat(agentBin); err != nil {
		name := "kontango-windows-" + zitiArchString() + ".exe"
		if err := downloadBinary(baseURL+"/download/"+name, agentBin); err != nil {
			return fmt.Errorf("download kontango agent: %w", err)
		}
	}
	binPath := fmt.Sprintf(`"%s" agent -identity "%s"`, agentBin, identityFile)
	return exec.Command("sc.exe", "create", "KontangoAgent",
		fmt.Sprintf("binPath=%s", binPath), "start=auto", "DisplayName=Kontango Machine Agent").Run()
}

func (p *WindowsPlatform) InstallCaddy(baseURL, identityFile string) error {
	caddyBin := filepath.Join(p.KontangoDir(), "bin", "caddy.exe")
	if _, err := os.Stat(caddyBin); err != nil {
		name := "caddy-windows-" + zitiArchString() + ".exe"
		if err := downloadBinary(baseURL+"/download/"+name, caddyBin); err != nil {
			return fmt.Errorf("download caddy: %w", err)
		}
	}

	caddyfile := filepath.Join(p.KontangoDir(), "Caddyfile")
	if _, err := os.Stat(caddyfile); err != nil {
		content := fmt.Sprintf("# Kontango egress proxy\n# Add site blocks to expose local services onto the mesh.\n# identity: %s\n", identityFile)
		if err := os.WriteFile(caddyfile, []byte(content), 0644); err != nil {
			return fmt.Errorf("write Caddyfile: %w", err)
		}
	}

	binPath := fmt.Sprintf(`"%s" run --config "%s" --adapter caddyfile`, caddyBin, caddyfile)
	return exec.Command("sc.exe", "create", "KontangoCaddy",
		fmt.Sprintf("binPath=%s", binPath), "start=auto", "DisplayName=Kontango Caddy Egress").Run()
}

func (p *WindowsPlatform) InstallZiti(version string) error {
	if _, err := os.Stat(p.ZitiBinaryPath()); err == nil {
		return nil
	}
	url := p.ZitiDownloadURL(version)
	return downloadAndExtractZip(url, filepath.Join(p.KontangoDir(), "bin"), "ziti.exe")
}

func (p *WindowsPlatform) InstallService(identityFile string) error {
	binPath := fmt.Sprintf(`"%s" tunnel host -i "%s" -v`, p.ZitiBinaryPath(), identityFile)
	return exec.Command("sc.exe", "create", "KontangoTunnel",
		fmt.Sprintf("binPath=%s", binPath), "start=auto", "DisplayName=Kontango Root Tunnel").Run()
}

func (p *WindowsPlatform) StartService() error {
	return exec.Command("sc.exe", "start", "KontangoTunnel").Run()
}
func (p *WindowsPlatform) StopService() error {
	return exec.Command("sc.exe", "stop", "KontangoTunnel").Run()
}
func (p *WindowsPlatform) ServiceStatus() string {
	out, err := exec.Command("sc.exe", "query", "KontangoTunnel").Output()
	if err != nil {
		return "not installed"
	}
	if bytes.Contains(out, []byte("RUNNING")) {
		return "running"
	}
	return "stopped"
}

func (p *WindowsPlatform) WaitForTunnel(timeout time.Duration) error {
	zitiBin := p.ZitiBinaryPath()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if p.ServiceStatus() != "running" {
			time.Sleep(2 * time.Second)
			continue
		}
		out, err := exec.Command(zitiBin, "agent", "stats",
			"--app-type", "tunnel", "--timeout", "3s").Output()
		if err == nil && len(out) > 10 {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("tunnel did not become healthy within %s", timeout)
}

// ensure runtime import is used
var _ = runtime.GOOS
