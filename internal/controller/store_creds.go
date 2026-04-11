package controller

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/KontangoOSS/TangoKore/internal/controller/clients"
)

// stepStoreCreds stores all credentials in Bao KV v2
func stepStoreCreds(cfg *Config) error {
	log.Println("step 6/13: storing credentials in Bao...")

	// Load Bao credentials from step 4
	rootToken, _, err := loadBaoInit(cfg)
	if err != nil {
		return fmt.Errorf("load bao init: %w", err)
	}

	client, err := clients.NewBaoClient("https://127.0.0.1:8200", rootToken, "")
	if err != nil {
		return fmt.Errorf("create bao client: %w", err)
	}

	// 1. Enable KV v2 secret engine
	log.Println("  → enabling KV v2 engine...")
	if err := client.EnableEngine("secret", "kv-v2"); err != nil {
		return fmt.Errorf("enable kv v2: %w", err)
	}

	// 2. Write Bao init data (unseal key + root token)
	log.Println("  → storing Bao init data...")
	initData := map[string]interface{}{
		"unseal_key": cfg.BaoUnsealKey,
		"root_token": cfg.BaoRootToken,
	}
	if err := client.KVPut("secret", "kontango/bao/init", initData); err != nil {
		return fmt.Errorf("store bao init: %w", err)
	}

	// 3. Write Ziti admin credentials
	log.Println("  → storing Ziti admin credentials...")
	zitiData := map[string]interface{}{
		"username": cfg.ZitiAdminUser,
		"password": cfg.ZitiAdminPass,
	}
	if err := client.KVPut("secret", "kontango/ziti/admin", zitiData); err != nil {
		return fmt.Errorf("store ziti admin: %w", err)
	}

	// 4. Write PKI certificates for cross-node reference
	log.Println("  → storing PKI certificates...")
	rootCertPath := filepath.Join(cfg.EtcDir, "pki", "root-ca.crt")
	intCertPath := filepath.Join(cfg.EtcDir, "pki", "intermediate.crt")

	rootCert, err := os.ReadFile(rootCertPath)
	if err != nil {
		return fmt.Errorf("read root cert: %w", err)
	}
	intCert, err := os.ReadFile(intCertPath)
	if err != nil {
		return fmt.Errorf("read int cert: %w", err)
	}

	pkiData := map[string]interface{}{
		"root_cert": string(rootCert),
		"int_cert":  string(intCert),
	}
	if err := client.KVPut("secret", "kontango/pki/ca", pkiData); err != nil {
		return fmt.Errorf("store pki certs: %w", err)
	}

	// 5. Create Caddy cert storage policy
	log.Println("  → creating Caddy cert storage policy...")
	caddyPolicy := `path "secret/data/caddy/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
path "secret/metadata/caddy/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}`

	if err := client.CreatePolicy("caddy-storage", caddyPolicy); err != nil {
		return fmt.Errorf("create caddy policy: %w", err)
	}

	// 6. Create Caddy token (long-lived, renewable)
	log.Println("  → creating Caddy token...")
	caddyToken, err := client.CreateAppRoleSecret("caddy-storage")
	if err != nil {
		// If CreateAppRoleSecret fails, try creating AppRole first, then secret
		log.Println("  → creating AppRole 'caddy-storage'...")
		if err := client.CreateAppRoleRole("caddy-storage", []string{"caddy-storage"}, "8760h"); err != nil {
			return fmt.Errorf("create caddy approle: %w", err)
		}

		caddyToken, err = client.CreateAppRoleSecret("caddy-storage")
		if err != nil {
			return fmt.Errorf("create caddy token: %w", err)
		}
	}

	// 7. Write Caddy token to disk
	log.Println("  → saving Caddy token...")
	caddyTokenPath := filepath.Join(cfg.EtcDir, "caddy-bao-token")
	if err := os.WriteFile(caddyTokenPath, []byte(caddyToken), 0600); err != nil {
		return fmt.Errorf("write caddy token: %w", err)
	}

	log.Println("  ✓ credentials stored in Bao")
	return nil
}
