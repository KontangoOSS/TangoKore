# TangoKore Documentation Index

**Complete guide to understanding the SDK, its philosophy, and how to use it.**

---

## 🎯 Start Here

### For First-Time Users
1. **[README.md](README.md)** — What is TangoKore? Overview and quick start
2. **[SCHMUTZ_PHILOSOPHY.md](SCHMUTZ_PHILOSOPHY.md)** — Understand the core concept: "You are your fingerprint"
3. **[MIRANDA_RIGHTS.md](MIRANDA_RIGHTS.md)** — What's the minimum required? What's optional?

### For Operators/DevOps
1. **[ENROLLMENT_DESIGN.md](ENROLLMENT_DESIGN.md)** — How enrollment works server-side
2. **[SCHMUTZ_PHILOSOPHY.md](SCHMUTZ_PHILOSOPHY.md)** — Trust levels and machine lifecycle
3. **[SDK_COMPLETE.md](SDK_COMPLETE.md)** — Features, architecture, deployment

### For Compliance/Legal
1. **[ULA.md](ULA.md)** — Full legal terms and data handling
2. **[PRIVACY.md](PRIVACY.md)** — Privacy rights, data subject requests, compliance checklists
3. **[MIRANDA_RIGHTS.md](MIRANDA_RIGHTS.md)** — What data we need and why

---

## 📚 Documentation by Topic

### Philosophy & Core Concepts
- **[SCHMUTZ_PHILOSOPHY.md](SCHMUTZ_PHILOSOPHY.md)** — "You are your fingerprint"
  - Why fingerprinting is better than passwords
  - Trust levels and graduated authentication
  - How consistency becomes identity
  - Real-world examples (week 1, 2, 4)

### User & Installation
- **[MIRANDA_RIGHTS.md](MIRANDA_RIGHTS.md)** — Minimum data requirements
  - Required: OS, architecture, machine ID (auto-issued if needed)
  - Optional: hostname, credentials, custom identifier
  - Zero user input required for basic enrollment
  - User control over data disclosure

- **[README.md](README.md)** — SDK overview
  - What is TangoKore?
  - Quick start (one command)
  - Components and architecture
  - How enrollment works

### Privacy & Compliance
- **[PRIVACY.md](PRIVACY.md)** — Privacy controls
  - What data is collected and why
  - Who sees it (public vs. private controller)
  - Data subject rights (GDPR/CCPA)
  - Deletion process
  - Compliance checklists

- **[ULA.md](ULA.md)** — Universal License Agreement
  - Full data handling terms
  - Where data goes
  - Transparency & disclosure
  - Legal basis for processing
  - Limitations and liabilities
  - Contact information

### Production Architecture
- **[architecture/production-topology.md](architecture/production-topology.md)** — Cluster layout and deployment
  - 3-node controller cluster (DigitalOcean)
  - 4-node Proxmox edge router mesh
  - Systemd services, ports, binary locations
  - Bootstrap sequence

- **[architecture/service-mesh.md](architecture/service-mesh.md)** — Ziti service mesh design
  - Service registration pattern (host.v2 + intercept.v1)
  - DNS resolution flow (.tango domains)
  - Router modes (host vs tproxy)
  - Policy matrix and service attributes
  - Smartrouting and naming conventions

- **[architecture/caddy-routing.md](architecture/caddy-routing.md)** — Caddy L4 gateway
  - TLS termination and certificate management
  - Public route structure
  - L4 SNI passthrough design
  - Reusable snippets

- **[architecture/pki.md](architecture/pki.md)** — Certificate architecture
  - Certificate hierarchy (Root → Intermediate → Server/Router/Identity)
  - File locations and SANs
  - Trust model between components
  - Key operations (generate, enroll, rotate)

### Enrollment & Technical
- **[ENROLLMENT_DESIGN.md](ENROLLMENT_DESIGN.md)** — Enrollment architecture
  - How the server determines method (new/scan/trusted)
  - Event streaming (verify → decision → identity)
  - Three enrollment paths

- **[E2E_TEST_EXAMPLES.md](E2E_TEST_EXAMPLES.md)** — Real test output
  - What each test shows
  - Actual enrollment logs
  - Server responses
  - Configuration delivery


- **[SDK_COMPLETE.md](SDK_COMPLETE.md)** — Deployment guide
  - Production-ready checklist
  - Architecture overview
  - Test coverage
  - Verification steps

---

## 🔑 Key Concepts

### Three Things You Need to Know

1. **Fingerprinting as Identity**
   - Your machine's fingerprint (OS, hardware, behavior) becomes your identity
   - You don't memorize passwords—you just keep being yourself
   - Server recognizes you by consistent behavior
   - See: [SCHMUTZ_PHILOSOPHY.md](SCHMUTZ_PHILOSOPHY.md)

