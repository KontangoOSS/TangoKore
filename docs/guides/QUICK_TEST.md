# Quick Test: See TangoKore Enrollment UI in Action

## Right Now: View the Web UI

### Step 1: Start Web Server

```bash
cd ~/git/kore/TangoKore/web/public
python3 -m http.server 8000
```

You should see:
```
Serving HTTP on 0.0.0.0 port 8000 (http://0.0.0.0:8000/) ...
```

### Step 2: Open in Browser

Open your browser to: **http://localhost:8000/**

You'll see:
```
┌─────────────────────────────────────────┐
│         Welcome to TangoKore            │
│     Connect your machine. Stay yourself. │
├─────────────────────────────────────────┤
│  How would you like to get started?     │
│                                         │
│  ┌──────────────────┐  ┌──────────────┐│
│  │  👤 Just Get Me  │  │  ⚙️ Show Me  ││
│  │    Connected     │  │  Everything  ││
│  │                  │  │              ││
│  │ I'm new to this  │  │ I want to    ││
│  │ Walk me through  │  │ see and      ││
│  │ step-by-step     │  │ control      ││
│  │                  │  │ exactly      ││
│  │ Beginner-       │  │ Beginner-    ││
│  │ friendly        │  │ friendly     ││
│  └──────────────────┘  └──────────────┘│
└─────────────────────────────────────────┘
```

---

## Path 1: Simple Flow (Click "Just Get Me Connected")

### Screen 1: Give Your Machine a Name

```
┌──────────────────────────────────────────┐
│  Let's Connect Your Machine              │
├──────────────────────────────────────────┤
│  Step 1: Give Your Machine a Name        │
│  This helps you remember which machine   │
│  this is.                                │
│                                          │
│  Machine Name (optional)                 │
│  Examples: "laptop", "server-1", "pi-   │
│  [_________________________]             │
│                                          │
│  Your machine: machine-7a3k9m2q          │
│                                          │
│  Step 2: What We Need                    │
│  Just basic info so we recognize your    │
│  machine again.                          │
│                                          │
│  🔐 Always Sent (Required)               │
│  These help us identify your machine:    │
│  • Operating System                      │
│  • Architecture                          │
│  • Machine ID                            │
│                                          │
│  Why? So we know how to help your        │
│  machine and recognize it if it comes    │
│  back.                                   │
│                                          │
│  📊 Extra Details (Optional)             │
│  Help us know your machine better.       │
│  ☑ Hostname                              │
│  ☑ OS Version                            │
│  ☑ CPU Info                              │
│  ☐ Memory                                │
│  ☐ Network Interfaces                    │
│                                          │
│  Your Privacy Matters to Us              │
│  ✓ We never sell your data               │
│  ✓ You can delete everything anytime     │
│  ✓ Want it private? Run your own server  │
│  ✓ All code is open source               │
│                                          │
│  ☐ I understand what's being shared      │
│    and I'm ready to connect              │
│                                          │
│  [Continue to Installation] (disabled)   │
└──────────────────────────────────────────┘
```

**What to do:**
1. Leave name empty (auto-generates) OR type a name
2. Checkboxes already selected by default
3. Check the confirmation box
4. Click "Continue to Installation"

### Screen 2: Enrollment in Progress

```
┌──────────────────────────────────────────┐
│  Enrollment in Progress                  │
├──────────────────────────────────────────┤
│  ████████████████░░░░░░░░░░░░░░░░ 50%    │
│  Connecting to controller...             │
│                                          │
│  Live Activity                           │
│  [14:23:15] Initializing enrollment...  │
│  [14:23:15] Method: new                 │
│  [14:23:15] Machine ID: machine-7a3k9m │
│  [14:23:16] Scanning machine info...    │
│  [14:23:17] Operating System: linux     │
│  [14:23:17] Architecture: amd64         │
│  [14:23:18] Building enrollment...      │
│  [14:23:19] Connecting to controller... │
│  [14:23:20] Sending request...          │
│  [14:23:21] Verification: OK ✓          │
│  [14:23:22] Decision: approved ✓        │
│  [14:23:23] Enrollment complete! ✓      │
└──────────────────────────────────────────┘
```

Watch the logs show each step. Progress bar fills up.

### Screen 3: Success!

```
┌──────────────────────────────────────────┐
│            ✓ You're Connected!           │
├──────────────────────────────────────────┤
│                                          │
│  Your machine is now registered with     │
│  the mesh network.                       │
│                                          │
│  Your Machine Identity                   │
│  ┌──────────────────────────────────────┐│
│  │ Machine ID: machine-7a3k9m2q         ││
│  │ Status: enrolled                     ││
│  │ Trust Level: stage-0 (quarantine)    ││
│  └──────────────────────────────────────┘│
│                                          │
│  What's Next?                            │
│  1. Download the TangoKore CLI           │
│  2. Run the installer: curl ... | sh     │
│  3. Your machine connects automatically  │
│                                          │
│  [Back to Start]                         │
└──────────────────────────────────────────┘
```

Click "Back to Start" to try the other flow.

---

## Path 2: Advanced Flow (Click "Show Me Everything")

### Screen 1: Machine Identity

