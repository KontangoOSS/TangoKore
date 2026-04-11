package controller

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"time"

	"github.com/KontangoOSS/TangoKore/internal/controller/clients"
)

// stepBaoInit initializes OpenBao: install, init/unseal, enable audit
func stepBaoInit(cfg *Config) error {
	log.Println("step 4/13: initializing OpenBao...")

	// 1. Create data directory
	baoDataDir := filepath.Join(cfg.Home, "data", "bao")
	if err := os.MkdirAll(baoDataDir, 0755); err != nil {
		return fmt.Errorf("mkdir bao data: %w", err)
	}

	// 2. Generate openbao.hcl config
	log.Println("  → generating Bao config...")
	if err := generateBaoConfig(cfg); err != nil {
		return fmt.Errorf("generate config: %w", err)
	}

	// 3. Install and start Bao systemd service
	log.Println("  → installing systemd service...")
	if err := installBaoService(cfg); err != nil {
		return fmt.Errorf("install systemd service: %w", err)
	}

	// 4. Start Bao
	log.Println("  → starting Bao...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "systemctl", "restart", "kontango-bao")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("systemctl restart: %w", err)
	}

	// Wait for Bao to be ready
	log.Println("  → waiting for Bao API (up to 30s)...")
	if !waitForPort("127.0.0.1:8200", 30*time.Second) {
		return fmt.Errorf("bao API not responding on port 8200")
	}

	// 5. Initialize or join Bao
	_, unsealKey, rootToken, err := initOrJoinBao(cfg)
	if err != nil {
		return fmt.Errorf("init/join: %w", err)
	}

	// 6. Write init state to disk (temp, will be moved to KV in step 6)
	log.Println("  → saving init credentials...")
	initPath := filepath.Join(cfg.EtcDir, "bao-init.json")

	// If already initialized (no unseal key returned), retrieve init from existing file or fail
	if unsealKey == "" {
		// For fresh bootstrap, if Bao claims to be initialized but we have no creds,
		// we need to either recover from backup or reset Bao
		existingInit, _ := os.ReadFile(initPath)
		if existingInit != nil && len(existingInit) > 0 {
			// Use existing init file
			log.Println("  ⚠ using existing Bao init credentials")
		} else {
			// Bao is initialized but we have no credentials — this is a problem
			// In this case, the operator needs to either:
			// 1. Provide the unseal key and root token manually
			// 2. Recover from backup
			// 3. Wipe Bao and restart
			return fmt.Errorf("Bao is initialized but no credentials found. Reset with: sudo systemctl stop kontango-bao && sudo rm -rf /var/lib/openbao && sudo rm -f %s", initPath)
		}
	} else {
		// Fresh init — save credentials
		initJSON := fmt.Sprintf(`{"unseal_key":"%s","root_token":"%s"}`, unsealKey, rootToken)
		if err := os.WriteFile(initPath, []byte(initJSON), 0600); err != nil {
			return fmt.Errorf("write init: %w", err)
		}
	}

	log.Println("  ✓ OpenBao initialized and ready")
	return nil
}

// generateBaoConfig writes the openbao.hcl configuration file
func generateBaoConfig(cfg *Config) error {
	configTmpl := `ui = true
disable_mlock = true

storage "raft" {
  path    = "{{.DataDir}}"
  node_id = "{{.NodeName}}"
}

cluster_addr = "https://{{.NodeName}}.{{.Domain}}:8201"
api_addr     = "https://{{.NodeName}}.{{.Domain}}:8200"

listener "tcp" {
  address         = "127.0.0.1:8200"
  tls_cert_file   = "{{.PKIDir}}/server.crt"
  tls_key_file    = "{{.PKIDir}}/server.key"
  cluster_address = "0.0.0.0:8201"
}
`

	tmpl, err := template.New("openbao.hcl").Parse(configTmpl)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	data := map[string]string{
		"DataDir":   filepath.Join(cfg.Home, "data", "bao"),
		"NodeName":  cfg.Name,
		"Domain":    cfg.Domain,
		"PKIDir":    cfg.PKIDir,
	}

	outPath := filepath.Join(cfg.EtcDir, "openbao.hcl")
	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("create config: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return nil
}

// installBaoService writes the systemd unit file
func installBaoService(cfg *Config) error {
	serviceTmpl := `[Unit]
Description=OpenBao Secret Store
After=network-online.target
Requires=network-online.target

[Service]
Type=simple
User=root
Group=root
ExecStart=/usr/local/bin/bao server -config={{.EtcDir}}/openbao.hcl
StandardOutput=journal
StandardError=journal
SyslogIdentifier=openbao
Restart=on-failure
RestartSec=5s
StartLimitIntervalSec=300
StartLimitBurst=5
LimitNOFILE=65536
LimitMEMLOCK=infinity
CapabilityBoundingSet=CAP_IPC_LOCK

[Install]
WantedBy=multi-user.target
`

	tmpl, err := template.New("openbao.service").Parse(serviceTmpl)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	data := map[string]string{
		"EtcDir": cfg.EtcDir,
	}

	outPath := "/etc/systemd/system/kontango-bao.service"
	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("create service: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	// Reload systemd and enable service
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}

	cmd = exec.CommandContext(ctx, "systemctl", "enable", "kontango-bao")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("enable service: %w", err)
	}

	return nil
}

// initOrJoinBao initializes Bao (init mode) or joins a cluster (join mode)
// Returns the initialized BaoClient, unseal key, and root token
func initOrJoinBao(cfg *Config) (*clients.BaoClient, string, string, error) {
	// Create client with self-signed cert (will be updated in step 5 when real cert is issued)
	client, err := clients.NewBaoClient("https://127.0.0.1:8200", "", "")
	if err != nil {
		return nil, "", "", fmt.Errorf("create client: %w", err)
	}

	if !cfg.JoinMode {
		// Init mode: initialize a new Bao cluster
		// Check if already initialized
		initialized, sealed, err := client.SealStatus()
		if err == nil && initialized {
			// Already initialized
			if !sealed {
				// Already initialized and unsealed — skip init
				log.Println("  ⚠ Bao already initialized and unsealed — skipping init")
				return client, "", "", nil
			} else {
				// Already initialized but sealed — need unseal key to proceed
				log.Println("  ⚠ Bao already initialized but sealed — requires unseal key")
				return client, "", "", fmt.Errorf("Bao is sealed. Provide unseal key or reset with: sudo systemctl stop kontango-bao && sudo rm -rf /var/lib/openbao")
			}
		}

		// Not initialized — initialize with 1 key, threshold 1 (unsealed immediately for single-node setup)
		unsealKey, rootToken, err := client.Init(1, 1)
		if err != nil {
			return nil, "", "", fmt.Errorf("init: %w", err)
		}

		// Unseal immediately
		if err := client.Unseal(unsealKey); err != nil {
			return nil, "", "", fmt.Errorf("unseal: %w", err)
		}

		return client.WithToken(rootToken), unsealKey, rootToken, nil

	} else {
		// Join mode: join an existing Bao cluster
		log.Printf("  → joining Bao cluster at %s\n", cfg.JoinLeader)

		// Use raft join via bao CLI (not yet available in client)
		// For now, just return an error indicating this needs CLI implementation
		return nil, "", "", fmt.Errorf("join mode requires CLI implementation (TODO: add bao operator raft join)")
	}
}

// waitForPort waits for a TCP port to be open
func waitForPort(addr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}
