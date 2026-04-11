package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// stepPKI generates PKI using Ziti native PKI tools (compatible with SPIFFE URIs)
// For production: Ziti PKI for SPIFFE/mesh, LE certs for Bao TLS listener
// This matches the working kore/kontango-installer pattern
func stepPKI(cfg *Config) error {
	log.Println("step 5/13: configuring PKI with Ziti...")

	pkiDir := cfg.PKIDir

	// Clean up old PKI state (ziti pki create fails if CAs already exist)
	if err := os.RemoveAll(pkiDir); err != nil {
		return fmt.Errorf("cleanup pki dir: %w", err)
	}

	if err := os.MkdirAll(pkiDir, 0755); err != nil {
		return fmt.Errorf("mkdir pki: %w", err)
	}

	// Generate PKI using ziti pki commands (which support SPIFFE URIs)
	// Based on: kore/kontango-installer/modules/lib/ziti.sh

	// 1. Root CA
	log.Println("  → generating root CA...")
	cmd := exec.Command("ziti", "pki", "create", "ca",
		"--pki-root", pkiDir,
		"--ca-name", "root-ca",
		"--ca-file", "root-ca",
		"--trust-domain", cfg.Domain)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("generate root ca: %s", string(out))
	}

	// 2. Intermediate CA
	log.Println("  → generating intermediate CA...")
	intName := fmt.Sprintf("intermediate-%s", cfg.Name)
	cmd = exec.Command("ziti", "pki", "create", "intermediate",
		"--pki-root", pkiDir,
		"--ca-name", "root-ca",
		"--intermediate-name", intName,
		"--intermediate-file", intName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("generate intermediate: %s", string(out))
	}

	// 3. Server certificate with SPIFFE URI (for Ziti)
	// Note: In production, this is self-signed SPIFFE-URI cert for Ziti mesh identity
	// Real LE certs come from Caddy/Cloudflare DNS-01 at runtime
	log.Println("  → generating server certificate...")
	sanDomains := fmt.Sprintf("%s,%s.%s,*.%s", cfg.Domain, cfg.Name, cfg.Domain, cfg.Domain)
	spiffeID := fmt.Sprintf("spiffe://%s/controller/%s", cfg.Domain, cfg.Name)
	cmd = exec.Command("ziti", "pki", "create", "server",
		"--pki-root", pkiDir,
		"--ca-name", intName,
		"--server-name", cfg.Name,
		"--server-file", "server",
		"--dns", sanDomains,
		"--ip", fmt.Sprintf("%s,127.0.0.1", cfg.NodePublicIP),
		"--spiffe-id", spiffeID)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("generate server cert: %s", string(out))
	}

	// 4. Client certificate
	log.Println("  → generating client certificate...")
	cmd = exec.Command("ziti", "pki", "create", "client",
		"--pki-root", pkiDir,
		"--ca-name", intName,
		"--client-name", "client",
		"--client-file", "client")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("generate client cert: %s", string(out))
	}

	// 5. Build certificate chains for use in Ziti config
	log.Println("  → building certificate chains...")
	intDir := filepath.Join(pkiDir, intName)

	// Server chain: server.cert + intermediate + root
	serverChainData, err := buildCertChain(
		filepath.Join(intDir, "certs", "server.cert"),
		filepath.Join(intDir, "certs", fmt.Sprintf("%s.cert", intName)),
		filepath.Join(pkiDir, "root-ca", "certs", "root-ca.cert"),
	)
	if err != nil {
		return fmt.Errorf("build server chain: %w", err)
	}

	// CA bundle: intermediate + root
	caBundleData, err := buildCertChain(
		filepath.Join(intDir, "certs", fmt.Sprintf("%s.cert", intName)),
		filepath.Join(pkiDir, "root-ca", "certs", "root-ca.cert"),
	)
	if err != nil {
		return fmt.Errorf("build ca bundle: %w", err)
	}

	// 6. Write chains to /etc/kontango/pki for use in config
	log.Println("  → writing certificates to disk...")
	etcPkiDir := filepath.Join(cfg.EtcDir, "pki")
	if err := os.MkdirAll(etcPkiDir, 0755); err != nil {
		return fmt.Errorf("mkdir etc/pki: %w", err)
	}

	// Write server files
	serverCertPath := filepath.Join(intDir, "certs", "server.cert")
	serverKeyPath := filepath.Join(intDir, "keys", "server.key")
	intKeyPath := filepath.Join(intDir, "keys", fmt.Sprintf("%s.key", intName))

	// Copy/symlink server cert
	if err := copyFile(serverCertPath, filepath.Join(etcPkiDir, "server.crt")); err != nil {
		return fmt.Errorf("copy server cert: %w", err)
	}

	// Copy/symlink server key
	if err := copyFile(serverKeyPath, filepath.Join(etcPkiDir, "server.key")); err != nil {
		return fmt.Errorf("copy server key: %w", err)
	}

	// Copy/symlink intermediate key (for signing)
	if err := copyFile(intKeyPath, filepath.Join(etcPkiDir, "intermediate.key")); err != nil {
		return fmt.Errorf("copy intermediate key: %w", err)
	}

	// Write server chain
	if err := os.WriteFile(filepath.Join(etcPkiDir, "server.crt"), serverChainData, 0644); err != nil {
		return fmt.Errorf("write server chain: %w", err)
	}

	// Write signing chain (for enrollment)
	if err := os.WriteFile(filepath.Join(etcPkiDir, "signing-chain.crt"), caBundleData, 0644); err != nil {
		return fmt.Errorf("write signing chain: %w", err)
	}

	// Write CA bundle
	if err := os.WriteFile(filepath.Join(etcPkiDir, "ca-bundle.pem"), caBundleData, 0644); err != nil {
		return fmt.Errorf("write ca bundle: %w", err)
	}

	log.Println("  ✓ PKI generated successfully")
	return nil
}

