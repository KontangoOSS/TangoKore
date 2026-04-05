# SDK ↔ Controller Integration Checklist

**Verify complete alignment between TangoKore SDK and schmutz-controller**

---

## Endpoints Alignment

### Install Endpoint

| Component | Implementation | Status |
|-----------|---|---|
| **SDK** | Expects `/install` endpoint | ✅ Implemented |
| **Controller** | Serves `/install` at `/api/install` | ✅ schmutz-controller/src/cmd/.../routes.go |
| **Flow** | curl https://controller/install → shell script | ✅ |
| **Session Token** | Generated and embedded in script | ✅ |
| **Environment** | BASE_URL passed to script | ✅ |

### SSE Enrollment Endpoint

| Component | Implementation | Status |
|-----------|---|---|
| **SDK** | Sends POST to `/api/enroll/stream` | ✅ SSEEnroll() |
| **Controller** | Handles POST at `/api/enroll/stream` | ✅ HandleSSE() |
| **Protocol** | Server-Sent Events (streaming) | ✅ |
| **Events** | verify → decision → identity | ✅ |
| **Fingerprint** | Minimal (OS, arch, ID) + optional details | ✅ |
| **Credentials** | role_id, secret_id, session optional | ✅ |

### WebSocket Enrollment Endpoint

| Component | Implementation | Status |
|-----------|---|---|
| **SDK** | Connects to `/api/ws/enroll` | ✅ WebSocketEnroll() |
| **Controller** | Handles WS at `/api/ws/enroll` | ✅ HandleWebSocket() |
| **Protocol** | WebSocket with JSON messages | ✅ |
| **Flow** | hello → 4 probes → identity | ✅ |
| **Probes** | OS, hardware, network, system | ✅ |

---

## Data Format Alignment

### Minimal Fingerprint (Required)

```
SDK Sends:
  {
    "os": "linux",           ← auto-detected
    "arch": "amd64",         ← auto-detected
    "issued_id": "a1b2c3d4"  ← auto-issued or user-provided
  }

Controller Expects:
  ✅ OS field
  ✅ Arch field
  ✅ Issued ID field
```

### Optional Fingerprint (Enhanced)

```
SDK Can Send:
  {
    "hostname": "...",
    "os_version": "...",
    "kernel_version": "...",
    "machine_id": "...",
    "cpu_info": "...",
    "memory_mb": ...,
    "mac_addrs": [...],
    "serial_number": "..."
  }

Controller Expects:
  ✅ All fields recognized
  ✅ Fingerprint hash computed
  ✅ Compared against database
```

### Credentials (Optional)

```
SDK Can Send:
  {
    "role_id": "...",    ← AppRole
    "secret_id": "...",  ← AppRole
    "session": "...",    ← One-time token
    "token": "..."       ← JWT (future)
  }

Controller Validates:
  ✅ AppRole against Bao
  ✅ Session token from Bao
  ✅ JWT signature (if provided)
```

### Response Format

```
Controller Sends (SSE):
  event: verify
  data: {"check":"fingerprint_match","passed":true}
  
  event: decision
  data: {"status":"approved"}
  
  event: identity
  data: {
    "id":"...",
    "nickname":"...",
    "status":"...",
    "identity":"...",  ← PKCS12 cert
    "config":{
      "hosts":[...],
      "tunnel":{...}
    }
  }

SDK Expects:
  ✅ verify events (multiple)
  ✅ decision event (one)
  ✅ identity event (one)
  ✅ All fields present in identity
```

---

## Trust Level Alignment

### Decision Logic

| Scenario | SDK Behavior | Controller Decision | Result |
|----------|---|---|---|
| New machine, no creds | Sends minimal data | fingerprint unknown, no creds → | QUARANTINE (stage-0) |
| Returning machine | Sends same fingerprint | fingerprint match → | APPROVED (restored ACL) |
| With AppRole creds | Sends creds + fingerprint | creds valid → | TRUSTED (stage-3) |
| With session token | Sends token + fingerprint | token valid → | APPROVED/TRUSTED |

### Stage Levels

```
SDK Expects (stage-0):
  ✅ Read-only access
  ✅ Can receive config
  ✅ Cannot modify services
  ✅ Cannot escalate

SDK Expects (stage-1):
  ✅ Read-write access
  ✅ Can write to assigned services
  ✅ Can update own config

SDK Expects (stage-2):
  ✅ Deploy services
  ✅ Manage ACLs (own)
  ✅ View cluster state

SDK Expects (stage-3):
  ✅ Full cluster admin
  ✅ Deploy infrastructure
  ✅ Manage all ACLs
```

---

## Verification Events Alignment

