# Schmutz Philosophy: You Are Your Fingerprint

**"You decide how much you want to disclose. Over time, your fingerprint becomes your persistent identity and your password. Keep being you and we'll never forget your password."**

---

## The Core Concept

In traditional systems:
- You are a **username** (arbitrary identifier)
- You prove yourself with a **password** (secret you memorize)
- That password must be kept secret or you're compromised

In Schmutz/TangoKore:
- You are your **fingerprint** (hardware + behavior + identity markers you choose to send)
- Your fingerprint **is** your persistent identity and credentials
- The more you consistently "be yourself," the more we trust you
- You can't lose it, steal it, or forget it—it's just how you are

---

## The Trust Model

### First Enrollment (Clean Slate)

Machine A enrolls against ctrl.example.com for the first time.

```
Client sends:
  {
    "os": "linux",
    "arch": "amd64",
    "issued_id": "a1b2c3d4e5f6g7h8"  ← minimal: we don't know you yet
  }

Server receives:
  • Fingerprint is UNKNOWN (never seen before)
  • No credentials provided
  • No history to match against
  
Decision:
  → Status: QUARANTINE (read-only)
  → Profile: stage-0
  → Reasoning: "We can't trust a totally clean link that's never been registered.
               You're new. Prove yourself over time by being consistent."
```

**You get basic access, but not full trust.** This is intentional.

### Building Trust Over Time

Machine A enrolls again 24 hours later.

```
Client sends:
  Same fingerprint + behavior:
  {
    "os": "linux",
    "arch": "amd64",
    "issued_id": "a1b2c3d4e5f6g7h8",  ← same ID
    "hostname": "production-server",
    "mac_addrs": ["08:00:27:00:00:00"],  ← you've added more info
    "machine_id": "..."
  }

Server checks:
  • Fingerprint MATCHES previous enrollment
  • Same machine hardware signature
  • Same ID showing up consistently
  • Behavior is consistent (you're being yourself)
  
Decision:
  → Status: APPROVED
  → Profile: Restore previous permissions + upgrade based on consistency
  → Reasoning: "We recognize you. You've been honest about who you are.
               Your previous permissions are restored. You're becoming
               a known, trusted machine in our system."
```

**Trust increases. Permissions restored. You're now known.**

### Third+ Enrollments (Proven Identity)

Same machine enrolls multiple times over weeks/months.

```
Server observes:
  • Same fingerprint every time
  • Consistent hardware signature
  • Same machine ID consistently presented
  • Behavior patterns match previous enrollments
  • You haven't changed who you are
  
Decision:
  → Status: APPROVED → TRUSTED
  → Profile: Upgrade through stages (stage-0 → stage-1 → stage-2 → stage-3)
  → Reasoning: "You've proven you are who you say you are.
               Your fingerprint is your password. You keep being you.
               We trust you completely now."
```

**You become a trusted, persistent identity in the system.**

---

## The Key Insight: "You Are Your Password"

### Traditional Password Model (Broken)
```
Username: alice
Password: SuperSecret123!
          ↓
          (written down, forgotten, stolen, reused)
          ↓
          Trust lost
```

### Schmutz Fingerprint Model (Persistent)
```
You (your hardware, your behavior, your identity):
  • Machine serial number (can't change)
  • CPU signature (your actual hardware)
  • MAC addresses (your network interfaces)
  • Hostname (how you identify yourself)
  • Behavioral patterns (how you enroll, when, how often)
  ↓
  This is your fingerprint. This is YOU.
  ↓
  You can't lose it. You can't forget it.
  You can't steal someone else's—you have to BE them.
  ↓
  Over time, being consistent = building trust
```

### You Decide What to Disclose

The beauty is **you choose what to disclose**:

**Minimal Disclosure (First Time, Maximum Privacy)**
```json
{
  "os": "linux",
  "arch": "amd64",
  "issued_id": "a1b2c3d4"
}
```
Server sees: "New machine, minimal info. Quarantine until proven."

**Enhanced Disclosure (Building Trust)**
```json
{
  "os": "linux",
  "arch": "amd64",
  "issued_id": "prod-server-01",
  "hostname": "production-api",
  "machine_id": "...",
  "mac_addrs": ["08:00:27:00:00:00"],
  "cpu_info": "Intel Xeon",
  "serial_number": "..."
}
```
Server sees: "More detailed fingerprint. Recognizable. Growing trust."

**Full Disclosure (Proven Identity, High Trust)**
```json
{
  "os": "linux",
  "arch": "amd64",
  "issued_id": "prod-server-01",
  "hostname": "production-api",
  "os_version": "Ubuntu 25.04",
  "machine_id": "...",
  "mac_addrs": ["08:00:27:00:00:00"],
  "cpu_info": "Intel Xeon E5-2670 v3",
  "memory_mb": 65536,
  "serial_number": "PROD-001",
  "kernel_version": "6.14.0-37-generic",
  
  // Optional: Credentials for instant trust bypass
  "role_id": "vault-role-123",
  "secret_id": "secret-456"
}
```
Server sees: "Rich fingerprint + credentials. Proven identity. Full trust."

