# Web Enrollment Interface

## Overview

The join webpage (`web/public/`) provides a web-based enrollment interface that mirrors the SDK's decision tree. It's meant to run at `controller.example.com` or any HTTP server.

**Philosophy:** Your path is what you choose. Same decision tree, different UI.

## The Decision Tree

Users make three choices:

### 1. Machine Identity
- **Option A:** Auto-generate (recommended for most users)
- **Option B:** Provide custom ID (for operators, special cases)

### 2. Optional Fingerprint Fields
Beyond the required minimum (OS, architecture, machine ID), users can opt in to:
- Hostname
- OS version
- Kernel version
- CPU info
- Memory
- MAC addresses
- System UUID
- Serial number

Each field is optional. Users see what it is, why we want it, and can choose to include it.

### 3. Credentials (Optional)
If pre-provisioned with AppRole credentials, users can optionally provide:
- Role ID
- Secret ID

This grants immediate trusted access (stage-3) instead of starting in quarantine.

## User Paths

### Simple Path (Non-Technical Users)

**Entry:** "Just Get Me Connected"

1. **Give Your Machine a Name**
   - Plain language: "laptop", "server-1", etc.
   - Or leave empty for auto-generated

2. **What We Need**
   - Required fields listed clearly
   - Optional fields with checkboxes
   - Each has a "Why?" explanation
   - Privacy assurance message

3. **Confirmation**
   - Single checkbox: "I understand and I'm ready"
   - Clear, non-threatening language

### Advanced Path (Developers/Operators)

**Entry:** "Show Me Everything"

1. **Machine Identity**
   - Custom ID input or auto-generate
   - Shows format and validation

2. **Fingerprint Configuration**
   - Live JSON preview of payload
   - Toggle each optional field
   - Technical field names shown
   - Data sources explained

3. **Optional Credentials**
   - Role ID / Secret ID inputs
   - Explains AppRole authentication

## Both Paths → Same SDK

```
Simple UI → Build Payload → POST /api/enroll/stream → Identity
Advanced UI → Build Payload → POST /api/enroll/stream → Identity
CLI TUI → Build Payload → POST /api/enroll/stream → Identity
CLI --no-tui → Build Payload → POST /api/enroll/stream → Identity
```

**Everything converges at the same SDK enrollment logic.**

## Files

```
web/
├── public/
│   ├── index.html       # Skill selection + both flows
│   ├── css/
│   │   └── style.css    # Material Design styling
│   └── js/
│       └── main.js      # Decision tree logic
└── README.md            # Customization guide
```

## Running Locally

```bash
cd web/public
python3 -m http.server 8000
# Visit http://localhost:8000
```

## API Integration

The webpage expects these endpoints:

**POST /api/enroll/stream**
- Accepts the enrollment payload
- Returns SSE events: verify → decision → identity

The page handles:
- Building the payload from user choices
- Sending to the API
- Displaying live logs
- Handling success/error states

## Customization

Edit these for your deployment:

**Colors:** `css/style.css` — CSS variables at `:root`
**Branding:** `index.html` — Header content
**API URL:** `js/main.js` — `CONFIG.enrollmentAPI`
**Messages:** `index.html` — User-facing text

## What Gets Sent

### Required (Always)
```json
{
  "os": "linux|darwin|windows",
  "arch": "amd64|arm64|...",
  "issued_id": "your-machine-id"
}
```

### Optional (User's Choice)
User's checked fields are added:
```json
{
  "hostname": "...",
  "os_version": "...",
  "cpu_info": "...",
  ...
}
```

### Credentials (If Provided)
```json
{
  "role_id": "...",
  "secret_id": "..."
}
```

## Design Philosophy

1. **Respect User Choice**
   - Non-technical users aren't less important
   - Advanced users get full control
   - Both get the same quality

2. **Transparency Without Overwhelming**
   - Show what's being sent
   - Explain why each field matters
   - Don't hide complexity, just present it kindly

3. **Privacy-First**
   - Never sell data
   - Users can delete anytime
   - Can run own server
   - Code is open source

4. **One Experience, Different UX**
   - Same enrollment logic
   - Same choices available
   - Just different interface

## Next: CLI/TUI Alignment

The CLI and TUI should mirror this same decision tree:

**CLI (`--no-tui` mode):**
- Show disclosure automatically
- Ask for machine ID (or auto)
- Ask which optional fields to include
- Show payload before sending
- Pause for confirmation

**CLI/TUI (interactive mode):**
- Interactive dialogs
- Same questions as web
- Show disclosure
- Full payload transparency
- Same decisions, just in terminal

## Testing

```bash
# Test simple flow
1. Click "Just Get Me Connected"
2. Leave machine name empty
3. Check optional fields
4. Confirm and enroll

# Test advanced flow
1. Click "Show Me Everything"
2. Enter custom machine ID
3. Toggle optional fields (watch payload update)
4. Add credentials if you have them
5. Generate and enroll

# Both should:
- Show live logs
- Display success with machine ID
- Show next steps clearly
```

---

**Philosophy:** "You are your password. Keep being you."

And: **"We understand you, respect your privacy, and want you to feel safe."**
