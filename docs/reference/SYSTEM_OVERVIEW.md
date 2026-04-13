# TangoKore System Overview

**What is TangoKore?**

TangoKore is the **complete zero-trust machine enrollment system**. It's not just an SDK — it's the foundation that includes:

1. **SDK** — CLI binary and enrollment library (Go)
2. **Controller** — Enrollment API server (schmutz-controller)
3. **Gateway** — Edge firewall (schmutz)
4. **Web UI** — User-friendly enrollment interface (can run on controller OR standalone)
5. **Documentation** — Guides for users and operators

All of these components work together to let machines join a secure mesh network with one command:

```bash
curl https://controller.example.com/install | sudo sh
```

---

## The Three Components

### 1. TangoKore SDK (`~/git/kore/TangoKore/`)

**What it is:**
- User-facing CLI and library
- Enrollment logic (SSE + WebSocket protocols)
- Agent for telemetry
- Documentation (README + MkDocs)
- **Web UI** (can be embedded in controllers OR run standalone)

**Where users interact:**
```bash
# Install via one-liner
curl https://controller.example.com/install | sudo sh

# Or via web interface
https://controller.example.com/
```

**What's inside:**
```
cmd/kontango/        - CLI commands (enroll, status, cluster, controller)
internal/enroll/     - Core enrollment library
tests/              - Unit, integration, regression tests
docs/               - Technical reference (MkDocs)
web/public/         - Web UI (standalone component)
  ├── index.html     - Enrollment interface
  ├── css/style.css  - Material Design styling
  └── js/main.js     - Decision tree logic
```

### 2. schmutz-controller (`~/git/kore/schmutz-controller/`)

**What it is:**
- The actual enrollment API server
- Runs on every controller (ctrl-1, ctrl-2, ctrl-3)
- Decides trust levels (quarantine → approved → trusted → admin)
- Manages secret store (OpenBao)

**What it serves:**
```
GET  /install              - Installer script with session token
GET  /join                 - Web UI (currently offline)
POST /api/enroll/stream    - SSE enrollment endpoint
POST /api/ws/enroll        - WebSocket enrollment endpoint
POST /api/verify/*         - Verification checks
```

**How it connects:**
```
Caddy (SSL/routing) → schmutz-controller (API + UI) → OpenBao (secrets) + Ziti (mesh)
```

### 3. schmutz (`~/git/kore/schmutz/`)

**What it is:**
- Edge gateway (L4 firewall)
- Enforces quarantine stage rules
- Sits between untrusted and trusted networks

**What it does:**
- Filters traffic
- Enforces ACLs
- No enrollment logic

---

## The Complete User Journey

### Step 1: Machine Discovers Join Endpoint

User runs one command:
```bash
curl https://controller.example.com/install | sudo sh
```

This connects to the **load-balanced join endpoint** (round-robin across ctrl-1, ctrl-2, ctrl-3).

### Step 2: schmutz-controller /install Handler

Controller's `/install` endpoint:
1. **Fingerprints the connection** (IP, TLS details, JA4)
2. **Screens** (is this a known attacker? Is this IP banned?)
3. **Generates session token** (one-time use)
4. **Serves shell script** with:
   - `BASE_URL` (which controller to enroll with)
   - `SESSION_TOKEN` (proof of download)
   - SDK command: `kontango enroll $BASE_URL --session $SESSION_TOKEN --no-tui`

### Step 3: Machine Runs SDK

The installer script runs:
```bash
kontango enroll https://controller.example.com \
  --session <session_token> \
  --no-tui
```

SDK:
1. **Shows disclosure** (what data will be sent)
2. **Collects fingerprint** (OS, arch, hardware info)
3. **Sends to /api/enroll/stream** (SSE protocol)
4. **Receives events**: verify → decision → identity
5. **Saves identity** to `~/.kontango/identity.p12`
6. **Starts Ziti tunnel**

### Step 4: Machine in Quarantine

Controller's decision:
- **New fingerprint** → QUARANTINE (stage-0, read-only)
- **Known fingerprint** → APPROVED (stage-1, restore ACL)
- **Valid credentials** → TRUSTED (stage-3, full access)

Machine can:
- See services in quarantine (read-only)
- Prove itself over time
- Escalate trust through consistent behavior

---

## Where Users Can Enroll

### Option 1: Web Interface (Easiest)

Visit: `https://controller.example.com/`

**Served by:** 
- Option A: schmutz-controller at `frontend/` (embedded, convenient)
- Option B: Standalone web service (flexible, can customize per instance)

**UI:** Material Design, skill-level selection  
**Flows:** Simple (beginner) or Advanced (developer)

```
User chooses:
1. Machine identity (auto or custom)
2. Optional fields (what to expose)
3. Submit and enroll
```

### Option 2: CLI Installer (Fastest)

```bash
curl https://controller.example.com/install | sudo sh
```

**Served by:** schmutz-controller `/install` endpoint  
**Flow:** Non-interactive (one-liner)  
**What SDK does:** Show disclosure, collect fingerprint, enroll

### Option 3: CLI TUI (Most Interactive)

```bash
kontango enroll https://controller.example.com
```

**Flow:** Interactive dialogs  
**What user sees:** Questions about identity, optional fields, credentials  
**What SDK does:** Same enrollment, just interactive prompts

---

## The Decision Tree (Same Everywhere)

Whether users enroll via **web**, **CLI installer**, or **CLI TUI**, they make three identical choices:

### Choice 1: Machine Identity
- **Auto-generate** (recommended): `machine-<random>`
- **Custom**: User provides their own ID

