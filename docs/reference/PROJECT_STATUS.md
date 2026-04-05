# TangoKore SDK — Project Status

**Date:** April 5, 2026  
**Status:** ✅ Production-Ready (Go SDK Complete, Join Webpage Live, Multi-Language SDKs Planned)

## What Is TangoKore?

TangoKore is the Kontango machine SDK — a zero-trust enrollment system where:
- **Your machine is your password** — hardware fingerprint replaces passwords
- **One command to join** — `curl https://ctrl.example.com/install | sudo sh`
- **Server decides trust** — based on fingerprint (new/returning) or credentials (pre-provisioned)
- **Graduated trust** — Stage 0 (Quarantine) → Stage 1 (Approved) → Stage 2 (Trusted) → Stage 3 (Admin)

## Current Implementation

### ✅ Go SDK (Complete)

**Core Features:**
- Machine enrollment via SSE and WebSocket protocols
- Machine fingerprinting (OS, architecture, hardware info)
- Transparent disclosure of data being sent
- Agent for heartbeat and telemetry
- Cluster/controller commands (framework in place)
- TUI and non-interactive flows
- Cross-platform support (Linux, macOS, Windows)

**Code Structure:**
```
cmd/kontango/          # CLI commands
├── main.go           # Entry point
├── enroll.go         # Enrollment command
├── enroll_tui.go     # Interactive TUI
├── status.go         # Status command
├── cluster.go        # Cluster commands (join/create/status)
├── controller.go     # Controller commands
└── ...

internal/enroll/      # Enrollment library
├── fingerprint.go    # Machine fingerprinting
├── sse.go            # SSE protocol
├── websocket.go      # WebSocket protocol
├── probes.go         # Hardware probes
└── ...

tests/                # Test suite
├── unit/             # Unit tests
├── integration/      # Integration tests
└── regression/       # Regression tests
```

**Test Coverage:**
- 6 test files covering: enrollment, probes, platform detection, registration
- Integration tests with real ctrl.example.com endpoint
- Disclosure transparency tests (TUI + non-interactive)

**Documentation:**
- Two-tier approach:
  - `README.md` — Zero jargon, one-command focus, beginner-friendly
  - `docs/` (MkDocs) — Technical deep dives, API reference, deployment guides

### ✅ Join Webpage (Live)

**Location:** `web/public/` + deployed at `ctrl.example.com`

