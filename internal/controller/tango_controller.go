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

	b.WriteString(fmt.Sprintf("LISTEN_ADDR=:%d\n", cfg.TangoControllerPort))
	b.WriteString(fmt.Sprintf("ZITI_ADDR=localhost:%d\n", cfg.ZitiCtrlPort))
	b.WriteString(fmt.Sprintf("ZITI_USER=%s\n", cfg.ZitiAdminUser))
	b.WriteString(fmt.Sprintf("ZITI_PASS=%s\n", cfg.ZitiAdminPass))
	b.WriteString(fmt.Sprintf("BAO_ADDR=https://localhost:%d\n", cfg.BaoPort))
	b.WriteString(fmt.Sprintf("BAO_TOKEN=%s\n", cfg.BaoRootToken))
	b.WriteString(fmt.Sprintf("WEB_DIR=%s\n", filepath.Join(cfg.Home, "controller", "web")))
	b.WriteString(fmt.Sprintf("NODE_NAME=%s\n", cfg.Name))
	b.WriteString(fmt.Sprintf("OVERLAY_DOMAIN=%s\n", cfg.OverlayDomain))

	return b.String()
}