// buildCertChain concatenates certificates in order
func buildCertChain(certPaths ...string) ([]byte, error) {
	var chain []byte

	for _, certPath := range certPaths {
		certData, err := os.ReadFile(certPath)
		if err != nil {
			return nil, fmt.Errorf("read cert %s: %w", certPath, err)
		}
		chain = append(chain, certData...)
		chain = append(chain, '\n')
	}

	return chain, nil
}

// copyFile copies a file from src to dst, preserving content
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dest: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	return nil
}

// stepPKIFromLeader fetches PKI from leader's Bao KV (for join nodes)
func stepPKIFromLeader(cfg *Config) error {
	log.Println("step 4b/13: fetching PKI from leader...")

	// For join nodes, PKI is already on disk from step 3 bao-init
	// Just verify the certs exist and are readable
	etcPkiDir := filepath.Join(cfg.EtcDir, "pki")

	requiredFiles := []string{
		"root-ca.crt",
		"intermediate.crt",
		"intermediate.key",
		"signing-chain.crt",
		"server.crt",
		"server.key",
	}

	// Check if all required files exist
	allExist := true
	for _, file := range requiredFiles {
		path := filepath.Join(etcPkiDir, file)
		if _, err := os.Stat(path); err != nil {
			allExist = false
			log.Printf("  ⚠ missing %s: %v", file, err)
		}
	}

	if !allExist {
		// PKI not yet available from leader; this will be populated when Bao syncs
		log.Println("  ⚠ PKI files not yet available (Bao still syncing from leader)")
		log.Println("  → waiting for Bao to replicate PKI...")

		// Poll for PKI files to appear (up to 60s)
		deadline := time.Now().Add(60 * time.Second)
		for time.Now().Before(deadline) {
			allExist = true
			for _, file := range requiredFiles {
				path := filepath.Join(etcPkiDir, file)
				if _, err := os.Stat(path); err != nil {
					allExist = false
					break
				}
			}
			if allExist {
				log.Println("  ✓ PKI files replicated from leader")
				break
			}
			time.Sleep(1 * time.Second)
		}

		if !allExist {
			return fmt.Errorf("PKI files not replicated from leader within 60s")
		}
	}

	log.Println("  ✓ PKI verified")
	return nil
}

// loadBaoInit loads the temporary bao-init.json from step 4
func loadBaoInit(cfg *Config) (rootToken, unsealKey string, err error) {
	initPath := filepath.Join(cfg.EtcDir, "bao-init.json")
	log.Printf("    DEBUG: reading init from %s\n", initPath)
	data, err := os.ReadFile(initPath)
	if err != nil {
		return "", "", fmt.Errorf("read init: %w", err)
	}

	log.Printf("    DEBUG: init file content: %s\n", string(data))
	var init map[string]string
	if err := json.Unmarshal(data, &init); err != nil {
		return "", "", fmt.Errorf("unmarshal init: %w", err)
	}

	return init["root_token"], init["unseal_key"], nil
}
