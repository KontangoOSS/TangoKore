# The Miranda Rights of TangoKore

**Minimum Required Information for SDK Installation**

Just like the Miranda Rights inform people of their fundamental rights, this document tells you exactly what information TangoKore needs to function—and nothing more unless you choose to provide it.

---

## The Bare Minimum (REQUIRED)

When you install the TangoKore SDK, we automatically collect exactly **three things**:

### 1. Operating System (OS)
- **What:** linux, darwin (macOS), or windows
- **Why:** We need to know what to install. Installing a Linux binary on macOS won't work.
- **How We Get It:** Built into the Go runtime—we ask the OS itself
- **User Control:** None needed. This is system information.

### 2. Architecture (CPU Type)
- **What:** amd64, arm64, armv7, etc.
- **Why:** We need to know the CPU architecture to download the right binary.
- **How We Get It:** Built into the Go runtime—we ask the OS itself
- **User Control:** None needed. This is system information.

### 3. Machine Identifier
- **What:** Either:
  - An ID you provide (your choice), OR
  - A randomly generated ID we issue to you (auto-generated if you don't provide one)
- **Why:** So we can recognize *this* machine as distinct from every other machine enrolling against the same controller.
- **How We Get It:** 
  - You give us one (optional), OR
  - We generate a cryptographically random 16-character string for you
- **User Control:** You can provide your own ID, or let us auto-issue one. Either way, it's yours.

---

## That's It. That's All We Need.

```
MINIMUM FINGERPRINT = OS + Architecture + Machine ID
```

Everything else is **optional enhancement**.

---

## Optional Enhancements (IF YOU CHOOSE)

If you want to opt-in to additional data collection for faster authentication or better machine recognition, you can provide:

### Enhanced Fingerprinting (Opt-In)
- **Hostname:** Your machine's name (helpful for identifying in logs/dashboards)
- **OS Version:** Which version of Linux/macOS/Windows (helps with compatibility checks)
- **CPU Model:** What processor you have (diagnostic purposes)
- **System Memory:** How much RAM you have (diagnostics, capacity planning)
- **Network MACs:** Your network interface addresses (machine recognition across network reconfigs)
- **Motherboard Serial:** Physical system identifier (hardware tracking)

### Credentials (Opt-In)
- **AppRole (Vault):** Pre-provisioned credentials for trusted enrollment (faster, skips quarantine)
- **Session Token:** One-time token from an installation script (trusted enrollment)
- **Custom Identifier:** A meaningful name instead of random ID

### Application Data (Opt-In, Future)
- Custom metadata (your application can define what helps identify it)
- Authorization hints (what services this machine should have access to)
- Geographic or organizational tags

---

## The Three Scenarios

### Scenario 1: Minimum Enrollment (Nothing Optional)
```bash
kontango enroll https://ctrl.example.com --no-tui
```

What gets sent:
```json
{
  "os": "linux",
  "arch": "amd64",
  "issued_id": "a1b2c3d4e5f6g7h8"  // We generated this for you
}
```

**Result:** Machine gets enrolled with automatic ID, lands in quarantine (read-only).

### Scenario 2: Enhanced Enrollment (With Opt-In Details)
```bash
kontango enroll https://ctrl.example.com \
  --hostname "my-server" \
  --role-id "$VAULT_ROLE_ID" \
  --secret-id "$VAULT_SECRET_ID"
```

What gets sent:
```json
{
  "os": "linux",
  "arch": "amd64",
  "issued_id": "my-server",  // You provided this
  "os_version": "Ubuntu 25.04",
  "hostname": "my-server",
  "cpu_info": "13th Gen Intel(R) Core(TM) i7-1360P",
  "memory_mb": 16384,
  "mac_addrs": ["08:00:27:00:00:00"],
  "serial_number": "...",
  "machine_id": "...",
  "hardware_hash": "56dd0f732bc62988",
  
  // Optional credentials (means "trust me, I'm pre-provisioned")
  "role_id": "$VAULT_ROLE_ID",
  "secret_id": "$VAULT_SECRET_ID"
}
```

**Result:** Server validates your credentials, machine gets full access (stage-3).

### Scenario 3: Returning Machine (Fingerprint Match)
```bash
kontango enroll https://ctrl.example.com --scan
```

SDK collects full fingerprint, server recognizes the hardware hash.

**Result:** Server restores your previous permissions and status automatically.

---

## Key Principles

### You Control What Gets Sent
- **Minimum:** Always sent (OS, arch, identifier)
- **Enhanced:** Only if you provide flags or opt-in
- **Credentials:** Only if you provide them

### You Control Your Identifier
- **Default:** We auto-issue a random 16-char ID
- **Custom:** You can provide your own hostname, name, or identifier
- **Your Choice:** Use the auto-issued one or replace it

### Zero Lock-In
- You can switch controllers anytime (export your identity, re-enroll elsewhere)
- You can run your own controller (all data stays private)
- You can delete your record from Kontango anytime

### Transparency
Before enrollment, you see:
- What minimum is being collected (always shown)
- What optional data you're providing (if any)
- Why it's needed
- Who will see it (Kontango or your private controller)

---

## The User's Rights

Just like Miranda Rights are non-negotiable, these are your rights:

1. **Right to Know:** See exactly what we're collecting before we collect it
2. **Right to Decline:** Provide just the minimum, no optional data
3. **Right to Control:** Choose your own identifier if you don't like the auto-issued one
4. **Right to Privacy:** Run your own controller and keep all data private
5. **Right to Delete:** Request deletion from Kontango anytime (email privacy@example.com)
6. **Right to Export:** Get your machine identity and certificates anytime
7. **Right to Migrate:** Move to a different controller without penalty

---

## Implementation

### Minimum Fingerprint Type
```go
type MinimalFingerprint struct {
  OS       string // "linux" | "darwin" | "windows"
  Arch     string // "amd64" | "arm64" | etc.
  IssuedID string // User-provided OR auto-generated
  Hostname string // Optional, usually present
}
```

### How It Works
1. **Installation:** SDK collects OS, arch, auto-issues ID
2. **Disclosure:** User sees what will be sent before enrollment
3. **Optional:** User can provide additional data or credentials (--role-id, --secret-id, etc.)
4. **Transmission:** Sends minimum + any opt-in data
5. **Server Decision:** Based on what was sent (minimum = new/quarantine, credentials = trusted, etc.)

### No User Input Required
- Zero prompts for minimum data
- Zero required fields
- If user doesn't provide an ID, one is automatically issued
- Works completely non-interactive

---

## This Is Your Machine

The machine identifier (whether auto-issued or user-provided) **is yours**. You own it. It's not a secret, not a password, not something we hold over you. It's just:
- A way for the controller to recognize "this is the same machine I saw before"
- A way for you to identify your machine in logs and dashboards
- Portable (you can use it with any controller, public or private)

---

## Compliance & Privacy

### GDPR
If you're in Europe:
- OS and architecture are not personally identifiable
- The issued ID is a pseudonym (not tied to you)
- You can request the ID be tied to you (GDPR right to rectification)
- You can request deletion anytime

### CCPA
If you're in California:
- You have the right to know what data is being collected
- You have the right to delete
- This document is that "right to know"

---

## FAQ

### Q: What if I don't provide an ID? Do I have one?
**A:** Yes. We auto-issue a random 16-character ID (e.g., `a1b2c3d4e5f6g7h8`). It's yours. You can use it, replace it, or request a new one.

### Q: Can I change my ID later?
**A:** Yes. Re-enroll with a new ID. The controller will treat it as a different machine or (if fingerprint matches) recognize it as the same machine under a new name.

### Q: Does TangoKore store my OS or architecture anywhere?
**A:** Yes, to know how to serve you the right binary and to match returning machines. This is not sensitive information.

### Q: What if I opt-out of everything optional?
**A:** You still get enrolled and on the mesh. You'll land in quarantine status (read-only), but you're authenticated and can receive configuration updates.

### Q: Can I provide a personally identifying ID (like my name)?
**A:** Yes. If you provide `--identifier "Alice's Laptop"`, that's what gets used. It's your choice whether to use a personal identifier or a random one.

### Q: What happens to my ID if I delete my enrollment record?
**A:** The ID is deleted with your record. If you re-enroll, you get a new random ID (unless you provide one).

### Q: Is this like the Miranda Rights because TangoKore is a law enforcement tool?
**A:** No! It's called "Miranda Rights" because both inform you of fundamental rights before something happens. Here, we're informing you of exactly what data we need and what's optional—before enrollment begins. Transparency, choice, and control.

---

## See Also

- **[PRIVACY.md](PRIVACY.md)** — Privacy controls and compliance details
- **[ULA.md](ULA.md)** — Full terms and conditions
- **example.com/privacy** — Kontango's public privacy policy

---

**The principle:** You provide the bare minimum to make the SDK work. Everything else is your choice.

*Last Updated: April 5, 2026*
