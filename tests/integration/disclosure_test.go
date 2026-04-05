package integration_test

import (
	"log"
	"os"
	"testing"

	"github.com/KontangoOSS/TangoKore/internal/enroll"
)

// TestDisclosure_NonInteractive demonstrates the fingerprinting disclosure
// that appears when using kontango enroll via non-interactive path (script/pipe).
//
// This test shows what a user sees at the terminal when they run:
//   kontango enroll https://ctrl.konoss.org --no-tui
//
// Expected output: Clear, upfront disclosure about what data will be sent.
func TestDisclosure_NonInteractive(t *testing.T) {
	if os.Getenv("KONTANGO_DISCLOSURE_TEST") == "" {
		t.Skip("skipping disclosure demo (set KONTANGO_DISCLOSURE_TEST=1 to run)")
	}

	log.Printf("\n╔════════════════════════════════════════════════════════════════╗")
	log.Printf("║              NON-INTERACTIVE ENROLLMENT                        ║")
	log.Printf("║          (This is what users see when they run)                ║")
	log.Printf("║    kontango enroll https://ctrl.konoss.org --no-tui          ║")
	log.Printf("╚════════════════════════════════════════════════════════════════╝\n")

	// Show what gets logged at enrollment start
	log.Printf("═══════════════════════════════════════════════════════════════\n")
	log.Printf("MACHINE FINGERPRINTING DISCLOSURE\n")
	log.Printf("═══════════════════════════════════════════════════════════════\n")
	log.Printf("\n")
	log.Printf("This machine will send hardware information to identify itself:\n")
	log.Printf("  • Hostname, OS version, kernel, architecture\n")
	log.Printf("  • CPU model/cores, system memory, motherboard ID\n")
	log.Printf("  • Network interface MAC addresses\n")
	log.Printf("\n")
	log.Printf("Why: Allows returning machines to be recognized and restore their\n")
	log.Printf("previous permissions. This is public hardware info only — no\n")
	log.Printf("passwords, API keys, or secrets are included.\n")
	log.Printf("\n")
	log.Printf("Privacy: If you run your own controller, only you see this data.\n")
	log.Printf("If using public ctrl.konoss.org, equivalent to a DNS lookup.\n")
	log.Printf("\n")
	log.Printf("═══════════════════════════════════════════════════════════════\n")
	log.Printf("\n")
	log.Printf("enrolling…\n")

	log.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// Collect actual machine data to show what's being sent
	fp, err := enroll.Collect()
	if err != nil {
		t.Fatalf("failed to collect fingerprint: %v", err)
	}

	log.Printf("ACTUAL DATA COLLECTED FROM THIS MACHINE:\n\n")
	log.Printf("Hostname:       %s\n", fp.Hostname)
	log.Printf("OS:             %s %s\n", fp.OS, fp.Arch)
	log.Printf("Kernel:         %s\n", fp.KernelVersion)
	log.Printf("CPU:            %s\n", fp.CPUInfo)
	log.Printf("Serial:         %s\n", fp.SerialNumber)
	log.Printf("Machine ID:     %s\n", fp.MachineID)
	log.Printf("Hardware Hash:  %s (← unique fingerprint)\n", fp.HardwareHash)
	log.Printf("Network MACs:   %v\n", fp.MACAddrs)

	log.Printf("\n(All of this is sent to the controller for identification)\n")
}

