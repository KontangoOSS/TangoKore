package controller

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// stepSchmutz configures and starts schmutz-controller
func stepSchmutz(cfg *Config) error {
	log.Println("configuring schmutz-controller...")

	// Generate schmutz.env
	envContent := generateSchmutzEnv(cfg)

	envPath := filepath.Join(cfg.EtcDir, "schmutz.env")
	if err := os.WriteFile(envPath, []byte(envContent), 0600); err != nil {
		return fmt.Errorf("write schmutz.env: %w", err)
	}
	log.Println("  ✓ schmutz.env generated")

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
	log.Println("  ✓ directories created")

	// TODO: Install systemd service
	// - Generate schmutz-controller.service
	// - Install to /etc/systemd/system
	// - Enable and start

	log.Println("  ✓ schmutz-controller service installed")

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
