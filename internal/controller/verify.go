package controller

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// stepVerify validates the installation
func stepVerify(cfg *Config) error {
	log.Println("verifying installation...")

	checks := []struct {
		name string
		fn   func(*Config) error
	}{
		{"PKI files exist", verifyPKI},
		{"Binaries executable", verifyBinaries},
		{"Configuration files", verifyConfig},
	}

	for _, check := range checks {
		if err := check.fn(cfg); err != nil {
			return fmt.Errorf("%s: %w", check.name, err)
		}
		log.Printf("  ✓ %s\n", check.name)
	}

	log.Println("  ✓ all verification checks passed")

	return nil
}

// verifyPKI checks that PKI files exist
func verifyPKI(cfg *Config) error {
	files := []string{
		filepath.Join(cfg.PKIDir, "root-ca.pem"),
		filepath.Join(cfg.PKIDir, "intermediate.pem"),
		filepath.Join(cfg.PKIDir, "server.crt"),
		filepath.Join(cfg.PKIDir, "server.key"),
		filepath.Join(cfg.PKIDir, "ca-bundle.pem"),
	}

	for _, file := range files {
		if _, err := os.Stat(file); err != nil {
			return fmt.Errorf("missing: %s", file)
		}
	}

	return nil
}

// verifyBinaries checks that all required binaries are executable
func verifyBinaries(cfg *Config) error {
	// In test mode, only ziti and caddy are required
	binaries := []string{
		filepath.Join(cfg.BinDir, "ziti"),
		filepath.Join(cfg.BinDir, "caddy"),
	}

	// In production mode, also check bao and schmutz
	if !cfg.TestMode {
		binaries = append(binaries,
			filepath.Join(cfg.BinDir, "bao"),
			filepath.Join(cfg.BinDir, "schmutz-controller"),
		)
	}

	for _, bin := range binaries {
		info, err := os.Stat(bin)
		if err != nil {
			return fmt.Errorf("missing: %s", bin)
		}
		if !isExecutable(info.Mode()) {
			return fmt.Errorf("not executable: %s", bin)
		}
	}

	return nil
}

// verifyConfig checks that configuration files exist
func verifyConfig(cfg *Config) error {
	files := []string{
		filepath.Join(cfg.EtcDir, "schmutz.env"),
		filepath.Join(cfg.EtcDir, "Caddyfile"),
	}

	for _, file := range files {
		if _, err := os.Stat(file); err != nil {
			return fmt.Errorf("missing: %s", file)
		}
	}

	return nil
}

// isExecutable checks if a file mode indicates executability
func isExecutable(mode os.FileMode) bool {
	return (mode & 0111) != 0
}
