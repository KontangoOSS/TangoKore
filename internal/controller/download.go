package controller

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/KontangoOSS/TangoKore/internal/util"
)

// stepDownload downloads required binaries
func stepDownload(cfg *Config) error {
	log.Println("downloading binaries...")

	binaries := []struct {
		name    string
		path    string
		url     string
		format  string // "tar.gz", "zip", or "binary"
		binName string // binary name within archive (for tar.gz/zip)
	}{
		{
			name:    "ziti",
			path:    filepath.Join(cfg.BinDir, "ziti"),
			url:     fmt.Sprintf("https://github.com/openziti/ziti/releases/download/v%s/ziti-linux-amd64-%s.tar.gz", cfg.ZitiVersion, cfg.ZitiVersion),
			format:  "tar.gz",
			binName: "ziti",
		},
		{
			name:    "caddy",
			path:    filepath.Join(cfg.BinDir, "caddy"),
			url:     "https://github.com/caddyserver/caddy/releases/download/v2.8.4/caddy_2.8.4_linux_amd64.tar.gz",
			format:  "tar.gz",
			binName: "caddy",
		},
	}

	// Note: bao and schmutz are optional for test mode
	if !cfg.TestMode {
		binaries = append(binaries,
			struct {
				name    string
				path    string
				url     string
				format  string
				binName string
			}{
				name:    "bao",
				path:    filepath.Join(cfg.BinDir, "bao"),
				url:     fmt.Sprintf("https://releases.hashicorp.com/openbao/%s/openbao_%s_linux_amd64.zip", cfg.BaoVersion, cfg.BaoVersion),
				format:  "zip",
				binName: "bao",
			},
		)
	}

	// Schmutz can come from JoinDomain
	if !cfg.TestMode {
		binaries = append(binaries,
			struct {
				name    string
				path    string
				url     string
				format  string
				binName string
			}{
				name:   "schmutz-controller",
				path:   filepath.Join(cfg.BinDir, "schmutz-controller"),
				url:    fmt.Sprintf("https://%s/download/schmutz-controller-linux-amd64", cfg.JoinDomain),
				format: "binary",
			},
		)
	}

	for _, bin := range binaries {
		// Check if already exists
		if _, err := os.Stat(bin.path); err == nil {
			log.Printf("  ✓ %s already downloaded\n", bin.name)
			continue
		}

		log.Printf("  → downloading %s from %s...\n", bin.name, bin.url)

		var err error
		destDir := filepath.Dir(bin.path)
		os.MkdirAll(destDir, 0755)

		switch bin.format {
		case "tar.gz":
			err = util.DownloadAndExtractTarGz(bin.url, destDir, bin.binName)
		case "zip":
			err = util.DownloadAndExtractZip(bin.url, destDir, bin.binName)
		case "binary":
			err = util.DownloadBinary(bin.url, bin.path)
		default:
			err = fmt.Errorf("unknown format: %s", bin.format)
		}

		if err != nil {
			return fmt.Errorf("download %s: %w", bin.name, err)
		}

		log.Printf("  ✓ %s downloaded and executable\n", bin.name)
	}

	return nil
}
