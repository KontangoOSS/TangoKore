# Kontango Enrollment Design

## Principle: Server Determines Method

**All clients send identical enrollment messages to the same endpoint.**

The server determines what method applies (new/scan/trusted) based on what it receives and finds in its database.

---

## The Enrollment Flow

### Step 1: Client Announces (Same for Everyone)

```
POST /api/enroll/stream

{
  "hostname": "leonardo-da-pc",
  "os": "linux",
  "os_version": "Ubuntu 25.04",
  "arch": "amd64",
  "kernel": "6.14.0-37-generic",
  "hardware_hash": "56dd0f732bc62988",
  "cpu_cores": 4,
  "memory_mb": 16384,
  "mac_addrs": ["08:00:27:00:00:00"],
  
  // Optional: credentials to prove identity
  "role_id": "vault-role-id",      // if AppRole pre-provisioned
  "secret_id": "vault-secret-id",  // if AppRole pre-provisioned
  "session": "one-time-token",     // if invited via session
  
  // Optional: profile preference
  "profile": "stage-1"             // preferred ACL level
}
```

**That's it.** One message format. No method flag.

### Step 2: Server Receives and Analyzes

Server sees the machine announcement and decides:

**Is this machine trusted?** (Has credentials)
- If role_id/secret_id provided → validate against OpenBao
- If session provided → check if valid token
- If credentials are valid → machine is TRUSTED

**Is this machine known?** (Fingerprint match)
- Hash the hardware data
- Look up by hardware_hash in database
- If found → machine is KNOWN (returning)
- If not found → machine is NEW

**Decision Table:**

| Has Credentials? | Fingerprint Match? | Status | Profile | Source |
|--|--|--|--|--|
| Yes | Either | `trusted` | `stage-3` | Credentials validate it |
| No | Yes | `approved` | Restored ACL | Previous enrollment |
| No | No | `quarantine` | `stage-0` | Unknown, new machine |

### Step 3: Server Sends Verification Checks

Regardless of which path, server streams events:

```
event: verify
data: {"check":"fingerprint_match","passed":false,"confidence":"unknown"}

event: verify
data: {"check":"os_validation","passed":true,"confidence":"high"}

event: verify
data: {"check":"banned_check","passed":true,"confidence":"high"}

event: decision
data: {"status":"quarantine","reason":"new machine"}

event: identity
data: {
  "id":"leonardo-da-pc-new",
  "nickname":"leonardo-da-pc",
  "status":"quarantine",
  "identity": {...pkcs12 cert...},
  "config": {
    "hosts": ["ziti.example.com"],
    "tunnel": {"endpoint":"ws://ziti.example.com:3021/edge","acl":"stage-0"}
  }
}
```

### Step 4: Client Receives and Applies

Client saves the identity and config:

```
✓ Enrollment complete
  • Machine ID: leonardo-da-pc-new
  • Status: quarantine
  • ACL: stage-0 (read-only, limited services)
  
Next: Start Ziti tunnel → Connect to mesh → Start agent
```

---

## Three Enrollment Paths (All Same Entry Point)

### Path 1: New Machine (No History, No Credentials)

```
Client sends:
  - Hostname, OS, arch, fingerprint
  - No credentials
  - No profile preference

Server checks:
  1. Fingerprint → NOT FOUND (new machine)
  2. Credentials → NONE
  
Result:
  Status: quarantine
  Profile: stage-0 (read-only)
  ACL: #quarantine-services only
```

### Path 2: Returning Machine (Known Fingerprint, No Credentials)

```
Client sends:
  - Hostname, OS, arch, fingerprint
  - No credentials
  - No profile preference

Server checks:
  1. Fingerprint → FOUND (in database)
  2. Credentials → NONE
  
Result:
  Status: approved
  Profile: RESTORED from previous enrollment
  ACL: Same as before (what was assigned last time)
```

### Path 3: Trusted Machine (Valid Credentials)

```
Client sends:
  - Hostname, OS, arch, fingerprint
  - role_id + secret_id (AppRole)
  - OR: session token
  - OR: JWT token
  
Server checks:
  1. Credentials → VALID (validates against OpenBao/auth provider)
  2. (Fingerprint not needed, credentials are sufficient)
  
Result:
  Status: trusted
  Profile: stage-3 (full access)
  ACL: All services, full permissions
```

---

## Key Principle

**All clients send the same message.** Server determines what method applies.

- ✅ Machines always send identical announcements
- ✅ Server fingerprint-matches automatically (no flag needed)
- ✅ Server validates credentials if provided
- ✅ Server makes the trust/status decision
- ✅ Client logic is simpler
- ✅ Policy can change without client updates
- ✅ Security is stronger (no client-side method claims)
