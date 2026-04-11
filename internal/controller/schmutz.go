package controller

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// stepSchmutz configures and starts schmutz (controller or gateway)
func stepSchmutz(cfg *Config) error {
	log.Println("step 9/13: configuring schmutz enrollment service...")

	// For now, assume controller mode (TODO: add NodeRole to Config for split-node)
	isController := !cfg.JoinMode

	// 1. Generate schmutz environment
	log.Println("  → generating schmutz environment...")
	envContent := generateSchmutzEnv(cfg)
	envPath := filepath.Join(cfg.EtcDir, "schmutz.env")
	if err := os.WriteFile(envPath, []byte(envContent), 0600); err != nil {
		return fmt.Errorf("write schmutz.env: %w", err)
	}

	// 2. Create required directories
	log.Println("  → creating directories...")
	dirs := []string{
		filepath.Join(cfg.Home, "nats"),
		filepath.Join(cfg.Home, "join", "bin"),
		filepath.Join(cfg.Home, "frontend"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	// 3. Install systemd service
	log.Println("  → installing systemd service...")
	var serviceName string
	if isController {
		serviceName = "kontango-schmutz-controller"
		if err := installSchmutzService(cfg, serviceName, "/usr/local/bin/schmutz-controller"); err != nil {
			return fmt.Errorf("install controller: %w", err)
		}
	} else {
		serviceName = "kontango-schmutz-gateway"
		if err := installSchmutzService(cfg, serviceName, "/usr/local/bin/schmutz-gateway"); err != nil {
			return fmt.Errorf("install gateway: %w", err)
		}
	}

	// 4. Start schmutz service
	log.Println("  → starting schmutz...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "systemctl", "restart", serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("start %s: %w", serviceName, err)
	}

	// 5. Wait for schmutz to be ready
	log.Printf("  → waiting for schmutz (up to 30s)...\n")
	healthPort := fmt.Sprintf("127.0.0.1:%d", cfg.SchmutzPort)
	if !waitForPort(healthPort, 30*time.Second) {
		return fmt.Errorf("schmutz not responding on %s", healthPort)
	}

	roleStr := "controller"
	if !isController {
		roleStr = "gateway"
	}
	log.Printf("  ✓ schmutz-%s initialized and operational\n", roleStr)

	return nil
}

// installSchmutzService installs the systemd service for schmutz
func installSchmutzService(cfg *Config, serviceName, execPath string) error {
	serviceTmpl := `[Unit]
Description=Schmutz Enrollment Service
After=network-online.target
Requires=network-online.target

[Service]
Type=simple
User=root
Group=root
EnvironmentFile={{.ConfDir}}/schmutz.env
ExecStart={{.ExecPath}}
StandardOutput=journal
StandardError=journal
SyslogIdentifier=schmutz
Restart=on-failure
RestartSec=5s
StartLimitIntervalSec=300
StartLimitBurst=5

[Install]
WantedBy=multi-user.target
`

	tmpl, err := template.New(serviceName).Parse(serviceTmpl)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	data := map[string]string{
		"ConfDir":  cfg.EtcDir,
		"ExecPath": execPath,
	}

	outPath := filepath.Join("/etc/systemd/system", serviceName+".service")
	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("create service: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	// Reload systemd
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}

	// Enable service
	cmd = exec.CommandContext(ctx, "systemctl", "enable", serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("enable service: %w", err)
	}

	return nil
}

// generateSchmutzEnv creates the schmutz environment file
func generateSchmutzEnv(cfg *Config) string {
	var b strings.Builder

	b.WriteString("# --- Core ---\n")
	b.WriteString(fmt.Sprintf("LISTEN_ADDR=0.0.0.0:%d\n", cfg.SchmutzPort))
	b.WriteString(fmt.Sprintf("NODE_NAME=%s\n", cfg.Name))
	b.WriteString("\n")

	b.WriteString("# --- Ziti Controller ---\n")
	b.WriteString(fmt.Sprintf("ZITI_CTRL_ADDR=localhost:%d\n", cfg.ZitiCtrlPort))
	b.WriteString(fmt.Sprintf("ZITI_ADMIN_USER=%s\n", cfg.ZitiAdminUser))
	b.WriteString(fmt.Sprintf("ZITI_ADMIN_PASS=%s\n", cfg.ZitiAdminPass))
	b.WriteString(fmt.Sprintf("ZITI_BIN=%s\n", filepath.Join(cfg.BinDir, "ziti")))
	b.WriteString(fmt.Sprintf("ZITI_VERSION=%s\n", cfg.ZitiVersion))
	b.WriteString("\n")

	b.WriteString("# --- OpenBao ---\n")
	b.WriteString(fmt.Sprintf("BAO_ADDR=https://127.0.0.1:%d\n", cfg.BaoPort))
	b.WriteString(fmt.Sprintf("BAO_TOKEN=%s\n", cfg.BaoRootToken))
	b.WriteString("\n")

	b.WriteString("# --- PKI ---\n")
	b.WriteString(fmt.Sprintf("CA_BUNDLE_PATH=%s\n", filepath.Join(cfg.PKIDir, "ca-bundle.pem")))
	b.WriteString("\n")

	b.WriteString("# --- Public API ---\n")
	b.WriteString(fmt.Sprintf("PUBLIC_ZT_API=https://%s.%s:443/edge/client/v1\n", cfg.Name, cfg.Domain))
	b.WriteString("\n")

	b.WriteString("# --- Join Endpoint ---\n")
	b.WriteString(fmt.Sprintf("JOIN_URL=https://%s\n", cfg.JoinDomain))
	b.WriteString("\n")

	b.WriteString("# --- Frontend ---\n")
	b.WriteString(fmt.Sprintf("WEB_DIR=%s\n", filepath.Join(cfg.Home, "frontend")))
	b.WriteString("\n")

	b.WriteString("# --- Downloads ---\n")
	b.WriteString(fmt.Sprintf("GITHUB_RELEASE=https://%s/download\n", cfg.JoinDomain))
	b.WriteString("GITHUB_RAW=https://raw.githubusercontent.com/KontangoOSS/schmutz/main\n")
	b.WriteString("\n")

	b.WriteString("# --- NATS ---\n")
	b.WriteString(fmt.Sprintf("NATS_STORE_DIR=%s\n", filepath.Join(cfg.Home, "nats")))

	return b.String()
}
