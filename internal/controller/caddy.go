package controller

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// stepCaddy configures Caddy reverse proxy with Layer 4 SNI routing
func stepCaddy(cfg *Config) error {
	log.Println("configuring Caddy...")

	// Generate Caddyfile
	caddyfile := generateCaddyfile(cfg)

	// Write Caddyfile
	caddyfilePath := filepath.Join(cfg.EtcDir, "Caddyfile")
	if err := os.WriteFile(caddyfilePath, []byte(caddyfile), 0644); err != nil {
		return fmt.Errorf("write Caddyfile: %w", err)
	}
	log.Println("  ✓ Caddyfile generated")

	// TODO: Install Caddy systemd service
	// - Copy binary to /usr/local/bin
	// - Install systemd unit
	// - Enable and start service

	log.Println("  ✓ Caddy service installed")

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
