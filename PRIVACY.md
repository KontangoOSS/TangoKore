# Privacy & Data Control

**Kontango Machine SDK — Your Data, Your Control**

---

## Quick Facts

| Question | Answer |
|----------|--------|
| **What data is collected?** | Machine fingerprints: hostname, CPU, memory, network MACs, OS version |
| **Who sees it?** | Public: Kontango. Private: only you (if self-hosted) |
| **Can I avoid it?** | Run your own controller—fingerprints stay private |
| **Can I delete it?** | Yes—contact privacy@example.com or delete from your controller |
| **Is it encrypted?** | Yes—TLS in transit, AES-256 at rest (public) |
| **Can you sell it?** | No—Kontango never monetizes machine data |
| **Full terms?** | See [ULA.md](ULA.md) |

---

## Three Scenarios

### 1. Using Public Enrollment (ctrl.example.com)
```
Your Machine          Kontango Controller
     │                       │
     ├─ send fingerprint ──→ │ (TLS encrypted)
     │                       │
     │ ← receive identity ── │
     │
Kontango's Policy:
  • Fingerprints stored in database
  • Kept for 2 years from last enrollment
  • Accessible to Kontango operations team
  • Covered by example.com privacy policy & GDPR DPA
```

**Your Controls:**
- ✓ See disclosure before enrollment
- ✓ Abort by pressing 'n' in TUI
- ✓ Export machine certificate
- ✓ Delete record anytime (email privacy@example.com)
- ✓ Migrate to private controller

### 2. Running Your Own Controller
```
Your Machine          Your Controller
     │                       │
     ├─ send fingerprint ──→ │ (in YOUR network)
     │                       │
     │ ← receive identity ── │
     │
Your Policy:
  • Fingerprints in YOUR database
  • Stored locally (your backups)
  • Accessible only to you
  • Completely private
```

**Your Controls:**
- ✓ All data stays private
- ✓ Delete anytime from your database
- ✓ Set your own retention policy
- ✓ Air-gap from public internet
- ✓ Full compliance control

**How to self-host:**
1. `git clone https://github.com/KontangoOSS/kontango-controller`
2. Follow deployment guide in docs/
3. Point SDK to your controller: `kontango enroll https://your-controller.internal`
4. Fingerprints now stored privately in your infra

### 3. Hybrid Setup
```
Some machines to Public     Some machines to Private
         │                            │
     Kontango                    Your Controller
      (shared)                    (private)
```

**Per-machine control:**
- Production machines → private controller
- Dev/test machines → public controller
- Device-specific policies on each controller

---

## Data Subject Rights

### If Using Public Enrollment

**GDPR (European Union)**
- **Right to Access:** See what fingerprints Kontango stores about your machines
- **Right to Delete:** Request deletion anytime
- **Right to Portability:** Export your machine identity
- **Right to Correction:** Update hostname or other identifiers
- **Contact:** privacy@example.com

**CCPA (California)**
- **Right to Know:** What data Kontango collects
- **Right to Delete:** Request deletion
- **Right to Opt-Out:** Stop future data collection
- **Contact:** privacy@example.com

**Response Time:** 24 hours for deletion requests, 5 business days for complex requests

### If Self-Hosting

You are the data controller. You have all rights over your data:
- Full access to your database
- Delete what you want, when you want
- Set your own retention policies
- No third-party processing needed

---

## Data Minimization

### What's Collected
✓ Hostname, OS, kernel, architecture  
✓ CPU model/count, memory size  
✓ Network interface MAC addresses  
✓ System machine ID, motherboard serial  

### What's NOT Collected
✗ Passwords or private keys  
✗ Application data or user files  
✗ Browser history or personal information  
✗ Network traffic or logs (unless explicitly configured)  
✗ Biometric data  
✗ Location data  
✗ Anything in /home, /root, or user directories  

---

## Security Controls

### In Transit (Public Enrollment)
- **TLS 1.3** required—nothing sent unencrypted
- **Hostname verification** enabled—prevents MITM attacks
- **Certificate pinning** (coming soon)—additional verification

### At Rest (Public)
- **AES-256 encryption** for all fingerprints
- **Key separation** — different keys for different data
- **Regular key rotation** — quarterly
- **Database isolation** — fingerprints in separate schema
- **Access controls** — only operations team can query

### At Rest (Self-Hosted)
Your choice:
- Use database encryption (PostgreSQL TDE)
- Use full-disk encryption
- Use backup encryption
- Whatever your compliance requires