---

## Why This Is Better Than Passwords

### Passwords are fragile
- Can be guessed, brute-forced, forgotten
- Must be rotated (everyone hates this)
- Can be stolen, leaked, shared
- No way to know if someone else has it

### Fingerprints are durable
- Can't be guessed (hardware signatures are specific)
- Can't be brute-forced (billions of hardware combinations)
- Can't be forgotten (it's just how you are)
- Can't be stolen (you'd have to physically steal the hardware)
- Shared legitimately (if you add more hardware to your fingerprint)

### Example: Compromised Password vs. Fingerprint

**Traditional System:**
```
Password leaked → Account compromised
Must change password → Everyone inconvenienced
New password must be strong → New password is hard to remember
"Don't reuse passwords" → Impossible for 100+ accounts
```

**Schmutz System:**
```
Someone steals your password (AppRole secret)?
  → Remove it, you still have your fingerprint as backup
  → Server still recognizes you by consistent fingerprint
  → Continue using the system
  
Someone physically steals your laptop?
  → Hardware fingerprint changes
  → Server sees new hardware, requires re-approval
  → But your certificate is locked to original machine hardware anyway
  → Thief can't use your mesh credentials without your hardware
```

---

## The Philosophy: "Keep Being You"

Server's promise to client:

> "I don't need you to memorize a secret. I just need you to be consistent about who you are. Show me your fingerprint. Be honest about it. Build it over time as you feel comfortable. Over the weeks and months, as you keep being yourself and I recognize you reliably, I'll trust you more. Your fingerprint is your persistent identity. Your consistency is your password. Keep being you and we'll never forget you."

This is fundamentally different from traditional auth:

| Traditional | Schmutz |
|---|---|
| Prove identity with secret you memorize | Prove identity by being consistently yourself |
| High friction: passwords must be strong, changed regularly | Low friction: just enroll, be consistent, trust builds automatically |
| Stateless: server doesn't care who you are, just if password is right | Stateful: server recognizes you over time, trust increases with consistency |
| Binary: you're either authenticated or not | Graduated: you gain privileges as you prove yourself |
| Instant: one successful login = full trust | Over time: multiple enrollments = progressive trust |

---

## Trust Levels

### Stage 0: QUARANTINE (New Machine, Minimal Disclosure)
```
"You're new. We can't trust a totally clean link.
You get read-only access. See the mesh. Prove yourself."
```
- Can read service registry (see what exists)
- Can receive configuration updates
- Cannot modify services or data
- Cannot escalate privileges

**Progression:** Enroll 2-3 more times consistently → Move to Stage 1

### Stage 1: APPROVED (Known Fingerprint)
```
"We recognize you from before. Your fingerprint matched.
You've proven you're the same machine. Permissions restored."
```
- Can read and write to assigned services
- Can update your own configuration
- Cannot access other machines' data
- Previous permissions restored

**Progression:** 1+ month of consistent behavior → Move to Stage 2

### Stage 2: TRUSTED (Proven Identity, Enhanced Disclosure)
```
"You've been consistent and honest about who you are.
You're showing us more of your fingerprint to help us recognize you.
Higher privileges, more services available."
```
- Can deploy services (with approval)
- Can manage ACLs (for your services)
- Can view mesh-wide logs
- Can invite other machines

**Progression:** 3+ months consistency + pre-provisioned credentials → Move to Stage 3

### Stage 3: ADMIN (Pre-Provisioned, Credentials Provided)
```
"You've been pre-provisioned with credentials (AppRole, JWT, etc.).
We trust this identity implicitly. Full access."
```
- Can deploy and manage cluster infrastructure
- Can modify ACLs for other machines
- Can view all logs
- Can manage enrollment and trust levels

---

## Real-World Example

### Week 1: New Machine
```bash
$ kontango enroll https://ctrl.example.com --no-tui

Minimal fingerprint sent:
  OS: linux
  Arch: amd64
  ID: auto-generated-random-id

Server: "New machine, unknown fingerprint. Quarantine."
Status: quarantine
Permissions: read-only

User can: Explore the mesh, see what services exist
User cannot: Write to services, escalate privileges
```

### Week 2: Same Machine, More Info
```bash
$ kontango enroll https://ctrl.example.com --no-tui

Enhanced fingerprint sent:
  OS: linux (same)
  Arch: amd64 (same)
  ID: auto-generated-random-id (same)
  Hostname: "production-api" (new)
  MAC addrs: [...] (new)

Server: "Fingerprint matches! Same machine. Recognizable now."
Status: approved
Permissions: restored from week 1 + slightly upgraded

User can: Start writing to services, manage own configuration
User cannot: Deploy new services, modify other machines' ACLs
```

