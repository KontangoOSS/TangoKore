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

// stepCaddy configures Caddy reverse proxy (edge-routers only)
func stepCaddy(cfg *Config) error {
	log.Println("step 8/13: configuring Caddy reverse proxy...")

	// Skip Caddy entirely — controllers don't run Caddy
	// Edge router support is future work
	log.Println("  ⚠ skipping Caddy (controller only deployment)")
	return nil

	// 1. Generate Caddyfile
	log.Println("  → generating Caddyfile...")
	caddyfile := generateCaddyfile(cfg)
	caddyfilePath := filepath.Join(cfg.EtcDir, "Caddyfile")
	if err := os.WriteFile(caddyfilePath, []byte(caddyfile), 0644); err != nil {
		return fmt.Errorf("write Caddyfile: %w", err)
	}

	// 2. Generate Caddy environment file (for Cloudflare token)
	log.Println("  → generating Caddy environment...")
	if err := generateCaddyEnv(cfg); err != nil {
		return fmt.Errorf("generate env: %w", err)
	}

	// 3. Install systemd service
	log.Println("  → installing systemd service...")
	if err := installCaddyService(cfg); err != nil {
		return fmt.Errorf("install service: %w", err)
	}

	// 4. Start Caddy
	log.Println("  → starting Caddy...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "systemctl", "restart", "kontango-caddy")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("start caddy: %w", err)
	}

	// 5. Wait for Caddy to be ready
	timeout := 30 * time.Second
	if !cfg.TestMode {
		timeout = 120 * time.Second // LE cert acquisition takes time
	}
	log.Printf("  → waiting for Caddy (up to %.0fs)...\n", timeout.Seconds())
	if !waitForPort("127.0.0.1:443", timeout) {
		return fmt.Errorf("caddy not responding on port 443")
	}

	log.Println("  ✓ Caddy configured and operational")
	return nil
}

// isEdgeRouter checks if this is an edge router node (heuristic for now)
func isEdgeRouter(cfg *Config) bool {
	// In split-node architecture, edge routers would have a NodeRole field
	// For now, assume single controller node = controller, otherwise edge router
	return false // TODO: check cfg.NodeRole when added
}

// generateCaddyEnv creates environment variables for Caddy (Cloudflare token)
func generateCaddyEnv(cfg *Config) error {
	env := ""
	if !cfg.TestMode && cfg.CloudflareToken != "" {
		env = fmt.Sprintf("export CLOUDFLARE_API_TOKEN=%s\n", cfg.CloudflareToken)
	}

	envPath := filepath.Join(cfg.EtcDir, "caddy.env")
	if err := os.WriteFile(envPath, []byte(env), 0600); err != nil {
		return fmt.Errorf("write env: %w", err)
	}

	return nil
}

// installCaddyService installs the Caddy systemd service
func installCaddyService(cfg *Config) error {
	serviceTmpl := `[Unit]
Description=Caddy Web Server
After=network-online.target
Requires=network-online.target

[Service]
Type=simple
User=root
Group=root
ExecStart=/usr/local/bin/caddy run --config {{.ConfDir}}/Caddyfile
StandardOutput=journal
StandardError=journal
SyslogIdentifier=caddy
Restart=on-failure
RestartSec=5s
StartLimitIntervalSec=300
StartLimitBurst=5

[Install]
WantedBy=multi-user.target
`

	tmpl, err := template.New("caddy.service").Parse(serviceTmpl)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	data := map[string]string{
		"ConfDir": cfg.EtcDir,
	}

	outPath := "/etc/systemd/system/kontango-caddy.service"
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
	cmd = exec.CommandContext(ctx, "systemctl", "enable", "kontango-caddy")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("enable service: %w", err)
	}

	return nil
}

// generateCaddyfile creates the Caddyfile content
func generateCaddyfile(cfg *Config) string {
	var b strings.Builder

	tlsBlock := generateTLSBlock(cfg)

	// Root domain block
	b.WriteString(fmt.Sprintf("%s {\n", cfg.Domain))
	b.WriteString(fmt.Sprintf("  %s\n", tlsBlock))
	b.WriteString("  reverse_proxy localhost:1280 {\n")
	b.WriteString("    header_uri /edge/management /edge/management\n")
	b.WriteString("  }\n")
	b.WriteString("}\n\n")

	// Wildcard block
	b.WriteString(fmt.Sprintf("*.%s {\n", cfg.Domain))
	b.WriteString(fmt.Sprintf("  %s\n", tlsBlock))
	b.WriteString("  @api path /api/*\n")
	b.WriteString("  reverse_proxy @api localhost:3080\n")
	b.WriteString("  @schmutz path /schmutz/*\n")
	b.WriteString("  reverse_proxy @schmutz localhost:3080\n")
	b.WriteString("}\n\n")

	// Stage domains
	if !cfg.TestMode {
		stages := []string{"quarantine", "members", "lab", "admin"}
		for _, stage := range stages {
			b.WriteString(fmt.Sprintf("*.%s.%s {\n", stage, cfg.Domain))
			b.WriteString(fmt.Sprintf("  %s\n", tlsBlock))
			b.WriteString("  reverse_proxy localhost:3080\n")
			b.WriteString("}\n\n")
		}
	}

	return b.String()
}

// generateTLSBlock creates the TLS configuration block
func generateTLSBlock(cfg *Config) string {
	if cfg.TestMode {
		return "tls internal"
	}

	// Production: DNS-01 for root domain
	return "tls {\n    dns cloudflare {env.CLOUDFLARE_API_TOKEN}\n  }"
}