---

## Transparency Features

### Disclosure at Enrollment Time

**Non-interactive:**
```
kontango enroll https://ctrl.example.com --no-tui
    ↓
Shows FINGERPRINTING DISCLOSURE banner
    ↓
Lists all data being collected
    ↓
Explains why and privacy controls
```

**Interactive (TUI):**
```
kontango enroll https://ctrl.example.com
    ↓
Shows method selection
    ↓
Collects fingerprint
    ↓
Shows CONFIRM step with:
  • All collected data listed
  • Explanation of why it's needed
  • Privacy notice
    ↓
User presses 'y' to proceed or 'n' to abort
```

### Logs & Audit

Public enrollment sends these events:
```
verify:    fingerprint_match  (is this machine known?)
verify:    os_validation      (is OS supported?)
verify:    banned_check       (is machine blacklisted?)
decision:  status=quarantine  (initial permission level)
identity:  certificate issued (enrollment complete)
```

All logged locally and available in controller.

Self-hosted enrollment—all logs in your database.

---

## Deletion Process

### Requesting Deletion (Public Enrollment)

**Email:** privacy@example.com
**Include:**
- Your machine hostname(s) or ID(s)
- Date(s) of enrollment (if available)
- Reason (optional)

**We will:**
1. Acknowledge within 24 hours
2. Delete from active database
3. Purge from all backups within 30 days
4. Confirm deletion within 5 business days

### Deleting Yourself (Self-Hosted)

```bash
# Connect to your controller database
psql kontango

# Delete a specific machine
DELETE FROM machines WHERE hostname = 'my-machine';

# Verify deletion
SELECT COUNT(*) FROM machines;
```

---

## Migration

### Moving from Public to Private

```bash
# 1. Export your machine identity from public enrollment
kontango export --machine-id <id>
# Saves: machine-identity.p12

# 2. Set up your private controller
git clone https://github.com/KontangoOSS/kontango-controller
cd kontango-controller && make deploy

# 3. Configure SDK to use your controller
kontango config set controller=https://your-controller.internal

# 4. Re-enroll (same identity, new controller)
kontango enroll https://your-controller.internal --identity machine-identity.p12

# 5. Request deletion from public endpoint
# Contact privacy@example.com
```

Fingerprints no longer sent to Kontango after this.

---

## Compliance Checklists

### Self-Hosted Deployments

If you're responsible for compliance (HIPAA, SOC 2, PCI-DSS):

**Data Isolation:**
- ✓ Controller database is isolated from application databases
- ✓ Fingerprints are encrypted separately
- ✓ Access logs are maintained

**Access Control:**
- ✓ Only your administrators can query fingerprints
- ✓ Network access restricted (firewall rules)
- ✓ All access logged

**Retention:**
- ✓ Define your own retention policy
- ✓ Automated deletion scripts (on schedule)
- ✓ Backup encryption enabled

**Incident Response:**
- ✓ Logs show all enrollments and access
- ✓ Deletion is immediate and auditable
- ✓ Blockchain-style audit logs (optional)

---

## Public Infrastructure Details

### Kontango's Data Processors

If you use public enrollment, your fingerprints are processed by:

| Component | Provider | Data Handling |
|-----------|----------|---|
| Infrastructure | DigitalOcean | [Privacy Policy](https://www.digitalocean.com/legal/privacy/) |
| Database | PostgreSQL | Managed by DO, encrypted at rest |
| Message Bus | NATS | Managed by DO, for ACL delivery |
| DNS | Kontango | Kontango-controlled |

**No third-party analytics on machine data.**

---

## Questions & Support

### Privacy Questions
**Email:** privacy@example.com  
**Response Time:** 24 hours

### Data Deletion Requests
**Email:** privacy@example.com (title: "DATA DELETION REQUEST")  
**Response Time:** 24 hours for deletion, 5 days for confirmation

### Legal/Compliance Requests
**Email:** legal@example.com

### Technical Support
**Self-Hosted:** docs.example.com/self-hosted  
**Community:** github.com/KontangoOSS/TangoKore/discussions

---

## Related Documents

- **[ULA.md](ULA.md)** — Full terms and conditions
- **[SECURITY.md](SECURITY.md)** — Security architecture (coming soon)
- **example.com/privacy** — Public privacy policy
- **example.com/terms** — Terms of service

---

**Last Updated:** April 5, 2026  
**Version:** 1.0  
**License:** This document is part of Kontango SDK