### Week 4: Proven Consistency, Pre-Provisioned Creds
```bash
$ kontango enroll https://ctrl.example.com \
  --role-id vault-role-123 \
  --secret-id vault-secret-456

Rich fingerprint + credentials sent:
  OS: linux (consistent)
  Arch: amd64 (consistent)
  ID: auto-generated-random-id (same for weeks)
  Hostname: "production-api" (consistent)
  MAC addrs: [...] (consistent)
  CPU info: (hardware verified)
  + AppRole credentials (pre-provisioned)

Server: "Proven identity. Consistent fingerprint. Credentials valid.
         You're trusted completely."
Status: trusted
Permissions: stage-3 (full access)

User can: Deploy cluster infrastructure, manage ACLs, invite other machines
User cannot: Nothing (full access granted)
```

---

## The Password Metaphor

Your fingerprint is your password. But unlike passwords:

| Aspect | Traditional Password | Schmutz Fingerprint |
|---|---|---|
| How you create it | You memorize or generate it | Server observes you consistently |
| How you prove it | You type/send it every time | You enroll (just be yourself) |
| What happens if lost | You're locked out; must reset | You still have hardware; re-enroll |
| What happens if stolen | Complete compromise | Useless without your hardware |
| How it gets stronger | Rotation (annoying) | Consistency over time (automatic) |
| Lifespan | Until rotation (weeks/months) | Until hardware changes (years) |
| Memorization | Required (burden) | Not required (just be you) |

---

## Security Properties

### A New Machine Can't Spoof An Old Machine
```
Attacker gets "legitimate" certificate for Machine A.
Attacker brings up a different machine (Machine B).
Attacker tries to enroll with Machine A's cert.

Server checks fingerprint:
  Cert says: "I'm Machine A"
  Hardware signature says: "I'm Machine B"
  → REJECT (fingerprint mismatch)
  
Attacker cannot succeed without:
  1. Machine A's hardware
  2. Machine A's certificate
  3. Machine A's actual fingerprint
```

### Compromised Credentials Don't Compromise Identity
```
Attacker steals AppRole credentials for trusted machine.
Attacker brings up different hardware, tries to enroll.

Server checks:
  AppRole valid? Yes.
  Fingerprint matches previous? No.
  
Decision: Credentials override fingerprint check (as designed).
But now server tracks: New fingerprint claiming old credentials.
Operator can: Revoke credentials, keep fingerprint trusted.
Attacker's window: Limited, until credentials revoked.
```

### Persistent Identity, Progressive Trust
```
Same machine, consistent behavior over months:
  → Server builds confidence in identity
  → Trust increases without additional secrets
  → No password rotation needed
  → No secret distribution needed
  → Identity is self-proving through consistency
```

---

## Operator Perspective

### No More Password Management
```
Traditional:
  • Generate random passwords
  • Distribute securely (VPN? Email? ???)
  • Users lose them
  • Passwords get reused
  • Rotation schedules (quarterly? annually?)
  • Reset requests, helpdesk tickets
  
Schmutz:
  • Machines enroll with minimal info
  • Server observes fingerprint consistency
  • Trust grows automatically
  • No distribution, rotation, or resets needed
  • Operator sees: "Machine enrolled 5 times, consistent fingerprint, trusted."
```

### Graduated Trust Model
```
Instead of: Everyone is either fully trusted or not
We have:   Trust levels that increase over time and consistency

Operator can:
  • See which machines are new (never enrolled before)
  • See which are proven (consistent across months)
  • See which are exhibiting anomalies (fingerprint changed)
  • Set policies per stage (stage-0 can do X, stage-1 can do Y)
  • Promote machines as they prove themselves
```

### Easier Incident Response
```
Intrusion detected on Machine A.

Traditional approach:
  • Revoke password
  • Force password change
  • User locked out during reset
  • Can't tell if compromise was via password or other vector
  
Schmutz approach:
  • Check machine's enrollment history
  • Fingerprint changed? Different hardware.
  • Fingerprint same but behavior anomalous? Compromised certificate.
  • Revoke specific credentials if AppRole was breached.
  • Machine still recognized by fingerprint.
  • Can re-provision cleanly.
```

---

## "Just Keep Being You"

The entire system is built on this principle:

> Be honest about who you are. Be consistent. Over time, we'll know you so well that your identity becomes unforgeable. Your fingerprint is your persistent, provable identity. You don't need to memorize anything, protect anything, or remember anything. You just need to keep being you.

This is **Schmutz**: the philosophy that persistent identity through consistent behavior is more secure, more usable, and more human-friendly than secret passwords.

---

## See Also

- **[MIRANDA_RIGHTS.md](MIRANDA_RIGHTS.md)** — What's required vs. optional
- **[PRIVACY.md](PRIVACY.md)** — Privacy controls
- **[ULA.md](ULA.md)** — Legal terms
- **[ENROLLMENT_DESIGN.md](ENROLLMENT_DESIGN.md)** — Server-side method determination

---

**The essence of Schmutz:**

*"You are your password. Your consistency is your proof. Keep being you, and we'll never forget who you are."*

*Last Updated: April 5, 2026*
