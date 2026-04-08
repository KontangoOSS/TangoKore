package controller

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// stepDownload downloads required binaries
func stepDownload(cfg *Config) error {
	log.Println("downloading binaries...")

	binaries := []struct {
		name       string
		path       string
		url        string
		extractTar bool // if true, extract from tar.gz
		tarPath    string // path within tar
	}{
		{
			name:       "ziti",
			path:       filepath.Join(cfg.BinDir, "ziti"),
			url:        fmt.Sprintf("https://github.com/openziti/ziti/releases/download/v%s/ziti-linux-amd64-%s.tar.gz", cfg.ZitiVersion, cfg.ZitiVersion),
			extractTar: true,
			tarPath:    "ziti",
		},
		{
			name:       "caddy",
			path:       filepath.Join(cfg.BinDir, "caddy"),
			url:        "https://github.com/caddyserver/caddy/releases/download/v2.8.4/caddy_2.8.4_linux_amd64.tar.gz",
			extractTar: true,
			tarPath:    "caddy",
		},
	}

	// Note: bao and schmutz are optional for test mode
	if !cfg.TestMode {
		binaries = append(binaries,
			struct {
				name       string
				path       string
				url        string
				extractTar bool
				tarPath    string
			}{
				name: "bao",
				path: filepath.Join(cfg.BinDir, "bao"),
				url:  fmt.Sprintf("https://releases.hashicorp.com/openbao/%s/openbao_%s_linux_amd64.zip", cfg.BaoVersion, cfg.BaoVersion),
			},
		)
	}

	// Schmutz can come from JoinDomain
	if !cfg.TestMode {
		binaries = append(binaries,
			struct {
				name       string
				path       string
				url        string
				extractTar bool
				tarPath    string
			}{
				name: "schmutz-controller",
				path: filepath.Join(cfg.BinDir, "schmutz-controller"),
				url:  fmt.Sprintf("https://%s/download/schmutz-controller-linux-amd64", cfg.JoinDomain),
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

		if bin.extractTar {
			if err := downloadTarFile(bin.url, bin.path, bin.tarPath); err != nil {
				return fmt.Errorf("download %s: %w", bin.name, err)
			}
		} else {
			if err := downloadFile(bin.url, bin.path); err != nil {
				return fmt.Errorf("download %s: %w", bin.name, err)
			}
		}

		// Make executable
		if err := os.Chmod(bin.path, 0755); err != nil {
			return fmt.Errorf("chmod %s: %w", bin.name, err)
		}

		log.Printf("  ✓ %s downloaded and executable\n", bin.name)
	}

	return nil
}

// downloadFile downloads a file from URL to path
func downloadFile(url, path string) error {
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("http %d", resp.StatusCode)
	}

	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// downloadTarFile downloads a tar.gz file and extracts a specific file
func downloadTarFile(url, destPath, tarPath string) error {
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("http %d", resp.StatusCode)
	}

	// Decompress gzip
	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Extract tar
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return fmt.Errorf("file %s not found in archive", tarPath)
		}
		if err != nil {
			return fmt.Errorf("tar read: %w", err)
		}

		// Match filename
		if strings.Contains(header.Name, tarPath) || header.Name == tarPath {
			out, err := os.Create(destPath)
			if err != nil {
				return fmt.Errorf("create file: %w", err)
			}
			defer out.Close()

			_, err = io.Copy(out, tarReader)
			return err
		}
	}
}
