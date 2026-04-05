# Universal License Agreement (ULA)

**Kontango Machine SDK — Zero-Trust Mesh Enrollment**

Last Updated: April 5, 2026

---

## 1. Agreement Overview

This Universal License Agreement covers the use of the Kontango Machine SDK (TangoKore), including:
- The `kontango` CLI enrollment tool
- Machine fingerprinting and identity collection
- Enrollment against public and private controllers
- Data transmission to Kontango infrastructure or your own

**Key Principle:** You control your data. If you run your own controller, only you see the fingerprints.

---

## 2. What Data We Collect

When you enroll a machine, the SDK collects and transmits:

### Hardware Information (Fingerprinting)
- **Hostname** — machine name
- **CPU Model and Count** — processor information
- **System Memory** — total RAM
- **Motherboard Serial** — system identifier
- **MAC Addresses** — network interface identifiers
- **OS Version & Kernel** — operating system details
- **Architecture** — processor type (amd64, arm64, etc.)

### Purpose
The hardware fingerprint enables:
1. **Machine Identification** — recognize this machine uniquely
2. **Returning Machine Detection** — know if a machine has enrolled before
3. **ACL Restoration** — restore previous permissions automatically

### What We Don't Collect
- ❌ Passwords or private keys
- ❌ API credentials or tokens
- ❌ Application data or user files
- ❌ Network traffic or logs (unless explicitly configured elsewhere)
- ❌ Browser history or personal information
- ❌ Anything in `/home`, `/root`, or user directories

---

## 3. Where Data Goes

### Option A: Public Enrollment (ctrl.example.com)

If you enroll against the public Kontango controller:
- **Data Recipient:** Kontango Infrastructure (Kontango OSS)
- **Data Retention:** Machine fingerprints stored in Kontango database
- **Access:** Kontango operations team (for cluster management)
- **Use Case:** Shared infrastructure, zero-trust enrollment at scale

### Option B: Private Controller (Your Infrastructure)

If you run your own Kontango controller:
- **Data Recipient:** Your infrastructure only
- **Data Retention:** Your control (your database, your policies)
- **Access:** Only your administrators
- **Use Case:** Compliance, air-gapped networks, complete control

**Recommended for sensitive deployments**

---

## 4. Transparency & Disclosure

### Before Enrollment
The SDK shows a clear disclosure **before** connecting to any controller:

```
═══════════════════════════════════════════════════════════════
MACHINE FINGERPRINTING DISCLOSURE
═══════════════════════════════════════════════════════════════

This machine will send hardware information to identify itself:
  • Hostname, OS version, kernel, architecture
  • CPU model/cores, system memory, motherboard ID
  • Network interface MAC addresses

Why: Allows returning machines to be recognized and restore their
     previous permissions. This is public hardware info only — no
     passwords, API keys, or secrets are included.

Privacy: If you run your own controller, only you see this data.
         If using public ctrl.example.com, equivalent to a DNS lookup.
```

### Interactive (TUI) Mode
Users see collected data in the **Confirm** step and can review before pressing 'y' to proceed. Option to abort with 'n'.

### Non-Interactive Mode
Disclosure logged prominently before any connection.

---

## 5. Your Data, Your Control

### Self-Hosted Kontango Controller

You have the right to:
1. **Run your own controller** — use the kontango-controller package
2. **Store fingerprints locally** — in your own database
3. **Set your own retention policies** — delete fingerprints on any schedule
4. **Restrict access** — air-gap from public networks
5. **Audit all enrollment** — view all machines and their status
6. **Define permissions** — control which machines can access which services

### No Lock-In
- Machine identities are portable (PKCS12 certificates)
- Ziti configuration is exportable
- You can migrate to a different controller at any time

---

## 6. Policy Links