```
┌──────────────────────────────────────────┐
│  Enrollment Configuration                │
├──────────────────────────────────────────┤
│  Machine Identity                        │
│  ┌──────────────────────────────────────┐│
│  │ Machine ID (optional)                ││
│  │ Leave blank to auto-generate         ││
│  │ [_________________________]           ││
│  │ Auto-generated if empty: machine-... ││
│  └──────────────────────────────────────┘│
│                                          │
│  Fingerprint Configuration               │
│  Choose what information to send.        │
│  More data = faster trust escalation.    │
│                                          │
│  Minimal Fingerprint (Required)          │
│  Always sent. Cannot be disabled.        │
│  os        Operating system              │
│  arch      Architecture                  │
│  issued_id Machine identifier            │
│                                          │
│  Enhanced Fingerprint (Optional)         │
│  Send additional fields to speed up      │
│  trust decisions.                        │
│  ☑ hostname              System hostname │
│  ☑ os_version            Release/build   │
│  ☑ kernel_version        Kernel release  │
│  ☐ machine_uuid          System UUID     │
│  ☑ cpu_info              Processor info  │
│  ☑ memory_mb             RAM in MB       │
│  ☐ mac_addrs             Network MACs    │
│  ☐ serial_number         Hardware serial │
│                                          │
│  Enrollment Payload Preview              │
│  {                                       │
│    "os": "linux",                        │
│    "arch": "amd64",                      │
│    "issued_id": "auto-generated",        │
│    "hostname": "...",                    │
│    "os_version": "...",                  │
│    ...                                   │
│  }                                       │
│                                          │
│  Credentials (Optional)                  │
│  If you have pre-provisioned             │
│  credentials, enter them here.           │
│  Role ID: [_________________]            │
│  Secret ID: [_________________]          │
│                                          │
│  [Generate Enrollment Payload]           │
└──────────────────────────────────────────┘
```

**What to do:**
1. Leave Machine ID empty (or enter custom)
2. Toggle optional fields on/off
3. Watch payload preview update in real-time
4. Optionally add credentials
5. Click "Generate Enrollment Payload"

Same enrollment progress and success screens follow.

---

## Key Features to Notice

### Simple Flow
✅ **Beginner-friendly language**
- "Give your machine a name"
- "What we need"
- "Why?" explanations for each field

✅ **Sensible defaults**
- Optional fields pre-checked
- Auto-generates machine ID if empty
- Privacy assurance at the end

✅ **Clear next steps**
- What happens after enrollment
- Where to go next

### Advanced Flow
✅ **Technical details**
- Field names match API (`os_version`, `cpu_info`, etc.)
- Shows exact JSON payload
- Real-time payload preview

✅ **Full control**
- Custom machine ID
- Toggle each optional field
- Add credentials manually
- See exactly what will be sent

✅ **For developers**
- Understanding of API structure
- Validation of what goes out
- Debugging-friendly

---

## What's Happening Behind the Scenes

### When You Click "Continue to Installation" or "Generate Enrollment Payload"

1. **JavaScript collects your choices**
   ```javascript
   {
     os: "linux",
     arch: "amd64",
     issued_id: "machine-7a3k9m2q",
     hostname: true,
     os_version: true,
     cpu_info: true,
     memory_mb: false,
     ...
   }
   ```

2. **Build the payload** (same as SDK would build)
   ```json
   {
     "os": "linux",
     "arch": "amd64",
     "issued_id": "machine-7a3k9m2q",
     "hostname": "my-laptop",
     "os_version": "22.04",
     "cpu_info": "Intel Core i7-10700K, 8 cores"
   }
   ```

3. **Show fake logs** (simulating what SDK does)
   ```
   [14:23:15] Sending to /api/enroll/stream
   [14:23:16] Received verify events
   [14:23:17] Received decision
   [14:23:18] Received identity
   [14:23:19] Enrollment complete
   ```

4. **Show success** with your machine ID and trust level

---

## Test Different Scenarios

### Scenario 1: Minimal Data (Simple Flow, Default)
- Auto-generated machine ID
- Only required fields
- Result: Fastest enrollment, baseline trust

### Scenario 2: Maximum Data (Advanced Flow, All Checkboxes)
- Custom machine ID
- All optional fields
- Result: More information for faster trust escalation

### Scenario 3: With Credentials (Advanced Flow)
- Add AppRole role_id + secret_id
- Result: Immediate trusted access (stage-3)

Try all three! The payload changes each time but the enrollment process is the same.

---

## When Real Endpoints Are Online

Once `ctrl.example.com` is restored and the controller is running:

### The Web UI Will:
1. POST payload to `/api/enroll/stream`
2. Receive SSE events (verify, decision, identity)
3. Show real logs of what the controller is doing
4. Save real machine ID and certificate

### Instead of fake logs like:
```
[14:23:15] Initializing enrollment...
```

You'll see real controller logs like:
```
[14:23:15] verify: fingerprint_match = false
[14:23:16] verify: os_validation = true
[14:23:17] decision: status = quarantine
[14:23:18] identity: id = mch_a1b2c3d4e5f6g7h8
```

---

## Right Now: Go Test It

```bash
# Terminal 1: Start web server
cd ~/git/kore/TangoKore/web/public
python3 -m http.server 8000

# Terminal 2 or Browser: Open http://localhost:8000/
```

**Click through both flows.** Notice:
- How decisions look the same
- Payload stays consistent
- Just different UI for different skill levels
- Both paths reach the same outcome

---

## Summary

**You can see the complete UI RIGHT NOW by visiting http://localhost:8000/**

**Two different flows, same enrollment logic:**
- Simple: Beginner clicks buttons, guided through choices
- Advanced: Developer sees payloads, controls every field
- Both: Identical machine enrollment and trust level assignment

**When join endpoint is restored:**
- Replace fake logs with real controller events
- Real machine IDs and certificates
- Real trust level decisions

**The system is complete. Just waiting for the backend to be restored.**