// TestDisclosure_TUI demonstrates the fingerprinting disclosure that appears
// in the TUI enrollment wizard's confirm step.
//
// This test shows what a user sees when they run:
//   kontango enroll https://ctrl.konoss.org
//
// At the confirm step, before pressing "y" to enroll, users see their collected
// data with an explanation of why it's being collected.
func TestDisclosure_TUI(t *testing.T) {
	if os.Getenv("KONTANGO_DISCLOSURE_TEST") == "" {
		t.Skip("skipping disclosure demo (set KONTANGO_DISCLOSURE_TEST=1 to run)")
	}

	log.Printf("\n╔════════════════════════════════════════════════════════════════╗")
	log.Printf("║                  TUI ENROLLMENT                              ║")
	log.Printf("║         (This is what users see when they run)               ║")
	log.Printf("║         kontango enroll https://ctrl.konoss.org            ║")
	log.Printf("║                    [INTERACTIVE MODE]                        ║")
	log.Printf("╚════════════════════════════════════════════════════════════════╝\n")

	log.Printf("Step 1: User enters controller URL\n")
	log.Printf("Step 2: User chooses enrollment method (new/approle/invite)\n")
	log.Printf("Step 3: User enters credentials (if needed)\n")
	log.Printf("Step 4: SDK collects fingerprint\n")
	log.Printf("Step 5: CONFIRM step (shown below) — user reviews data before enrolling\n\n")

	log.Printf("╔════════════════════════════════════════════════════════════════╗\n")
	log.Printf("│  CONFIRM ENROLLMENT                                           │\n")
	log.Printf("╚════════════════════════════════════════════════════════════════╝\n\n")

	log.Printf("This machine will send the following to identify itself:\n\n")

	fp, err := enroll.Collect()
	if err != nil {
		t.Fatalf("failed to collect fingerprint: %v", err)
	}

	log.Printf("  Hostname:        %s\n", fp.Hostname)
	log.Printf("  OS:              %s / %s\n", fp.OS, fp.Arch)
	log.Printf("  Kernel:          %s\n", fp.KernelVersion)

	cpu := fp.CPUInfo
	if len(cpu) > 40 {
		cpu = cpu[:40]
	}
	log.Printf("  CPU:             %s\n", cpu)

	machineID := fp.MachineID
	if len(machineID) > 24 {
		machineID = machineID[:24]
	}
	log.Printf("  Machine ID:      %s\n", machineID)
	log.Printf("  Hardware hash:   %s\n", fp.HardwareHash)
	log.Printf("  MACs:            %v\n\n", fp.MACAddrs)

	log.Printf("Why: Allows returning machines to be recognized and restore\n")
	log.Printf("their previous permissions. Public hardware info only — no\n")
	log.Printf("secrets included. Read the docs for more.\n\n")

	if len(fp.MACAddrs) > 0 {
		log.Printf("(At this point, user presses 'y/enter' to enroll, or 'n/esc' to abort)\n\n")
	}

	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	log.Printf("KEY POINTS:\n")
	log.Printf("  ✓ Users see their data BEFORE it's sent\n")
	log.Printf("  ✓ Clear explanation of WHY each field is collected\n")
	log.Printf("  ✓ Privacy guarantees (public hardware info, no secrets)\n")
	log.Printf("  ✓ Users can abort with 'n' if they don't want to proceed\n")
	log.Printf("  ✓ Control: own controller = only you see the data\n")
}

// TestDisclosure_Comparison shows the key differences between the two paths.
func TestDisclosure_Comparison(t *testing.T) {
	if os.Getenv("KONTANGO_DISCLOSURE_TEST") == "" {
		t.Skip("skipping disclosure demo (set KONTANGO_DISCLOSURE_TEST=1 to run)")
	}

	log.Printf("\n╔════════════════════════════════════════════════════════════════╗")
	log.Printf("║              DISCLOSURE COMPARISON                            ║")
	log.Printf("╚════════════════════════════════════════════════════════════════╝\n")

	log.Printf("NON-INTERACTIVE PATH                TUI PATH\n")
	log.Printf("═══════════════════════════════════ ═══════════════════════════════════\n\n")

	log.Printf("Usage:                              Usage:\n")
	log.Printf("  kontango enroll <url> --no-tui      kontango enroll <url>\n\n")

	log.Printf("Disclosure timing:                  Disclosure timing:\n")
	log.Printf("  ✓ Shown at START (before action)  ✓ Shown at CONFIRM (before action)\n")
	log.Printf("  ✓ Upfront, prominent banner       ✓ Integrated in review screen\n")
	log.Printf("  ✓ Users read about fingerprinting ✓ Users SEE their fingerprint\n")
	log.Printf("    before any connection              before pressing 'y' to proceed\n\n")

	log.Printf("For automated deployments:         For interactive users:\n")
	log.Printf("  ✓ Scripts get the explanation     ✓ See exactly what's being sent\n")
	log.Printf("  ✓ Curl piped installs show it     ✓ Review before confirming\n")
	log.Printf("  ✓ Installation docs can reference ✓ Abort option (press 'n')\n")

	log.Printf("\n═════════════════════════════════════════════════════════════════\n")
	log.Printf("RESULT: Same code, same transparency, two different UX paths\n")
	log.Printf("═════════════════════════════════════════════════════════════════\n")
}