### Privacy Policy
For public enrollment (ctrl.example.com):
- [example.com/privacy](https://example.com/privacy)
- Details about data processing, retention, third parties

### Terms of Service
For public controller usage:
- [example.com/terms](https://example.com/terms)
- SLAs, limitations, acceptable use

### Self-Hosted Documentation
To run your own controller:
- [docs.example.com/self-hosted](https://docs.example.com/self-hosted)
- Installation, configuration, compliance options

---

## 7. Legal Basis (Public Enrollment)

### Data Processing
By enrolling against ctrl.example.com, you consent to:
- Collection of hardware fingerprint data
- Storage for machine identification and ACL management
- Processing by Kontango operations team
- Transmission over encrypted channels (TLS 1.3)

### Data Subject Rights (GDPR)
If you are in a GDPR jurisdiction:
- Right to access: request your machine's fingerprint data
- Right to deletion: request deletion of your machine record
- Right to portability: export your machine identity certificate

Contact: **privacy@example.com**

### Data Subject Rights (CCPA)
If you are in California:
- Right to know: what data Kontango collects about your machine
- Right to delete: request deletion of machine records
- Right to opt-out: use private controller instead

Contact: **privacy@example.com**

---

## 8. Security

### Encryption in Transit
All enrollment data is sent over **TLS 1.3** connections:
- Public endpoint: `https://ctrl.example.com:443`
- Hostname verification enabled
- Certificate validation required

### Encryption at Rest (Public)
Machine fingerprints in Kontango database:
- AES-256 encryption
- Separate keys for different data categories
- Regular key rotation

### Encryption at Rest (Self-Hosted)
Your controller, your encryption:
- Use your own database encryption policies
- TDE (Transparent Data Encryption) recommended
- Access control via your infrastructure

---

## 9. Compliance & Certifications

### Kontango-Hosted (Public)
- **ISO 27001** — Information Security Management (in progress)
- **SOC 2 Type II** — Audit in progress
- **GDPR Compliant** — Data processing agreements available
- **HIPAA-Eligible** — Available for covered entities (BAA required)

### Self-Hosted
- Compliance is your responsibility
- Kontango provides the tools and documentation
- Audit logs available locally

---

## 10. What Happens After Enrollment

### Configuration Delivery
After enrollment, the server sends:
- **Ziti identity** (certificate for overlay network)
- **ACL permissions** (which services this machine can access)
- **Profile level** (quarantine / approved / trusted)

### Subsequent Calls
The machine can then:
- Connect to the Ziti mesh with its certificate
- Report telemetry via NATS (optional, configurable)
- Receive updates via NATS channels
- Restore previous ACLs on re-enrollment

### No Additional Data Collection
- After enrollment, no additional fingerprinting occurs
- Only service telemetry (heartbeats, logs) are sent
- All subsequent communication uses Ziti overlay (encrypted)

---

## 11. Third Parties

### Public Infrastructure (ctrl.example.com)
- **Cloud Provider:** DigitalOcean
- **Database:** PostgreSQL (managed)
- **Message Bus:** NATS (managed)
- **DNS:** Kontango-controlled

**Data Subprocessors:**
- DigitalOcean (infrastructure provider) — [DO Privacy Policy](https://www.digitalocean.com/legal/privacy/)
- Sentry (optional error reporting) — [Sentry Privacy](https://sentry.io/privacy/)

### Self-Hosted
You choose all infrastructure providers.

---

## 12. Data Retention

### Public Enrollment
**Default Policy:**
- Active machines: fingerprint kept for 2 years from last enrollment
- Inactive machines: deleted after 90 days of inactivity
- Deleted data: purged from all backups within 30 days

**Your Control:**
- Request deletion anytime: **delete@example.com**
- Request export anytime: **export@example.com**
- Disable future enrollment: contact support

### Self-Hosted
Your retention policy. You decide:
- Keep permanently
- Delete on schedule
- Archive to cold storage
- Encrypt before deletion

---

## 13. Children & Special Categories

The SDK does **not** knowingly collect data about children under 13.

If you are running a school or youth program:
- Contact **compliance@example.com**
- We offer education-focused deployment options
- Parental consent may be required

---

## 14. Changes to This Agreement

We may update this ULA:
- **Public Notice:** Changes posted at example.com/ula
- **Email Notice:** To enrolled contacts (if contact info provided)
- **Effective Date:** 30 days after posting
- **Your Choice:** Accept new terms or opt out

If you disagree with changes and are using public enrollment:
- Export your machine identity certificate
- Deploy your own controller
- No penalty for migration

---

## 15. Limitations & Disclaimers

### Liability
Kontango provides the SDK "as is" without warranty. Except for gross negligence:
- Kontango is not liable for data loss or breach of your private controller
- Kontango is not liable for misuse of exported identities
- Kontango's total liability is limited to fees paid (if any)

### Indemnification
You agree to indemnify Kontango against:
- Claims from your use of machine fingerprints
- Violations of this agreement
- Illegal use of the SDK

---

## 16. Contact Us

### Privacy & Data Questions
**Email:** privacy@example.com  
**Portal:** example.com/contact

### Legal Requests (Law Enforcement)
We respond to valid legal requests.  
**Email:** legal@example.com

### Self-Hosting Support
**Docs:** docs.example.com/self-hosted  
**Community:** github.com/KontangoOSS/TangoKore

---

## 17. Acceptance

By enrolling a machine with the Kontango SDK, you accept this ULA.

### For Public Enrollment
Running `kontango enroll https://ctrl.example.com` means:
- You accept this ULA
- You understand the data being collected
- You consent to Kontango processing that data

### For Private/Self-Hosted
This ULA applies to data flows within **your** infrastructure:
- You are the data controller
- You determine retention and access policies
- Kontango is just the software provider

---

## 18. Appendix: Data Flow Diagrams

### Public Enrollment Flow
```
┌─────────┐           ┌──────────────┐           ┌───────────────┐
│ Machine │ collect   │  Fingerprint │ HTTPS TLS │ Kontango      │
│         │──────────>│  Data        │──────────>│ Controller    │
│         │           │              │           │               │
│         │           │ • Hostname   │           │ • Fingerprint │
│         │           │ • CPU info   │           │   stored      │
│         │           │ • Memory     │           │ • ACL created │
│         │           │ • MAC addrs  │           │ • Cert issued │
│         │           │ • OS version │           │               │
└─────────┘           └──────────────┘           └───────────────┘
```

### Private/Self-Hosted Flow
```
┌─────────┐           ┌──────────────┐           ┌───────────────┐
│ Machine │ collect   │  Fingerprint │  HTTPS    │ Your          │
│         │──────────>│  Data        │─────────>│ Controller    │
│         │           │              │           │               │
│         │           │ (Same)       │           │ • Fingerprint │
│         │           │              │           │   in your DB  │
│         │           │              │           │ • Your control│
└─────────┘           └──────────────┘           └───────────────┘

Your Infrastructure (Private Network)
├─ Controller
├─ Database
├─ NATS server
└─ Ziti mesh
```

---

## 19. Frequently Asked Questions

### Q: Can Kontango see my machine data if I self-host?
**A:** No. If you run your own controller, Kontango has zero access to your fingerprints. The controller software is open-source and runs entirely on your infrastructure.

### Q: What if I want to delete all my data from the public endpoint?
**A:** Contact privacy@example.com with your machine IDs. We'll delete within 24 hours and remove from backups within 30 days.

### Q: Can I migrate from public to private?
**A:** Yes. Export your machine's identity certificate and configure it to connect to your private controller. No additional enrollment needed.

### Q: Does Kontango sell my fingerprint data?
**A:** No. Kontango never sells, shares, or monetizes machine fingerprints. Data is used only for overlay network access control.

### Q: Is this GDPR compliant?
**A:** Yes, with caveats. If you self-host, compliance is your responsibility. If you use public enrollment, we have a GDPR Data Processing Agreement. Contact us to execute it.

---

**End of Universal License Agreement**

*For questions or concerns, contact: legal@example.com*
