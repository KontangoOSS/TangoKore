package controller

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"time"

	"github.com/KontangoOSS/TangoKore/internal/controller/clients"
)

// stepZiti initializes the Ziti controller and edge router
func stepZiti(cfg *Config) error {
	log.Println("step 7/13: configuring Ziti controller and router...")

	// 1. Generate Ziti controller config
	log.Println("  → generating controller config...")
	if err := generateZitiControllerConfig(cfg); err != nil {
		return fmt.Errorf("generate config: %w", err)
	}

	// 2. Install systemd services
	log.Println("  → installing systemd services...")
	if err := installZitiServices(cfg); err != nil {
		return fmt.Errorf("install services: %w", err)
	}

	// 3. Start Ziti controller
	log.Println("  → starting Ziti controller...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "systemctl", "restart", "kontango-ziti-controller")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("start controller: %w", err)
	}

	// 4. Wait for controller to be ready
	log.Println("  → waiting for Ziti controller (up to 60s)...")
	if !waitForPort(fmt.Sprintf("127.0.0.1:%d", cfg.ZitiCtrlPort), 60*time.Second) {
		return fmt.Errorf("ziti controller not responding on port %d", cfg.ZitiCtrlPort)
	}

	// 5. Create Ziti client and login
	log.Println("  → authenticating with Ziti controller...")
	zitiClient, err := clients.NewZitiClient(
		fmt.Sprintf("https://127.0.0.1:%d/edge/management/v1", cfg.ZitiCtrlPort),
		cfg.ZitiAdminUser,
		cfg.ZitiAdminPass,
	)
	if err != nil {
		return fmt.Errorf("create ziti client: %w", err)
	}

	// 6. Verify controller is operational
	_, err = zitiClient.ListEdgeRouters()
	if err != nil {
		return fmt.Errorf("verify controller operational: %w", err)
	}

	log.Println("  ✓ Ziti controller initialized and operational")
	return nil
}

// generateZitiControllerConfig writes the Ziti controller configuration
func generateZitiControllerConfig(cfg *Config) error {
	configTmpl := `# Ziti Controller Configuration
v: 3

identity:
  cert: {{.CertFile}}
  server_cert: {{.CertFile}}
  key: {{.KeyFile}}
  ca: {{.CAFile}}

controllers:
  edge:
    bindPoints:
      - address: tls:0.0.0.0:{{.EdgePort}}
      - address: wss:0.0.0.0:{{.WebsocketPort}}
    options:
      minVersion: TLS12
      maxVersion: TLS12
  fabric:
    bindPoints:
      - address: tls:0.0.0.0:{{.LinkPort}}

listeners:
  - binding: edge
    address: tls:0.0.0.0:{{.EdgePort}}
    options:
      advertise: {{.Domain}}:{{.EdgePort}}
  - binding: fabric
    address: tls:0.0.0.0:{{.LinkPort}}
    options:
      advertise: {{.Domain}}:{{.LinkPort}}

database:
  type: bbolt
  path: {{.DataDir}}/ziti/db.bbolt

events:
  logForwarder:
    enabled: true

edge:
  api:
    activityUpdateBatchSize: 100
    activityUpdateInterval: 5s
  enrollmentDuration: 15m
`

	tmpl, err := template.New("ctrl.yaml").Parse(configTmpl)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	data := map[string]interface{}{
		"CertFile":      filepath.Join(cfg.EtcDir, "pki", "server.crt"),
		"KeyFile":       filepath.Join(cfg.EtcDir, "pki", "server.key"),
		"CAFile":        filepath.Join(cfg.EtcDir, "pki", "ca-bundle.pem"),
		"EdgePort":      cfg.ZitiEdgePort,
		"LinkPort":      cfg.ZitiLinkPort,
		"WebsocketPort": 8081,
		"Domain":        cfg.Domain,
		"DataDir":       cfg.Home,
	}

	zitiDir := filepath.Join(cfg.EtcDir, "ziti")
	if err := os.MkdirAll(zitiDir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	configPath := filepath.Join(zitiDir, "ctrl.yaml")
	f, err := os.OpenFile(configPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0640)
	if err != nil {
		return fmt.Errorf("create config: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return nil
}

// installZitiServices writes systemd unit files for Ziti controller and router
func installZitiServices(cfg *Config) error {
	controllerServiceTmpl := `[Unit]
Description=OpenZiti Controller
After=network-online.target
Requires=network-online.target

[Service]
Type=simple
User=root
Group=root
ExecStart=/usr/local/bin/ziti controller run {{.ConfDir}}/ziti/ctrl.yaml
StandardOutput=journal
StandardError=journal
SyslogIdentifier=ziti-controller
Restart=on-failure
RestartSec=5s
StartLimitIntervalSec=300
StartLimitBurst=5

[Install]
WantedBy=multi-user.target
`

	routerServiceTmpl := `[Unit]
Description=OpenZiti Edge Router
After=network-online.target ziti-controller.service
Requires=network-online.target

[Service]
Type=simple
User=root
Group=root
ExecStart=/usr/local/bin/ziti router run {{.ConfDir}}/ziti/router.yaml
StandardOutput=journal
StandardError=journal
SyslogIdentifier=ziti-router
Restart=on-failure
RestartSec=5s
StartLimitIntervalSec=300
StartLimitBurst=5

[Install]
WantedBy=multi-user.target
`

	services := map[string]string{
		"kontango-ziti-controller": controllerServiceTmpl,
		"kontango-ziti-router":     routerServiceTmpl,
	}

	tmplData := map[string]string{
		"ConfDir": cfg.EtcDir,
	}

	for name, tmplStr := range services {
		tmpl, err := template.New(name).Parse(tmplStr)
		if err != nil {
			return fmt.Errorf("parse %s: %w", name, err)
		}

		outPath := filepath.Join("/etc/systemd/system", name+".service")
		f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("create %s: %w", name, err)
		}
		if err := tmpl.Execute(f, tmplData); err != nil {
			f.Close()
			return fmt.Errorf("execute %s: %w", name, err)
		}
		f.Close()
	}

	// Reload systemd
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}

	// Enable services
	cmd = exec.CommandContext(ctx, "systemctl", "enable", "kontango-ziti-controller", "kontango-ziti-router")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("enable services: %w", err)
	}

	return nil
}