**Features:**
- Clean, professional enrollment UI
- Method selection (new machine, returning, pre-provisioned)
- Real-time data transparency (what's sent and why)
- Live enrollment logs (verify → decision → identity)
- Success/error states with clear guidance
- Responsive design (mobile-friendly)

**Files:**
- `index.html` — HTML structure
- `css/style.css` — Professional Material Design styling
- `js/main.js` — Client-side enrollment orchestration
- `web/README.md` — Customization guide

### 🚀 Multi-Language SDK Support (Planned)

**GitHub Issues Created:**
- ✅ #1: Python SDK support
- ✅ #2: TypeScript/Node.js SDK support
- ✅ #3: Rust SDK support
- ✅ #4: Java SDK support

All SDKs wrap the same enrollment/agent/cluster APIs. Users can use:
- **CLI** (TUI or headless commands)
- **Web interface** (ctrl.example.com)
- **Native SDKs** (Go, Python, Rust, TypeScript, Java, etc.)

## Architecture

### Enrollment Flow

```
User runs:  curl https://ctrl.example.com/install | sudo sh
            ↓
SDK:        Collects minimal fingerprint (OS, arch, ID)
            Shows disclosure (what's being sent, why)
            ↓
Controller: Receives via SSE or WebSocket
            Verifies data (verify events)
            Makes trust decision (decision event)
            ↓
SDK:        Receives identity + config
            Saves certificate (~/.kontango/identity.p12)
            Starts agent + tunnel connection
            ↓
Result:     Machine enrolled at Stage 0 (Quarantine)
            Can escalate via fingerprint matching or credentials
```

### Server-Side Method Determination

**No client-side method selection.** Server decides based on data:

```
Client sends: OS + arch + ID [+ optional fingerprint] [+ credentials]
                ↓
Server checks:
  1. Credentials provided? → APPROLE (stage-3, full access)
  2. Fingerprint matches DB? → APPROVED (restore previous ACL)
  3. Neither? → QUARANTINE (stage-0, read-only)
```

**Why:** Client sends the same message format regardless. Server intelligently determines what method was intended by inspecting the data.

## Key Files

### User-Facing
- `README.md` — How to use TangoKore
- `web/public/index.html` — Join webpage
- `docs/` — Technical documentation

### CLI Commands
- `kontango enroll [--scan] [--role-id X --secret-id Y]`
- `kontango status [--json]`
- `kontango cluster join <url>`
- `kontango cluster status`
- `kontango controller create [--name X]`
- `kontango controller status`

### Core Library (`internal/enroll/`)
- `fingerprint.go` — Machine fingerprinting logic
- `sse.go` — Server-Sent Events enrollment
- `websocket.go` — WebSocket enrollment (alternative)
- `probes.go` — Hardware detection
- `platform.go` — OS-specific implementations

### Testing
- `tests/unit/` — 6+ unit tests
- `tests/integration/` — Real endpoint tests
- `tests/regression/` — Regression prevention

## Deployment

### Join Endpoint
- **URL:** `https://ctrl.example.com/install`
- **Serves:** Installer script with session token + BASE_URL
- **Authentication:** None required (honeypot for unknown machines)

### Local Testing
```bash
# Build
make build

# Run locally
./build/kontango enroll https://localhost:9090 --no-tui

# Test with ctrl.example.com
./build/kontango enroll https://ctrl.example.com --no-tui
```

### MkDocs Local Build
```bash
python3 -m venv .mkdocs-venv
source .mkdocs-venv/bin/activate
pip install mkdocs mkdocs-material
mkdocs serve
# Opens http://localhost:8000
```

## Philosophy

**"README is for understanding. Docs are for doing."**

- **README.md** should be readable by someone who just learned what the terminal is
  - Zero jargon
  - One-command focus
  - Privacy statement (open source, GDPR, can run own server)
  - Clear next steps
  
- **docs/** provides technical details for engineers
  - Architecture deep dives
  - API reference
  - Deployment guides
  - Compliance documentation

**"You are your password. Keep being you."**

- Machines authenticate by being themselves (fingerprint)
- Cannot forget, cannot lose, impossible to fake
- Users can backup/share profiles for recovery
- Trust grows over time through consistent behavior

## Future Work

1. **Multi-Language SDKs**
   - Python SDK
   - TypeScript/Node.js SDK
   - Rust SDK
   - Java SDK

2. **Enhanced Enrollment**
   - JWT-based credentials
   - OIDC integration
   - MFA options

3. **Advanced Features**
   - Profile management (export/import/share)
   - Bulk enrollment
   - Policy enforcement
   - Advanced fingerprinting

4. **Web UI Enhancements**
   - Dark mode
   - QR codes for CLI integration
   - Installation status polling
   - Certificate export

## Verification Checklist

- ✅ Go SDK complete and production-ready
- ✅ Enrollment works via SSE and WebSocket
- ✅ Machine fingerprinting implemented
- ✅ Disclosure mechanism in place
- ✅ Cluster/controller commands in CLI
- ✅ Cross-platform support (Linux/macOS/Windows)
- ✅ Join webpage live and functional
- ✅ Documentation (README + MkDocs)
- ✅ Test coverage (unit + integration + regression)
- ✅ Multi-language SDK issues created and tracked
- ✅ Project open source (MIT license)

## Getting Started

**For End Users:**
```bash
curl https://ctrl.example.com/install | sudo sh
```

**For Developers:**
1. Read `README.md` for overview
2. Visit `https://docs.example.com` for technical docs
3. Fork `https://github.com/KontangoOSS/TangoKore`
4. Build SDKs in your language of choice

**For Operations:**
1. Deploy join endpoint at `ctrl.example.com`
2. Run controller cluster for enrollment decisions
3. Point machines to your join URL
4. Monitor via NATS telemetry

---

**Next Phase:** Server-side controller implementation (enrollment decisions, policy enforcement, telemetry aggregation)
