package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/KontangoOSS/TangoKore/internal/controller"
)

func cmdController(args []string) {
	if len(args) < 1 {
		printControllerUsage()
		os.Exit(1)
	}

	subcommand := args[0]
	switch subcommand {
	case "install":
		cmdControllerInstall(args[1:])
	case "create":
		cmdControllerCreate(args[1:])
	case "status":
		cmdControllerStatus(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown controller subcommand: %s\n", subcommand)
		printControllerUsage()
		os.Exit(1)
	}
}

func cmdControllerInstall(args []string) {
	fs := flag.NewFlagSet("controller install", flag.ExitOnError)

	// Core
	name := fs.String("name", "ctrl-1", "node name")
	zitiPass := fs.String("ziti-pass", "", "Ziti admin password (required)")
	domain := fs.String("domain", "", "base domain (required in production)")

	// Test mode
	testMode := fs.Bool("test", false, "self-signed certs, no DNS required")

	// Join mode
	joinLeader := fs.String("join", "", "SSH to leader for join mode")
	joinUnsealKey := fs.String("join-bao-unseal", "", "Bao unseal key from leader")

	// TLS
	cfToken := fs.String("cf-token", "", "Cloudflare API token")
	acmeEmail := fs.String("acme-email", "", "ACME certificate email")

	// Versions
	zitiVer := fs.String("ziti-version", "1.6.15", "Ziti version")
	baoVer := fs.String("bao-version", "2.5.2", "OpenBao version")

	// Paths
	home := fs.String("home", "/opt/kontango", "install root directory")
	etcDir := fs.String("etc", "/etc/kontango", "config directory")

	// Other
	overlayDomain := fs.String("overlay", "tango", "overlay DNS domain")

	fs.Parse(args)

	log.SetFlags(0)

	// Validate required fields
	if *zitiPass == "" {
		fmt.Fprintf(os.Stderr, "error: --ziti-pass is required\n")
		os.Exit(1)
	}

	if !*testMode && *domain == "" {
		fmt.Fprintf(os.Stderr, "error: --domain is required in production mode\n")
		os.Exit(1)
	}

	if !*testMode && *cfToken == "" {
		fmt.Fprintf(os.Stderr, "error: --cf-token is required in production mode\n")
		os.Exit(1)
	}

	if !*testMode && *acmeEmail == "" {
		fmt.Fprintf(os.Stderr, "error: --acme-email is required in production mode\n")
		os.Exit(1)
	}

	// Set default domain in test mode
	if *testMode && *domain == "" {
		*domain = "kontango.local"
	}

	// Create config
	cfg := &controller.Config{
		Name:           *name,
		Domain:         *domain,
		JoinDomain:     "join." + *domain,
		OverlayDomain:  *overlayDomain,
		ZitiAdminUser:  "admin",
		ZitiAdminPass:  *zitiPass,
		CloudflareToken: *cfToken,
		ACMEEmail:      *acmeEmail,
		TestMode:       *testMode,
		ZitiVersion:    *zitiVer,
		BaoVersion:     *baoVer,
		Home:           *home,
		EtcDir:         *etcDir,
		JoinMode:       *joinLeader != "",
		JoinLeader:     *joinLeader,
		JoinBaoUnsealKey: *joinUnsealKey,
	}

	// Run installation
	if err := controller.Install(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "installation failed: %v\n", err)
		os.Exit(1)
	}
}

func cmdControllerCreate(args []string) {
	fs := flag.NewFlagSet("controller create", flag.ExitOnError)
	name := fs.String("name", "", "controller cluster name")
	domain := fs.String("domain", "example.com", "base domain for the cluster")
	fs.Parse(args)

	log.SetFlags(0)

	// Validate required fields
	if *name == "" {
		fmt.Fprintf(os.Stderr, "error: --name is required\n")
		printControllerUsage()
		os.Exit(1)
	}

	log.Printf("creating controller cluster '%s' on domain '%s'…\n", *name, *domain)
	log.Printf("\ncontroller create is not yet fully implemented")
	log.Printf("the infrastructure setup logic exists elsewhere in the codebase\n")
	log.Printf("\nthis command will eventually:")
	log.Printf("  1. Deploy Ziti overlay mesh")
	log.Printf("  2. Deploy OpenBao secrets management")
	log.Printf("  3. Deploy schmutz-controller (HA cluster)")
	log.Printf("  4. Configure initial NATS message bus")
	log.Printf("  5. Bootstrap Raft consensus\n")
}

func cmdControllerStatus(args []string) {
	fs := flag.NewFlagSet("controller status", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "output as JSON")
	fs.Parse(args)

	log.SetFlags(0)

	// Check if controller config exists
	// For now, just return a status indicating if we're in a controlled state
	status := map[string]interface{}{
		"configured": false,
		"healthy":    false,
		"reason":     "controller not configured",
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(status)
	} else {
		fmt.Printf("controller status: not configured\n")
		fmt.Printf("run: kontango controller create --name <name>\n")
	}
}

func printControllerUsage() {
	fmt.Fprintf(os.Stderr, `usage: kontango controller <subcommand> [flags]

subcommands:
  install [flags]  bootstrap a new controller node
  create [flags]   deploy a new HA controller cluster
  status [flags]   show controller status

flags for 'install':
  --name <name>              node name (default: ctrl-1)
  --ziti-pass <pass>         Ziti admin password (required)
  --domain <domain>          base domain (required in production)
  --test                     self-signed certs, no DNS required
  --join <leader>            SSH to leader for join mode
  --join-bao-unseal <key>    Bao unseal key from leader
  --cf-token <token>         Cloudflare API token (production only)
  --acme-email <email>       ACME certificate email (production only)
  --ziti-version <ver>       Ziti version (default: 1.6.15)
  --bao-version <ver>        OpenBao version (default: 2.5.2)
  --home <dir>               install root (default: /opt/kontango)
  --etc <dir>                config directory (default: /etc/kontango)
  --overlay <domain>         overlay domain (default: tango)

flags for 'create':
  --name <name>       cluster name (required)
  --domain <domain>   base domain (default: example.com)

flags for 'status':
  --json              output as JSON

examples:
  kontango controller install --test --name ctrl-test --ziti-pass TestPass123
  kontango controller install --name ctrl-1 --domain example.com \
    --ziti-pass <pass> --cf-token <tok> --acme-email ops@example.com
  kontango controller create --name prod --domain example.com
  kontango controller status --json
`)
}