### Verify Events

| Event | SDK Expects | Controller Sends | Purpose |
|-------|---|---|---|
| `fingerprint_match` | true/false | Checks DB | Is this a known machine? |
| `os_validation` | true/false | Validates OS | Is OS supported? |
| `banned_check` | true/false | Checks ban list | Is machine blacklisted? |
| `credentials_valid` | true/false | Validates auth | Are creds valid? |
| (more checks...) | ... | ... | ... |

### Controller Must Send All Checks

```
SDK Code:
  for event := range eventStream {
    if event.Type == "verify" {
      log.Printf("%s: %v", event.Check, event.Passed)
    }
  }

Controller Code:
  ✅ Sends verify events for each check
  ✅ All checks completed before decision
  ✅ Each event has: check name, passed bool, confidence
```

---

## Certification & Storage Alignment

### Certificate Format

```
Controller Issues:
  ✅ PKCS12 certificate
  ✅ Valid for Ziti edge
  ✅ Machine-bound (can't be shared to different hardware)

SDK Expects:
  ✅ Can save to ~/.kontango/identity.p12
  ✅ Can load and use with Ziti CLI
  ✅ Can export/backup in profile
```

### Configuration Format

```
Controller Provides:
  {
    "hosts": ["ziti.example.com", "ziti-internal.tango"],
    "tunnel": {
      "endpoint": "ws://ziti:3021/edge",
      "acl": "stage-0",  ← Trust level
      "version": "latest"  ← Ziti version
    }
  }

SDK Expects:
  ✅ Extract hosts (Ziti endpoints)
  ✅ Extract tunnel config
  ✅ Use endpoint to connect
  ✅ Apply ACL from profile
```

---

## Installer Script Alignment

### Generated Script Contents

```
Controller Creates:
  ✓ Shebang: #!/bin/bash
  ✓ BASE_URL set to controller endpoint
  ✓ SESSION_TOKEN embedded
  ✓ Call to `kontango enroll` with flags:
    - URL: $BASE_URL
    - Session: $SESSION_TOKEN
    - --no-tui (non-interactive)
    - Optional: --role-id, --secret-id (if pre-provisioned)

SDK Executes:
  ✓ Receives environment variables
  ✓ Parses command-line flags
  ✓ Calls runEnroll() with parsed args
  ✓ Passes session token to SSEEnroll()
  ✓ Saves identity after success
```

### Script Validation

```bash
# Download installer and check content
curl -k https://localhost/install | head -50

# Should contain:
  - BASE_URL=https://...
  - SESSION_TOKEN=...
  - kontango enroll $BASE_URL --session $SESSION_TOKEN --no-tui
```

---

## Server-Side Method Determination Alignment

### Principle Check

✅ **SDK sends data, Server decides**

```
SDK sends:
  • OS + arch + ID (always, required)
  • + fingerprint match data (optional)
  • + credentials (optional)

Server receives and checks:
  1. Are credentials provided?
     → Yes: Validate → TRUSTED
     → No: Check fingerprint
  
  2. Does fingerprint match database?
     → Yes: Match found → APPROVED
     → No: Unknown → QUARANTINE

Server decides (not client):
  ✅ Never asks client "what method?"
  ✅ Server inspects what was sent
  ✅ Server makes trust decision
  ✅ Client always sends same message format
```

---

## Disclosure & Transparency Alignment

### Pre-Enrollment Disclosure

```
SDK Shows (Both Paths):
  ✓ "KONTANGO SDK DISCLOSURE"
  ✓ "Required minimum: OS, arch, ID"
  ✓ "Optional enhancements: ..."
  ✓ Lists what will be sent
  ✓ Shows privacy notice

TUI Shows (Interactive):
  ✓ Confirm step with collected data
  ✓ Shows "This machine will send:"
  ✓ User can review before confirming
  ✓ Can abort by pressing 'n'
```

### Session Token Tracking

```
Controller Tracks:
  ✓ Session token generated at /install
  ✓ Session token → source IP mapping in Bao
  ✓ Session token checked during enrollment

SDK Sends:
  ✓ Session token in enrollment payload
  ✓ Session token from installer script
  ✓ Allows server to correlate /install → enrollment
```

---

## Cross-Platform Compatibility

### Linux

```
SDK: ✅ Fingerprinting works
  - ip -o link (MACs)
  - /etc/machine-id (system ID)
  - /proc/cpuinfo (CPU info)
  - uname -r (kernel)

Controller: ✅ Recognizes Linux machines
  - Validates kernel version
  - Checks OS support
  - Issues appropriate certs
```

### macOS

