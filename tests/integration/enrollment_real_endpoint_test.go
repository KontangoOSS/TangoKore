package integration_test

import (
	"encoding/json"
	"log"
	"os"
	"testing"

	"github.com/KontangoOSS/TangoKore/internal/enroll"
)

// TestE2E_RealEndpoint_Announcement tests enrollment against the real public endpoint.
// This test demonstrates the actual flow machines will use.
//
// IMPORTANT: This test sends real data to the public endpoint!
// Set KONTANGO_REAL_ENDPOINT=1 to enable (disabled by default for safety).
func TestE2E_RealEndpoint_Announcement(t *testing.T) {
	if os.Getenv("KONTANGO_REAL_ENDPOINT") == "" {
		t.Skip("skipping real endpoint test (set KONTANGO_REAL_ENDPOINT=1 to run)")
	}

	log.Printf("\n╔════════════════════════════════════════════════════════════════╗")
	log.Printf("║                 REAL ENDPOINT TEST                             ║")
	log.Printf("║          Enrolling against ctrl.konoss.org                   ║")
	log.Printf("╚════════════════════════════════════════════════════════════════╝\n")

	controllerURL := "https://ctrl.konoss.org:1280"

	log.Printf("═══════════════════════════════════════════════════════════════════")
	log.Printf("MACHINE ANNOUNCEMENT PAYLOAD")
	log.Printf("═══════════════════════════════════════════════════════════════════\n")

	log.Printf("The SDK collects the following information about this machine:\n")

	// Collect machine data (same as the SDK does)
	osData := enroll.ProbeOS()
	hwData := enroll.ProbeHardware()
	netData := enroll.ProbeNetwork()
	sysData := enroll.ProbeSystem()

	// Build the payload that will be sent
	payload := map[string]interface{}{}

	log.Printf("1. OPERATING SYSTEM INFORMATION:")
	log.Printf("   Why: Server needs to know what OS this is for compatibility checks")
	for k, v := range osData {
		if k != "type" {
			payload[k] = v
			log.Printf("   • %s: %v", k, v)
		}
	}

	log.Printf("\n2. HARDWARE INFORMATION:")
	log.Printf("   Why: Creates a unique fingerprint for machine identification")
	log.Printf("   (Allows returning machines to be recognized)")
	for k, v := range hwData {
		if k != "type" {
			payload[k] = v
			switch k {
			case "hardware_hash":
				log.Printf("   • %s: %v (← UNIQUE FINGERPRINT)", k, v)
			case "machine_id":
				log.Printf("   • %s: %v (← System machine ID)", k, v)
			default:
				log.Printf("   • %s: %v", k, v)
			}
		}
	}

	log.Printf("\n3. NETWORK INFORMATION:")
	log.Printf("   Why: Server can identify network segment and location")
	for k, v := range netData {
		if k != "type" {
			payload[k] = v
			log.Printf("   • %s: %v", k, v)
		}
	}

	log.Printf("\n4. SYSTEM INFORMATION:")
	log.Printf("   Why: Additional context for verification and security checks")
	for k, v := range sysData {
		if k != "type" {
			payload[k] = v
			log.Printf("   • %s: %v", k, v)
		}
	}

	log.Printf("\n═══════════════════════════════════════════════════════════════════")
	log.Printf("FINGERPRINTING EXPLAINED")
	log.Printf("═══════════════════════════════════════════════════════════════════\n")

	log.Printf("What is being fingerprinted:")
	log.Printf("  • CPU model and count")
	log.Printf("  • System motherboard/serial numbers")
	log.Printf("  • MAC addresses of network interfaces")
	log.Printf("  • Total system memory")
	log.Printf("  • Operating system version")
	log.Printf("  • System hostname")

	log.Printf("\nWhy it works:")
	log.Printf("  1. If a machine enrolls as NEW:")
	log.Printf("     → Server stores this fingerprint")
	log.Printf("     → Server gives it QUARANTINE status (read-only)")
	log.Printf("")
	log.Printf("  2. If the same machine re-enrolls later:")
	log.Printf("     → Server recognizes the fingerprint")
	log.Printf("     → Server RESTORES its previous status and ACL")
	log.Printf("     → Machine automatically gets its previous permissions")
	log.Printf("")
	log.Printf("  3. If credentials (AppRole) are provided:")
	log.Printf("     → Server validates the credentials")
	log.Printf("     → Server TRUSTS the machine (full access)")
	log.Printf("     → Fingerprint is secondary to credentials")

	log.Printf("\nWhy it's safe:")
	log.Printf("  • Fingerprinting data is NOT sensitive")
	log.Printf("  • It's just hardware info that's public on the machine")
	log.Printf("  • No passwords, keys, or secrets in fingerprint")
	log.Printf("  • Same as what `uname -a` or `lspci` would show")
	log.Printf("")
	log.Printf("  • If you run your OWN controller:")
	log.Printf("    → You control who sees this data")
	log.Printf("    → No exposure to public endpoints")
	log.Printf("    → You decide who to trust")

	log.Printf("\n═══════════════════════════════════════════════════════════════════")
	log.Printf("SENDING ANNOUNCEMENT")
	log.Printf("═══════════════════════════════════════════════════════════════════\n")

	log.Printf("Connecting to: %s\n", controllerURL)

	// Send the enrollment
	result, err := enroll.SSEEnroll(controllerURL, "", "", "", "")

	if err != nil {
		log.Printf("\n❌ ENROLLMENT FAILED: %v\n", err)
		log.Printf("This is expected if:")
		log.Printf("  • Network cannot reach the controller")
		log.Printf("  • Controller is not running")
		log.Printf("  • DNS resolution failed\n")
		log.Printf("This test is for demonstrating the flow.\n")
		t.Skip("Could not reach real endpoint")
	}

	log.Printf("\n╔════════════════════════════════════════════════════════════════╗")
	log.Printf("║              ENROLLMENT SUCCESSFUL                             ║")
	log.Printf("╚════════════════════════════════════════════════════════════════╝\n")

	log.Printf("Server Response:")
	log.Printf("  • Machine ID: %s", result.ID)
	log.Printf("  • Nickname: %s", result.Nickname)
	log.Printf("  • Status: %s", result.Status)
	log.Printf("  • Hosts: %v", result.Config.Hosts)
	log.Printf("  • Tunnel: %v\n", result.Config.Tunnel)

	log.Printf("═══════════════════════════════════════════════════════════════════")
	log.Printf("WHAT HAPPENED")
	log.Printf("═══════════════════════════════════════════════════════════════════\n")

	log.Printf("1. Machine collected its fingerprint (hardware info)")
	log.Printf("2. Machine sent announcement to server:")
	log.Printf("   POST /api/enroll/stream")
	log.Printf("   Body: %d bytes of machine data", len(mustMarshal(payload)))
	log.Printf("")
	log.Printf("3. Server received the announcement:")
	log.Printf("   • Checked if fingerprint was known (new machine)")
	log.Printf("   • Ran verification checks (OS valid, not banned, etc)")
	log.Printf("   • Assigned status: %s", result.Status)
	log.Printf("   • Generated certificate and config")
	log.Printf("")
	log.Printf("4. Server sent back:")
	log.Printf("   • Identity certificate (PKCS12)")
	log.Printf("   • Ziti configuration")
	log.Printf("   • ACL permissions")
	log.Printf("")
	log.Printf("5. Machine received identity:")
	log.Printf("   • Can now authenticate to Ziti mesh")
	log.Printf("   • Can connect to allowed services")
	log.Printf("   • Can report telemetry via NATS")

	log.Printf("\n═══════════════════════════════════════════════════════════════════")
	log.Printf("TRANSPARENCY SUMMARY")
	log.Printf("═══════════════════════════════════════════════════════════════════\n")

	log.Printf("What the SDK is fingerprinting:")
	log.Printf("  ✓ Exactly what you see above")
	log.Printf("  ✓ Hardware info (CPU, memory, motherboard)")
	log.Printf("  ✓ Network info (MAC addresses)")
	log.Printf("  ✓ System info (OS version, hostname)")
	log.Printf("  ✓ All logged so you know what's sent")
	log.Printf("")

	log.Printf("Why we fingerprint:")
	log.Printf("  ✓ Identify machines uniquely")
	log.Printf("  ✓ Recognize returning machines")
	log.Printf("  ✓ Enable fingerprint-based authentication")
	log.Printf("  ✓ Prevent spoofing (can't claim to be a trusted machine)")
	log.Printf("")

	log.Printf("How it identifies machines:")
	log.Printf("  ✓ First enrollment: fingerprint + status stored")
	log.Printf("  ✓ Later enrollments: fingerprint matched against history")
	log.Printf("  ✓ Trusted path: credentials override fingerprint")
	log.Printf("  ✓ All logged transparently")
	log.Printf("")

	log.Printf("Your privacy:")
	log.Printf("  ✓ Running private controller? Only YOU see fingerprints")
	log.Printf("  ✓ Using public ctrl.konoss.org? Same as DNS lookup")
	log.Printf("  ✓ No passwords, keys, or secrets in fingerprint")
	log.Printf("  ✓ All within your control")
	log.Printf("")

	log.Printf("═══════════════════════════════════════════════════════════════════\n")

	// Verify we got an identity
	if result.ID == "" {
		t.Error("expected machine ID in response")
	}
	if result.Status == "" {
		t.Error("expected status in response")
	}
}

func mustMarshal(data interface{}) []byte {
	b, _ := json.Marshal(data)
	return b
}
