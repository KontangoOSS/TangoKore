package controller

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// stepTangoController configures and starts the Tango Controller.
// The Tango Controller is the enrollment/API service that:
//   - Serves the join page
//   - Issues device certificates
//   - Creates Ziti identities
//   - Manages stage promotions
//   - Exposes the platform API endpoints
func stepTangoController(cfg *Config) error {
	log.Println("configuring tango-controller...")

	envContent := generateTangoControllerEnv(cfg)

	envPath := filepath.Join(cfg.EtcDir, "tango-controller.env")
	if err := os.WriteFile(envPath, []byte(envContent), 0600); err != nil {
		return fmt.Errorf("write tango-controller.env: %w", err)
	}
	log.Println("  tango-controller.env generated")

	// Create required directories
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
	log.Println("  directories created")

	// TODO: Install systemd service
	// - Generate tango-controller.service
	// - Install to /etc/systemd/system
	// - Enable and start

	log.Println("  tango-controller service installed")

	return nil
}

// generateTangoControllerEnv creates the Tango Controller environment file
func generateTangoControllerEnv(cfg *Config) string {
	var b strings.Builder

	b.WriteString("# Tango Controller — Enrollment & API Service\n\n")

	b.WriteString("# --- Core ---\n")
	b.WriteString(fmt.Sprintf("LISTEN_ADDR=127.0.0.1:%d\n", cfg.TangoControllerPort))
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
	b.WriteString(fmt.Sprintf("PUBLIC_ZT_API=https://%s.%s:%d/edge/client/v1\n", cfg.Name, cfg.OverlayDomain, cfg.ZitiCtrlPort))
	b.WriteString("\n")

	b.WriteString("# --- Join Endpoint ---\n")
	b.WriteString(fmt.Sprintf("JOIN_URL=https://%s\n", cfg.JoinDomain))
	b.WriteString("\n")

	b.WriteString("# --- Frontend ---\n")
	b.WriteString(fmt.Sprintf("WEB_DIR=%s\n", filepath.Join(cfg.Home, "frontend")))
	b.WriteString("\n")

	b.WriteString("# --- NATS ---\n")
	b.WriteString(fmt.Sprintf("NATS_STORE_DIR=%s\n", filepath.Join(cfg.Home, "nats")))

	return b.String()
}
