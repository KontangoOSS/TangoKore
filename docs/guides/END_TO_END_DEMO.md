# End-to-End Demo: TangoKore Enrollment

## Quick Start

See the web UI in action RIGHT NOW:

```bash
cd ~/git/kore/TangoKore/web/public
python3 -m http.server 8000
```

Then visit: **http://localhost:8000/**

---

## What You'll See

### Step 1: Skill Level Selection

Two options appear:
- **👤 Just Get Me Connected** — For beginners (simple flow)
- **⚙️ Show Me Everything** — For developers (advanced flow)

Click either one.

---

## Path A: Simple Flow (Beginner)

### Step 1: Give Your Machine a Name

```
Input: Machine name (optional)
Examples: "laptop", "server-1", "pi-home"
Auto-generated: machine-a1b2c3d4 (if left empty)
```

Your machine name displays: `machine-a1b2c3d4`

### Step 2: What We Need

Shows:
- **Required (Always Sent)**
  - Operating System
  - Architecture
  - Machine ID

- **Optional (Choose What to Send)**
  - ☑ Hostname
  - ☑ OS Version
  - ☑ CPU Info
  - ☐ Memory
  - ☐ Network Interfaces

Each has a **"Why?"** box explaining why we want it.

### Step 3: Privacy Assurance

✓ We never sell your data  
✓ You can delete everything anytime  
✓ Want it private? Run your own server  
✓ All code is open source

### Step 4: Confirm

Checkbox: "I understand what's being shared and I'm ready to connect"

Button: "Continue to Installation" (enabled when checkbox is checked)

### Step 5: Enrollment Progress

See live logs:
```
[14:23:15] Initializing enrollment...
[14:23:15] Method: new
[14:23:15] Machine ID: machine-a1b2c3d4
[14:23:16] Scanning machine information...
[14:23:17] Operating System: linux
[14:23:17] Architecture: amd64
[14:23:17] Machine ID: machine-a1b2c3d4
[14:23:18] Building enrollment payload...
[14:23:19] Connecting to controller...
[14:23:20] Sending enrollment request...
[14:23:21] Verification: fingerprint recognized [success]
[14:23:22] Decision: approved [success]
[14:23:23] Enrollment complete!
```

### Step 6: Success

Shows:
```
✓ You're Connected!

Your Machine Identity
Machine ID: machine-a1b2c3d4
Status: enrolled
Trust Level: stage-0 (quarantine)

What's Next?
1. Download the TangoKore CLI for your machine
2. Run the installer: curl ... | sh
3. Your machine connects to the mesh automatically
```

---

## Path B: Advanced Flow (Developer)

### Step 1: Machine Identity

Input field: Custom machine ID or auto-generate

Shows: `auto-generated if empty`

### Step 2: Fingerprint Configuration

**Live JSON Payload Preview:**
```json
{
  "os": "linux",
  "arch": "amd64",
  "issued_id": "auto-generated",
  "hostname": "...",
  "os_version": "...",
  ...
}
```

**Optional Fields (Technical Names):**
- ☑ `hostname` — System hostname
- ☑ `os_version` — Release/build number
- ☑ `kernel_version` — Kernel release
- ☐ `machine_uuid` — System UUID/BIOS ID
- ☑ `cpu_info` — Processor model and count
- ☑ `memory_mb` — RAM in megabytes
- ☐ `mac_addrs` — Network interface MACs
- ☐ `serial_number` — Hardware serial

**Payload updates in real-time** as you toggle fields.

### Step 3: Credentials (Optional)

Input fields:
- Role ID: `[optional]`
- Secret ID: `[optional]`

If you have AppRole credentials, enter them. If not, skip.

### Step 4: Generate Enrollment Payload

Button: "Generate Enrollment Payload"

Same enrollment flow as simple path, but you controlled exactly what goes in.

---

## Both Paths Lead to Same Result

```
Simple Flow    }
Advanced Flow  } → Same Enrollment Payload → Same SDK → Same Result
```

Both produce identical machine enrollment with identical trust level assignment.

---

## The Real End-to-End (When Join Endpoint Is Online)

### What Users See

**Step 1: Download Installer**
```bash
curl https://ctrl.konoss.org/install | head -3
```

Returns:
```bash
#!/bin/bash
export BASE_URL="https://ctrl.konoss.org"
export SESSION_TOKEN="sess_a1b2c3d4e5f6g7h8"
```

