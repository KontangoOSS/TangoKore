package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-isatty"

	"github.com/KontangoOSS/TangoKore/internal/enroll"
)

func cmdEnroll(args []string) {
	fs := flag.NewFlagSet("enroll", flag.ExitOnError)
	roleID := fs.String("role-id", "", "Bao AppRole role ID")
	secretID := fs.String("secret-id", "", "Bao AppRole secret ID")
	session := fs.String("session", "", "one-time session token from install script")
	scan := fs.Bool("scan", false, "re-enroll a known machine (uses fingerprint matching)")
	noTUI := fs.Bool("no-tui", false, "disable interactive wizard (non-interactive mode)")
	fs.Parse(args)

	// URL is optional in TUI mode — the wizard will prompt for it.
	url := ""
	if fs.NArg() >= 1 {
		url = fs.Arg(0)
	}

	// Use the TUI wizard when:
	//   - stdin is a real terminal (not a pipe / curl | sh)
	//   - --no-tui was not passed
	//   - we are not being called by the agent (agent always passes --no-tui)
	interactive := isatty.IsTerminal(os.Stdin.Fd()) && !*noTUI

	if interactive {
		// TUI path — wizard handles URL prompt, method selection, creds, confirm.
		result, err := runEnrollTUI(url, *session, *roleID, *secretID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "enroll: %v\n", err)
			os.Exit(1)
		}
		if result == nil {
			// User quit the wizard
			os.Exit(0)
		}
		// Install services after wizard completes enrollment
		if err := installAfterEnroll(url, result); err != nil {
			fmt.Fprintf(os.Stderr, "install: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Non-interactive path (pipe, script, agent-triggered)
	if url == "" {
		fmt.Fprintf(os.Stderr, "usage: kontango enroll <url> [flags]\n")
		os.Exit(1)
	}
	if err := runEnroll(url, *session, *roleID, *secretID, *scan); err != nil {
		fmt.Fprintf(os.Stderr, "enroll: %v\n", err)
		os.Exit(1)
	}
}

// runEnroll is the full non-interactive enrollment flow.
// Called by cmdEnroll (non-TTY) and by the agent when it receives an enroll command.
func runEnroll(url, session, roleID, secretID string, scanMethod bool) error {
	log.SetFlags(0)

	platform, err := enroll.DetectPlatform()
	if err != nil {
		return fmt.Errorf("unsupported platform: %w", err)
	}
	if missing := platform.Preflight(); len(missing) > 0 {
		log.Println("preflight failed:")
		for _, m := range missing {
			log.Printf("  - %s", m)
		}
		return fmt.Errorf("preflight failed")
	}

	// Enrollment: Send machine data to the endpoint.
	// The server determines the method (new/scan/trusted) based on:
	// - Machine fingerprint (has it enrolled before?)
	// - Credentials provided (AppRole, JWT, session token)
	// - Server policy and decision pipeline
	//
	// If credentials are provided, the server will validate them.
	// Otherwise, the machine will be processed as new or returning (fingerprint match).

	log.Println("═══════════════════════════════════════════════════════════════")
	log.Println("MACHINE FINGERPRINTING DISCLOSURE")
	log.Println("═══════════════════════════════════════════════════════════════")
	log.Println("")
	log.Println("This machine will send hardware information to identify itself:")
	log.Println("  • Hostname, OS version, kernel, architecture")
	log.Println("  • CPU model/cores, system memory, motherboard ID")
	log.Println("  • Network interface MAC addresses")
	log.Println("")
	log.Println("Why: Allows returning machines to be recognized and restore their")
	log.Println("previous permissions. This is public hardware info only — no")
	log.Println("passwords, API keys, or secrets are included.")
	log.Println("")
	log.Println("Privacy: If you run your own controller, only you see this data.")
	log.Println("If using public join.kontango.net, equivalent to a DNS lookup.")
	log.Println("")
	log.Println("═══════════════════════════════════════════════════════════════")
	log.Println("")

	log.Println("enrolling…")
	// Always pass empty string for method - server determines it based on data
	sseResult, err := enroll.SSEEnroll(url, "", session, roleID, secretID)
	if err != nil {
		return fmt.Errorf("enrollment: %w", err)
	}

	var result *enrollResult
	result = &enrollResult{
		ID:       sseResult.ID,
		Nickname: sseResult.Nickname,
		Identity: sseResult.Identity,
		Status:   sseResult.Status,
		Hosts:    sseResult.Config.Hosts,
		Tunnel:   sseResult.Config.Tunnel,
	}

	nick := result.Nickname
	if nick == "" {
		nick = result.ID[:8]
	}
	log.Printf("enrolled: %s (%s) [%s]", nick, result.ID, result.Status)

	return installAfterEnroll(url, result)
}

// installAfterEnroll saves identity, installs ziti/tunnel/agent/caddy.
// Shared by both TUI and non-interactive paths.
func installAfterEnroll(url string, result *enrollResult) error {
	log.SetFlags(0)

	platform, err := enroll.DetectPlatform()
	if err != nil {
		return fmt.Errorf("unsupported platform: %w", err)
	}

	nick := result.Nickname
	if nick == "" {
		nick = result.ID[:8]
	}

	if err := platform.EnsureDir(); err != nil {
		return fmt.Errorf("create directories: %w", err)
	}

	idPath := platform.IdentityPath()
	if err := os.WriteFile(idPath, result.Identity, 0600); err != nil {
		return fmt.Errorf("save identity: %w", err)
	}
	log.Printf("  identity: %s", idPath)

	record, _ := json.MarshalIndent(map[string]interface{}{
		"id": result.ID, "nickname": nick,
		"registered_at": time.Now().Unix(),
	}, "", "  ")
	machineFile := idPath[:strings.LastIndex(idPath, "/")+1] + "machine.json"
	os.WriteFile(machineFile, record, 0600)

	zitiVersion := "latest"
	if v, ok := result.Tunnel["version"].(string); ok && v != "" && v != "latest" {
		zitiVersion = v
	}
	log.Println("installing ziti…")
	if err := platform.InstallZiti(zitiVersion); err != nil {
		return fmt.Errorf("install ziti: %w", err)
	}
	log.Println("starting tunnel…")
	if err := platform.InstallService(idPath); err != nil {
		return fmt.Errorf("install tunnel service: %w", err)
	}
	if err := platform.StartService(); err != nil {
		return fmt.Errorf("start tunnel: %w", err)
	}

	log.Println("verifying tunnel…")
	if err := platform.WaitForTunnel(60 * time.Second); err != nil {
		log.Printf("  warning: %v (continuing)", err)
	} else {
		log.Println("  connected ✓")
	}

	log.Println("installing agent…")
	if err := platform.InstallAgent(url, idPath); err != nil {
		log.Printf("  warning: agent install failed: %v", err)
	} else {
		log.Println("  agent started ✓")
	}

	log.Println("installing caddy…")
	if err := platform.InstallCaddy(url, idPath); err != nil {
		log.Printf("  warning: caddy install failed: %v", err)
	} else {
		log.Println("  caddy started ✓")
	}

	log.Printf("\ndone.\n  nickname: %s\n  id:       %s\n  status:   %s\n", nick, result.ID, result.Status)
	return nil
}

// -- Types -------------------------------------------------------------------

type enrollResult struct {
	ID       string
	Nickname string
	Identity []byte
	Status   string
	Hosts    []string
	Tunnel   map[string]interface{}
}