```
SDK: ✅ Fingerprinting works
  - ifconfig (MACs)
  - ioreg (system ID, serial)
  - sysctl (CPU info)
  - uname -r (kernel)

Controller: ✅ Recognizes macOS machines
  - Validates OS version
  - Checks CPU architecture
  - Issues appropriate certs
```

### Windows

```
SDK: ✅ Fingerprinting works
  - getmac (MACs)
  - wmic (CPU, serial)
  - registry (machine GUID)
  - ver (OS version)

Controller: ✅ Recognizes Windows machines
  - Validates Windows version
  - Checks architecture
  - Issues appropriate certs
```

---

## Verification Steps

Run these to verify alignment:

### 1. Check Endpoints Exist

```bash
# Controller should have these endpoints
curl -k https://localhost/install -I
# Expected: 200 OK (or 404 if screened)

curl -k https://localhost/api/enroll/stream -I
# Expected: 405 (POST required)

curl -k https://localhost/api/ws/enroll -I
# Expected: 400 (WebSocket upgrade required)
```

### 2. Download and Inspect Installer

```bash
curl -k https://localhost/install > install.sh
cat install.sh | grep -E "BASE_URL|SESSION_TOKEN|kontango enroll"
# Should show environment variables and SDK call
```

### 3. Test Fresh Enrollment

```bash
lxc launch ubuntu:22.04 test-sdk
lxc exec test-sdk -- bash << 'EOF'
  cd /tmp
  curl -k https://localhost/install | sh
EOF

lxc exec test-sdk -- kontango status
# Should show: status=quarantine, stage=0
```

### 4. Test Fingerprint Match

```bash
# Re-enroll same machine (delete cert, keep hardware)
lxc exec test-sdk -- rm ~/.kontango/identity.p12
lxc exec test-sdk -- bash << 'EOF'
  cd /tmp
  curl -k https://localhost/install | sh
EOF

lxc exec test-sdk -- kontango status
# Should show: status=approved (or higher)
# Fingerprint should match previous enrollment
```

### 5. Verify Disclosure Shown

```bash
# Check non-interactive mode shows disclosure
lxc launch ubuntu:22.04 test-disclosure
lxc exec test-disclosure -- bash << 'EOF'
  curl -k https://localhost/install -o install.sh
  chmod +x install.sh
  ./install.sh 2>&1 | grep -A 10 "KONTANGO SDK DISCLOSURE"
EOF
# Should see disclosure before enrollment starts
```

---

## Success Criteria

✅ **SDK ↔ Controller Perfectly Aligned** when:

1. ✅ Installer downloadable from `/install`
2. ✅ Installer embeds BASE_URL and SESSION_TOKEN
3. ✅ Installer calls `kontango enroll` correctly
4. ✅ Enrollment POST to `/api/enroll/stream` succeeds
5. ✅ SSE events streamed back (verify → decision → identity)
6. ✅ WebSocket alt at `/api/ws/enroll` works
7. ✅ Fingerprint recognized on re-enrollment
8. ✅ Trust level changes per decision logic
9. ✅ Certificate + config received and applied
10. ✅ Ziti tunnel connects
11. ✅ ACLs enforced per stage level
12. ✅ Disclosure shown before enrollment
13. ✅ All three platforms work (Linux/macOS/Windows)
14. ✅ Parallel enrollments work
15. ✅ Recovery from dropped connections works

---

## Testing Checklist

- [ ] Endpoints verified
- [ ] Installer download works
- [ ] Fresh machine enrolls → quarantine
- [ ] Returning machine recognized → approved
- [ ] Machine with creds → trusted
- [ ] SSE path works
- [ ] WebSocket path works
- [ ] Verification events logged
- [ ] Decision made correctly
- [ ] Certificate issued and saved
- [ ] Config applied correctly
- [ ] Ziti tunnel connects
- [ ] ACLs enforced
- [ ] Disclosure shown (non-interactive)
- [ ] Disclosure shown (TUI)
- [ ] Session token validated
- [ ] Fingerprint hash matches
- [ ] Multiple enrollments work
- [ ] Parallel enrollments work
- [ ] Failure scenarios handled
- [ ] Linux machines work
- [ ] macOS machines work
- [ ] Windows machines work
- [ ] Performance acceptable (<10s)
- [ ] Logs comprehensive

---

## Known Issues / Notes

| Issue | Status | Notes |
|-------|--------|-------|
| | | |

---

**Last Updated:** April 5, 2026  
**SDK Version:** 1.0  
**Controller Version:** schmutz-controller  
**Status:** ✅ Ready for integration testing

---

Next Step: Follow [TESTING_FLOW.md](TESTING_FLOW.md) to run end-to-end tests