### Choice 2: Optional Fields
Beyond required (OS, arch, ID), user chooses to include:
- Hostname
- OS version
- Kernel version
- CPU info
- Memory
- MAC addresses
- System UUID
- Serial number

### Choice 3: Credentials (Optional)
If pre-provisioned:
- AppRole `role_id` + `secret_id`
- Grants immediate trusted (stage-3) access

---

## The Four Trust Levels

All machines start at **Quarantine** by default:

```
Stage 0: QUARANTINE (New/Unknown)
├─ Read-only access
├─ Cannot modify services
├─ See quarantine services only
└─ Can escalate by proving itself

Stage 1: APPROVED (Returning/Recognized)
├─ Fingerprint matched
├─ Restored permissions from previous enrollment
└─ Escalate to trusted over time

Stage 2: TRUSTED (Proven Consistent)
├─ Shown consistent behavior
├─ Higher permissions
├─ Can deploy services

Stage 3: ADMIN (Pre-Provisioned/Proven)
├─ Valid credentials provided
├─ Full cluster access
└─ Can manage infrastructure
```

---

## Philosophy

### "You Are Your Password"
- No passwords needed
- Machine's fingerprint = identity
- Cannot forget, cannot lose, impossible to fake

### "Your Path Is What You Choose"
- Same options everywhere (web, CLI, TUI)
- Respect for different skill levels
- No gatekeeping or intimidation

### "Every Controller Is a Doorway"
- Whether user joins via public (`controller.example.com`) or private server
- They get the same quality experience
- Same decision tree, same trust model

---

## Repository Structure

```
TangoKore/
├── README.md                      - User entry point
├── SYSTEM_OVERVIEW.md            - This file
├── ARCHITECTURE_CLARIFICATION.md - How pieces fit together
├── PROJECT_STATUS.md             - Current implementation status
├── CURRENT_STATUS.md             - Snapshot of what's done
├── DEPLOY_JOIN_ENDPOINT.md      - How to get controller.example.com working
│
├── cmd/kontango/                 - CLI implementation
│   ├── enroll.go                - Enrollment command
│   ├── enroll_tui.go            - Interactive TUI
│   ├── status.go                - Status command
│   └── ...
│
├── internal/enroll/              - Core enrollment library
│   ├── fingerprint.go           - Machine fingerprinting
│   ├── sse.go                   - SSE protocol
│   ├── websocket.go             - WebSocket protocol
│   └── ...
│
├── web/public/                   - UI patterns for controllers
│   ├── index.html               - Main enrollment interface
│   ├── css/style.css            - Material Design styling
│   └── js/main.js               - Decision tree logic
│
├── docs/                         - Technical documentation (MkDocs)
│   ├── getting-started/         - Installation guides
│   ├── philosophy/              - Why fingerprints work
│   ├── architecture/            - Technical deep dives
│   └── privacy/                 - Compliance documentation
│
├── tests/                        - Test suite
│   ├── unit/                    - Unit tests
│   ├── integration/             - End-to-end tests
│   └── regression/              - Prevent regressions
│
└── web/                         - Web UI (standalone component)
    ├── public/
    │   ├── index.html          - Enrollment interface
    │   ├── css/style.css       - Material Design styling
    │   └── js/main.js          - Decision tree logic
    └── README.md               - Web deployment guide
```

**Web UI Deployment Options:**
- **Embedded:** Part of schmutz-controller (convenient, one deployment)
- **Standalone:** Separate service (flexible, can customize/deploy independently)
- **Both:** Controller serves it, but updates can come separately

---

## Getting Started

### For End Users
```bash
# One command to join
curl https://controller.example.com/install | sudo sh

# Or use web interface
https://controller.example.com/
```

### For Operators
1. Deploy controller (schmutz-controller) on your infrastructure
2. Point DNS to your controller
3. Users join via your controller instead of public `controller.example.com`
4. Same experience, total privacy (data stays on your network)

### For Developers
1. Read `README.md` and `docs/` for how it works
2. Build CLI: `make build`
3. Run tests: `make test-all`
4. Customize web UI: `web/public/`
5. Extend SDK: `internal/enroll/`

---

## Current Status

### ✅ Completed
- Go SDK implementation (complete)
- Enrollment protocols (SSE + WebSocket)
- Machine fingerprinting
- Trust level logic
- CLI and TUI
- Documentation
- **New web UI** (Material Design, decision tree)
- Test suite

### ⚠️ In Progress
- **Get join endpoint back online** (schmutz-controller service)
- Integrate new web UI into controllers
- Ensure all controllers serve UI

### 🚀 Next Phase
- Multi-language SDKs (Python, Rust, TypeScript, Java)
- TUI alignment with web decision tree
- Advanced features (profiles, backup, sharing)

---

## What's Next

**Immediate (This Week):**
1. Diagnose why `controller.example.com` is offline
2. Restore schmutz-controller service
3. Integrate new web UI (from TangoKore/web/public/)

**Short Term (This Sprint):**
1. Test full enrollment flow (web + CLI)
2. Verify all three controllers serving UI
3. Monitor for issues

**Medium Term (Next Sprint):**
1. Build Python SDK
2. Update CLI TUI for consistency
3. Add advanced enrollment features

---

## Philosophy Quote

> "You are your password. Keep being you."
>
> Your machine's fingerprint is impossible to fake. You can't forget it, you can't lose it, and you can't share it. It's your identity, trusted over time through consistent behavior.
>
> And you control what information you share. No tracking, no selling, no secrets. Just honest, transparent, respectful enrollment.
>
> Your path is what you choose — whether you're a beginner clicking buttons or a developer controlling every byte. We'll help you either way.

---

**TangoKore:** Zero-trust enrollment where your machine is your password.

Make it secure. Make it simple. Make it kind.