2. **Minimum Required Data**
   - OS (to install right binary)
   - CPU architecture (to download right binary)
   - Machine identifier (auto-issued if you don't provide one)
   - Everything else is optional
   - See: [MIRANDA_RIGHTS.md](MIRANDA_RIGHTS.md)

3. **Server Decides Trust**
   - Client sends data (minimal or enhanced)
   - Server determines: new/returning/trusted
   - Client never claims its own trust level
   - Server's decision is based on fingerprint + credentials
   - See: [ENROLLMENT_DESIGN.md](ENROLLMENT_DESIGN.md)

---

## 🛡️ Privacy & Security

### Your Rights
- ✓ Right to know what's collected (see disclosure before enrollment)
- ✓ Right to minimize (send only required data)
- ✓ Right to control (opt-in for enhancements)
- ✓ Right to privacy (run your own controller)
- ✓ Right to delete (request deletion anytime)
- ✓ Right to portability (export your identity)
- ✓ Right to migrate (switch controllers anytime)

See: [PRIVACY.md](PRIVACY.md) and [ULA.md](ULA.md)

### Security Model
- Fingerprints are durable (hardware-based, can't be guessed)
- Credentials are optional (fingerprint alone is enough)
- Trust grows over time (consistency is proof)
- No secrets to rotate (your identity isn't a password)
- Hardware-bound (machine can't be spoofed)

See: [SCHMUTZ_PHILOSOPHY.md](SCHMUTZ_PHILOSOPHY.md)

---

## 📋 By Use Case

### "I just want to install the SDK"
→ [README.md](README.md) Quick Start section

### "What data will you collect from my machine?"
→ [MIRANDA_RIGHTS.md](MIRANDA_RIGHTS.md) — See "The Bare Minimum"

### "Can I run my own controller and keep all data private?"
→ [PRIVACY.md](PRIVACY.md) — See "Running Your Own Controller"

### "What happens if I don't provide a machine ID?"
→ [MIRANDA_RIGHTS.md](MIRANDA_RIGHTS.md) — Auto-issued random ID

### "How does the server know if I'm trustworthy?"
→ [SCHMUTZ_PHILOSOPHY.md](SCHMUTZ_PHILOSOPHY.md) — Trust Levels section

### "What are the legal terms?"
→ [ULA.md](ULA.md)

### "Is this GDPR/CCPA compliant?"
→ [ULA.md](ULA.md) Section 17-18 (legal basis), [PRIVACY.md](PRIVACY.md) (data subject rights)

### "How do I delete my enrollment record?"
→ [PRIVACY.md](PRIVACY.md) — Deletion Process section

### "Can I migrate from public to private controller?"
→ [PRIVACY.md](PRIVACY.md) — Migration section

### "How do I understand the enrollment flow?"
→ [ENROLLMENT_DESIGN.md](ENROLLMENT_DESIGN.md)

### "What tests have been run?"
→ [SDK_COMPLETE.md](SDK_COMPLETE.md) — Test Coverage section, or [E2E_TEST_EXAMPLES.md](E2E_TEST_EXAMPLES.md)

---

## 🔗 Documentation Map

```
README.md (entry point)
  ├─ Quick Start
  ├─ Architecture Overview
  └─ Privacy & Compliance (→ links below)
      ├─ SCHMUTZ_PHILOSOPHY.md
      │   ├─ "You are your password"
      │   ├─ Trust levels (stage 0-3)
      │   ├─ Real-world examples
      │   └─ Security properties
      │
      ├─ MIRANDA_RIGHTS.md
      │   ├─ Required minimum
      │   ├─ Optional enhancements
      │   ├─ Three scenarios
      │   └─ User rights
      │
      ├─ PRIVACY.md
      │   ├─ Data minimization
      │   ├─ Public vs. private
      │   ├─ Data subject rights
      │   ├─ Migration
      │   └─ Compliance checklists
      │
      └─ ULA.md
          ├─ Full legal terms
          ├─ Data handling
          ├─ Liability
          └─ Contact info

Production Architecture:
  ├─ architecture/production-topology.md
  │   ├─ Cluster layout (DO + Proxmox)
  │   ├─ Services per node
  │   ├─ Firewall rules
  │   └─ Bootstrap sequence
  ├─ architecture/service-mesh.md
  │   ├─ .tango DNS resolution
  │   ├─ host.v2 config pattern
  │   ├─ Policy matrix
  │   └─ Naming conventions
  ├─ architecture/caddy-routing.md
  │   ├─ Public routes
  │   ├─ L4 SNI design
  │   └─ Certificate management
  └─ architecture/pki.md
      ├─ Certificate hierarchy
      ├─ Trust model
      └─ Key operations

Technical Documentation:
  ├─ ENROLLMENT_DESIGN.md
  │   ├─ Server-side method determination
  │   ├─ Three enrollment paths
  │   └─ Event streaming
  │
  ├─ SDK_COMPLETE.md
  │   ├─ Feature inventory
  │   ├─ Test coverage
  │   ├─ Architecture
  │   └─ Deployment guide
  │
  │   └─ Feature details
  │
  └─ E2E_TEST_EXAMPLES.md
      └─ Real test output
```

---

## 📖 Reading Paths

### Path 1: "I'm skeptical about fingerprinting"
1. [SCHMUTZ_PHILOSOPHY.md](SCHMUTZ_PHILOSOPHY.md) — Understand why it's better
2. [MIRANDA_RIGHTS.md](MIRANDA_RIGHTS.md) — See what's actually collected
3. [PRIVACY.md](PRIVACY.md) — Understand your control and rights

### Path 2: "I need to understand deployment"
1. [architecture/production-topology.md](architecture/production-topology.md) — How it's deployed
2. [architecture/service-mesh.md](architecture/service-mesh.md) — How services communicate
3. [architecture/caddy-routing.md](architecture/caddy-routing.md) — How public access works
4. [architecture/pki.md](architecture/pki.md) — How certificates work
5. [ENROLLMENT_DESIGN.md](ENROLLMENT_DESIGN.md) — How enrollment works

### Path 3: "I'm a compliance/legal officer"
1. [ULA.md](ULA.md) — Full terms
2. [PRIVACY.md](PRIVACY.md) — Data subject rights
3. [SCHMUTZ_PHILOSOPHY.md](SCHMUTZ_PHILOSOPHY.md) — Understand the model
4. [MIRANDA_RIGHTS.md](MIRANDA_RIGHTS.md) — Understand data minimization

### Path 4: "I just want to get started"
1. [README.md](README.md) — Quick Start
2. [MIRANDA_RIGHTS.md](MIRANDA_RIGHTS.md) — What gets collected (5 min read)
3. Run: `kontango enroll https://ctrl.example.com`

---

## 🎓 Learning Resources

### Understand the Philosophy
Start here to get the "why":
- [SCHMUTZ_PHILOSOPHY.md](SCHMUTZ_PHILOSOPHY.md) — 15-20 minute read
- Core insight: "You are your fingerprint" / "Keep being you"

### Understand the Data
What gets collected and why:
- [MIRANDA_RIGHTS.md](MIRANDA_RIGHTS.md) — 10-15 minute read
- Minimum required, optional enhancements, your control

### Understand Your Rights
Privacy and compliance:
- [PRIVACY.md](PRIVACY.md) — 15 minute read
- Data subject rights, compliance checklists, migration paths

### Understand the Legal Terms
Full terms and conditions:
- [ULA.md](ULA.md) — 20-30 minute read
- Data processing, liability, compliance frameworks

### Understand the Implementation
Technical architecture:
- [ENROLLMENT_DESIGN.md](ENROLLMENT_DESIGN.md) — 10 minute read
- [SDK_COMPLETE.md](SDK_COMPLETE.md) — 20 minute read

---

## ❓ FAQ

**Q: Do I have to provide any data to enroll?**  
A: No. The minimum (OS, arch) is collected automatically. Machine ID is auto-issued if you don't provide one. See [MIRANDA_RIGHTS.md](MIRANDA_RIGHTS.md).

**Q: Can I run a private controller?**  
A: Yes. Then fingerprints never leave your infrastructure. See [PRIVACY.md](PRIVACY.md).

**Q: What if I don't trust fingerprinting?**  
A: Read [SCHMUTZ_PHILOSOPHY.md](SCHMUTZ_PHILOSOPHY.md) to understand why it's more secure than passwords.

**Q: Is this GDPR compliant?**  
A: Yes. See [ULA.md](ULA.md) Section 17-18 and [PRIVACY.md](PRIVACY.md).

**Q: Can I delete my data?**  
A: Yes. Email privacy@example.com. See [PRIVACY.md](PRIVACY.md).

**Q: Can I migrate to a different controller?**  
A: Yes. Export your identity and re-enroll. See [PRIVACY.md](PRIVACY.md).

**Q: What makes this better than passwords?**  
A: Fingerprints can't be guessed, stolen, rotated, or forgotten. Trust grows automatically through consistency. See [SCHMUTZ_PHILOSOPHY.md](SCHMUTZ_PHILOSOPHY.md).

---

## 📞 Support & Contact

**Privacy Questions:** privacy@example.com  
**Data Deletion Requests:** privacy@example.com (title: "DATA DELETION REQUEST")  
**Legal/Compliance Requests:** legal@example.com  
**Community/Technical:** github.com/KontangoOSS/TangoKore/discussions

---

## 📝 Document Versions

| Document | Size | Updated |
|----------|------|---------|
| README.md | 20K | Apr 5, 2026 |
| SCHMUTZ_PHILOSOPHY.md | 12K | Apr 5, 2026 |
| MIRANDA_RIGHTS.md | 9.2K | Apr 5, 2026 |
| PRIVACY.md | 8.9K | Apr 5, 2026 |
| ULA.md | 14K | Apr 5, 2026 |
| SDK_COMPLETE.md | 11K | Apr 5, 2026 |
| ENROLLMENT_DESIGN.md | 4.4K | Apr 5, 2026 |
| E2E_TEST_EXAMPLES.md | 14K | Apr 5, 2026 |

**Total Documentation:** ~100K across 10 files

---

**Start with [README.md](README.md), then choose your path above.** 

All documents are in the root of the repository and linked from the README.

*Last Updated: April 5, 2026*
