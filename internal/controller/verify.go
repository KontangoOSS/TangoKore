package controller

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/KontangoOSS/TangoKore/internal/controller/clients"
)

// stepVerify checks health of all bootstrapped services
func stepVerify(cfg *Config) error {
	log.Println("step 13/13: verifying system health...")

	// Different verification based on node role
	if cfg.JoinMode {
		return verifyEdgeRouter(cfg)
	}
	return verifyController(cfg)
}

// verifyController checks controller node health
func verifyController(cfg *Config) error {
	log.Println("  → verifying controller services...")

	checks := []struct {
		name string
		fn   func() error
	}{
		{"Ziti controller", checkZitiController(cfg)},
		{"Bao vault", checkBaoVault(cfg)},
		{"Schmutz controller", checkSchmutzController(cfg)},
	}

	allPass := true
	for _, c := range checks {
		if err := c.fn(); err != nil {
			log.Printf("    ✗ %s: %v", c.name, err)
			allPass = false
		} else {
			log.Printf("    ✓ %s", c.name)
		}
	}

	if !allPass {
		return fmt.Errorf("some health checks failed")
	}

	log.Println("  ✓ all systems operational")
	return nil
}

// verifyEdgeRouter checks edge router node health
func verifyEdgeRouter(cfg *Config) error {
	log.Println("  → verifying edge router services...")

	checks := []struct {
		name string
		fn   func() error
	}{
		{"Ziti router", checkZitiRouter()},
		{"Caddy", checkCaddy(cfg)},
		{"Schmutz gateway", checkSchmutzGateway(cfg)},
	}

	allPass := true
	for _, c := range checks {
		if err := c.fn(); err != nil {
			log.Printf("    ✗ %s: %v", c.name, err)
			allPass = false
		} else {
			log.Printf("    ✓ %s", c.name)
		}
	}

	if !allPass {
		return fmt.Errorf("some health checks failed")
	}

	log.Println("  ✓ all systems operational")
	return nil
}

// checkZitiController verifies Ziti controller API
func checkZitiController(cfg *Config) func() error {
	return func() error {
		zitiClient, err := clients.NewZitiClient(
			fmt.Sprintf("https://127.0.0.1:%d/edge/management/v1", cfg.ZitiCtrlPort),
			cfg.ZitiAdminUser,
			cfg.ZitiAdminPass,
		)
		if err != nil {
			return fmt.Errorf("client: %w", err)
		}

		// Try to list edge routers as a health check
		_, err = zitiClient.ListEdgeRouters()
		return err
	}
}

// checkZitiRouter verifies Ziti router is running
func checkZitiRouter() func() error {
	return func() error {
		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		resp, err := client.Get("https://127.0.0.1:3022/version")
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status %d", resp.StatusCode)
		}
		return nil
	}
}

// checkBaoVault verifies Bao vault API
func checkBaoVault(cfg *Config) func() error {
	return func() error {
		rootToken, _, err := loadBaoInit(cfg)
		if err != nil {
			return fmt.Errorf("load init: %w", err)
		}

		baoClient, err := clients.NewBaoClient("https://127.0.0.1:8200", rootToken, "")
		if err != nil {
			return fmt.Errorf("client: %w", err)
		}

		sealed, unsealed, err := baoClient.SealStatus()
		if err != nil {
			return fmt.Errorf("seal status: %w", err)
		}

		if !unsealed || sealed {
			return fmt.Errorf("vault is sealed")
		}
		return nil
	}
}

// checkCaddy verifies Caddy reverse proxy
func checkCaddy(cfg *Config) func() error {
	return func() error {
		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		// Skip TLS verification for test mode
		url := fmt.Sprintf("https://127.0.0.1:443/health")
		if cfg.TestMode {
			// In test mode, use self-signed cert
			transport := &http.Transport{}
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			client.Transport = transport
		}

		resp, err := client.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status %d", resp.StatusCode)
		}
		return nil
	}
}

// checkSchmutzController verifies schmutz controller
func checkSchmutzController(cfg *Config) func() error {
	return func() error {
		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		url := fmt.Sprintf("http://127.0.0.1:%d/health", cfg.SchmutzPort)
		resp, err := client.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status %d", resp.StatusCode)
		}

		// Parse response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read body: %w", err)
		}

		var health map[string]interface{}
		if err := json.Unmarshal(body, &health); err != nil {
			return fmt.Errorf("parse: %w", err)
		}

		status, ok := health["status"].(string)
		if !ok || status != "healthy" {
			return fmt.Errorf("not healthy")
		}
		return nil
	}
}

// checkSchmutzGateway verifies schmutz gateway
func checkSchmutzGateway(cfg *Config) func() error {
	return func() error {
		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		url := fmt.Sprintf("http://127.0.0.1:%d/health", cfg.SchmutzPort)
		resp, err := client.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status %d", resp.StatusCode)
		}
		return nil
	}
}
