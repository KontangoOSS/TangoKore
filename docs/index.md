# TangoKore SDK Documentation

Welcome! This is the technical documentation for TangoKore.

If you just want to get started, **[read the README](../README.md) first**. It's simple and explains everything you need to know.

If you're looking for technical details, architecture, or advanced configuration, you're in the right place.

---

## Quick Navigation

### Just Getting Started?

- **[Installation](getting-started/installation.md)** - How to set up your first machine
- **[Your First Machine](getting-started/first-machine.md)** - Step-by-step walkthrough
- **[Understanding Trust Levels](getting-started/trust-levels.md)** - How your machine gains permissions over time

### Want to Understand the Philosophy?

- **[Why Fingerprinting](philosophy/why-fingerprinting.md)** - Why this is better than passwords
- **[You Are Your Password](philosophy/you-are-your-password.md)** - The core concept
- **[Never Forget](philosophy/never-forget.md)** - How to back up and recover

### Building with TangoKore?

- **[Enrollment Flows](user-guide/enrollment-flows.md)** - How machines enroll
- **[Managing Your Profile](user-guide/managing-profile.md)** - Backup, sharing, recovery
- **[Endpoints Reference](api/endpoints.md)** - API documentation

### Running Your Own Server?

- **[Deployment](operations/deployment.md)** - How to deploy TangoKore
- **[Architecture Overview](architecture/overview.md)** - How it all works
- **[Trust Model](architecture/trust-model.md)** - How trust decisions are made

---

## Core Concepts

### Fingerprinting as Identity

Instead of passwords, TangoKore uses your machine's **fingerprint** as its identity:
- Operating system
- Hardware specifications (CPU, memory, motherboard)
- Network interfaces
- System behavior patterns

This combination is unique to your machine and impossible to fake.

### Trust Levels

Machines progress through trust levels:

1. **Quarantine (Stage 0)** - New machine, read-only access (safe default)
2. **Approved (Stage 1)** - Recognized machine, restored permissions
3. **Trusted (Stage 2)** - Proven consistency, higher access
4. **Admin (Stage 3)** - Pre-provisioned or fully trusted, full access

### The One-Command Flow

```
User runs:  curl https://ctrl.example.com/install | sudo sh
           ↓
Controller: Fingerprints connection, generates session token, serves installer
           ↓
Installer:  Collects machine data, shows disclosure, calls SDK
           ↓
SDK:        Sends fingerprint + session to /api/enroll/stream
           ↓
Controller: Verifies, decides trust level, streams back events + identity
           ↓
Machine:    Receives certificate, installs tunnel, connects to mesh
           ↓
Result:     Machine is authenticated and on the secure network
```

---

## Privacy & Security

### What We Collect

- Operating system and version
- Hardware information (CPU, memory, motherboard IDs)
- Network interface MACs
- System identifiers (machine ID, UUID)

This is **all public information** that `uname -a` or similar commands show.

### What We Don't Collect

- Passwords or credentials
- Application data or files
- Browser history or personal information
- Network traffic
- Anything in your home directory

### Your Privacy Control

**Using our public server:** Your data is encrypted, secure, and deletable anytime (email privacy@example.com)

**Running your own server:** Your data never leaves your network. You control everything.

### 100% Open Source

All code is public MIT license. Read it. Audit it. Trust it.

```
github.com/KontangoOSS/TangoKore
```

---

## Common Questions

**Q: Is this really secure?**
A: Yes. Your machine's fingerprint is impossible to fake. To impersonate your machine, an attacker would need your exact hardware + months of perfect behavioral mimicry while your real machine keeps proving itself.

**Q: What if I lose my machine?**
A: Back up your `~/.kontango/machine.json` and restore from any machine. You're recognized instantly by your fingerprint.

**Q: What if I want my data private?**
A: Run your own TangoKore server. Your data never leaves your network.

**Q: Can you sell my data?**
A: No. We never monetize machine fingerprints. It's just hardware info for authentication.

**Q: Do you track my usage?**
A: No. We don't collect usage metrics, analytics, or behavior data beyond what's needed for authentication.

---

## Next Steps

1. Read the **[README](../README.md)** for a simple overview
2. Follow **[Installation](getting-started/installation.md)** to set up your first machine
3. Check **[Architecture](architecture/overview.md)** if you want to understand the technical details
4. Explore the **[API Reference](api/endpoints.md)** if you're building on top of TangoKore

---

## Still Have Questions?

- **Privacy questions:** privacy@example.com
- **Technical issues:** [GitHub Issues](https://github.com/KontangoOSS/TangoKore/issues)
- **General info:** [example.com](https://example.com)

---

**TangoKore: You are your password. Keep being you.**