**Step 2: Run Installer**
```bash
curl https://ctrl.konoss.org/install | sudo sh
```

What happens:
1. Script downloads
2. Script sets BASE_URL and SESSION_TOKEN
3. Script calls: `kontango enroll $BASE_URL --session $SESSION_TOKEN --no-tui`
4. SDK shows disclosure
5. SDK collects fingerprint
6. SDK sends to controller
7. Machine gets identity and config

**Step 3: Verify**
```bash
kontango status
```

Shows:
```
Machine ID: machine-a1b2c3d4
Status: enrolled
Trust Level: stage-0 (quarantine)
Connected: yes
```

---

## Testing Locally

### Option 1: Just See the UI (RIGHT NOW)

```bash
cd ~/git/kore/TangoKore/web/public
python3 -m http.server 8000
# Visit http://localhost:8000/
```

Click through the flows. See how decisions work. No backend needed.

### Option 2: Test with Real SDK (When Controller Is Up)

```bash
# Build SDK
cd ~/git/kore/TangoKore
make build

# Test enrollment against local controller
./build/kontango enroll https://localhost:9090 --no-tui

# Or with TUI
./build/kontango enroll https://localhost:9090

# Or with custom ID
./build/kontango enroll https://localhost:9090 --no-tui --machine-id "my-laptop"
```

### Option 3: Full Flow (When Join Endpoint Is Restored)

```bash
# Test installer script
curl -k https://ctrl.konoss.org/install | head -10

# Run full installation
curl -k https://ctrl.konoss.org/install | sudo sh

# Check status
kontango status
```

---

## The Key Insight

**Same decision tree, three entry points:**

1. **Web UI** (http://localhost:8000/)
   - Choose machine name
   - Choose optional fields
   - Confirm and enroll
   
2. **CLI Non-Interactive** (curl installer)
   - Show disclosure
   - Collect fingerprint
   - Send and enroll

3. **CLI TUI** (kontango command)
   - Interactive dialogs
   - Same choices as web
   - Same enrollment outcome

**All three paths:**
- Make identical decisions
- Generate identical payloads
- Produce identical results
- Get identical trust levels

---

## What's Actually Happening

### In the Web UI
1. JavaScript collects your choice (identity, fields, credentials)
2. Builds the exact payload the SDK expects
3. Shows it to you
4. Would send to `POST /api/enroll/stream` (SSE endpoint)

### In the CLI
1. SDK accepts command-line flags OR reads TUI input
2. Builds the exact same payload
3. Shows disclosure
4. Sends to `POST /api/enroll/stream` (SSE endpoint)

### On the Controller
1. Receives payload via SSE
2. Verifies fingerprint (is this machine known?)
3. Decides trust level (new/returning/trusted/admin)
4. Sends back identity (certificate + config)

### On the Machine
1. Receives and saves identity
2. Starts Ziti tunnel
3. Connects to mesh
4. Ready to use

---

## Right Now: Test the UI

**The web UI is LIVE and WORKING:**

```bash
# It's already running if you followed the quick start
curl http://localhost:8000/ | grep -c "Welcome to TangoKore"
# Output: 1 (page loaded)
```

**Visit in browser:** `http://localhost:8000/`

**Try both flows:**
1. Click "Just Get Me Connected" → simple flow
2. Go back, click "Show Me Everything" → advanced flow

**Watch what happens:**
- UI guides you through choices
- Live payload preview (advanced mode)
- Shows what data will be sent
- Simulates enrollment with fake logs

---

## When Join Endpoint Is Restored

Everything cascades:

1. **Web** — Users visit `https://ctrl.konoss.org/` → see new UI
2. **Installer** — `curl https://ctrl.konoss.org/install | sh` works
3. **CLI** — `kontango enroll` connects to controller
4. **Complete** — Full enrollment flow works end-to-end

---

## Summary

**You can see the UI RIGHT NOW:**

```
http://localhost:8000/
```

**The complete flow works once:**
1. Join endpoint is online ⚠️ (currently down)
2. Web UI is integrated into controller 🔄 (ready to integrate)
3. Controller is running and accessible ✓

**Everything is built and ready. Just needs to be assembled and deployed.**

---

## Next: Get This Online

1. **Diagnose join endpoint** — Why is it offline?
2. **Restore service** — Get `https://ctrl.konoss.org/` responding
3. **Integrate web UI** — Copy files to controller
4. **Test end-to-end** — Users can enroll via web, CLI, or both

Then: **The system is live and users can join the mesh.**
