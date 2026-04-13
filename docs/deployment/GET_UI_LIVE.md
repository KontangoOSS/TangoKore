# ACTION: Get Web UI Live on All Controllers

**Goal:** Every controller serves the enrollment UI + handles /install endpoint

**Status:** UI is built and tested locally. Need to deploy to controllers.

**Timeline:** ~1-2 hours per controller (3 controllers total)

---

## What Needs to Happen

### Current State
- Web UI exists at `TangoKore/web/public/` ✅
- UI is tested and working locally ✅
- Controllers have enrollment API ✅
- Controllers are NOT serving the UI to users ❌

### Desired State
1. User visits `https://controller.example.com/` → sees enrollment web UI
2. User can select skill level (simple vs advanced)
3. User fills out preferences
4. User gets download link for installer script
5. OR user clicks button to get `curl` command

```
https://controller.example.com/
    ↓
[Skill Selection]
    ↓
[Simple Flow] OR [Advanced Flow]
    ↓
[Enrollment Preferences]
    ↓
[Download Installer] OR [Copy curl Command]
    ↓
curl https://controller.example.com/install | sudo sh
```

---

## The Two Options

### Option A: Serve from schmutz-controller (RECOMMENDED)

**What to do:**
1. Copy `TangoKore/web/public/*` → `schmutz-controller/frontend/`
2. Update route handlers in controller
3. Rebuild controller binary
4. Deploy new binary to all 3 controllers

**Pros:**
- ✅ Single deployment (everything together)
- ✅ No extra services to manage
- ✅ UI and API on same host
- ✅ Shared SSL certificate

**Cons:**
- ❌ Requires rebuilding controller
- ❌ Bigger binary size

### Option B: Serve from Separate Service

**What to do:**
1. Deploy `TangoKore/web/public/` as separate HTTP service
2. Configure Caddy to route `/` and `/join` to this service
3. Keep `/install` and `/api/*` on controller

**Pros:**
- ✅ Can update UI without rebuilding controller
- ✅ Lighter controller binary
- ✅ Independent scaling

**Cons:**
- ❌ Extra service to manage
- ❌ Possible CORS issues
- ❌ More complex deployment

---

## Implementation (Option A - Recommended)

### Step 1: Prepare Files in schmutz-controller

```bash
# On your development machine
cd ~/git/kore/schmutz-controller/frontend

# Backup old files (just in case)
mv join-index.html join-index.html.bak
mv guide.html guide.html.bak

# Create directories if needed
mkdir -p css
mkdir -p js

# Copy new files from TangoKore
cp ~/git/kore/TangoKore/web/public/index.html ./join-index.html
cp ~/git/kore/TangoKore/web/public/css/style.css ./css/style.css
cp ~/git/kore/TangoKore/web/public/js/main.js ./js/main.js

# Verify
ls -la frontend/
```

### Step 2: Update Route Handlers

In `src/internal/controller/routes.go` (or wherever routes are defined):

**Current (old):**
```go
http.HandleFunc("/join", serveOldUI)
```

**New:**
```go
// Serve new join webpage
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path == "/" || r.URL.Path == "/join" {
        // Serve frontend/join-index.html with proper MIME type
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        http.ServeFile(w, r, "frontend/join-index.html")
        return
    }
    // 404 for other paths
    http.NotFound(w, r)
})

// Serve static assets (CSS, JS)
http.HandleFunc("/css/", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/css; charset=utf-8")
    http.ServeFile(w, r, filepath.Join("frontend", r.URL.Path))
})

http.HandleFunc("/js/", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
    http.ServeFile(w, r, filepath.Join("frontend", r.URL.Path))
})
```

### Step 3: Update Dockerfile (if needed)

Ensure Dockerfile includes the frontend files:

```dockerfile
# Copy frontend files into image
COPY frontend/ /app/frontend/
```

### Step 4: Build Locally

```bash
cd ~/git/kore/schmutz-controller
make build

# Test locally
./build/tango-controller

# Visit http://localhost:8080/ in browser
# Should see new enrollment UI
```

### Step 5: Deploy to Controllers

For each controller (ctrl-1, ctrl-2, ctrl-3):

```bash
# SSH to controller
ssh root@<controller-ip>

# Stop old service
systemctl stop tango-controller

# Backup old binary
cp /opt/kontango/tango-controller /opt/kontango/tango-controller.bak

# Copy new binary (from your build machine)
scp ~/git/kore/schmutz-controller/build/tango-controller \
    root@<controller-ip>:/opt/kontango/

# Also copy frontend files if not in binary
scp -r ~/git/kore/schmutz-controller/frontend/ \
    root@<controller-ip>:/opt/kontango/

# Start service
systemctl start tango-controller
systemctl status tango-controller

# Verify it's working
curl http://localhost:8080/ | grep -c "Welcome to TangoKore"
# Should output: 1
```

### Step 6: Verify on Public Endpoint

```bash
# Test public endpoint
curl https://controller.example.com/ | grep -c "Welcome to TangoKore"
# Should output: 1

# Test installer endpoint
curl https://controller.example.com/install | head -3
# Should return: #!/bin/bash

# Test CSS loads
curl https://controller.example.com/css/style.css | head -5
# Should return: CSS content

# Test JavaScript loads
curl https://controller.example.com/js/main.js | head -5
# Should return: JavaScript content
```

---

## Testing End-to-End

### From User Perspective

```bash
# 1. Visit web UI
curl https://controller.example.com/ | head -20
# Should see HTML with "Welcome to TangoKore"

# 2. Get installer script
curl https://controller.example.com/install | head -10
# Should return:
# #!/bin/bash
# export BASE_URL="https://controller.example.com"
# export SESSION_TOKEN="sess_..."
# kontango enroll $BASE_URL --session $SESSION_TOKEN --no-tui

# 3. Run installer (on test machine)
curl https://controller.example.com/install | sudo sh

# 4. Check status
kontango status
# Should show: status=enrolled, stage=0
```

### Visual Test in Browser

1. **Open** `https://controller.example.com/`
2. **See** skill-level selection (👤 Simple vs ⚙️ Advanced)
3. **Click** "Just Get Me Connected"
4. **Fill in** machine name (or leave empty)
5. **Check** optional fields
6. **Click** confirm checkbox
7. **Click** "Continue to Installation"
8. **Watch** enrollment progress with live logs
9. **See** success screen with machine ID

---

## Rollback Plan

If something breaks:

```bash
# SSH to controller
ssh root@<controller-ip>

# Restore old binary
cp /opt/kontango/tango-controller.bak /opt/kontango/tango-controller

# Restart
systemctl restart tango-controller

# Verify old UI works
curl http://localhost:8080/join | grep "Tango"
```

---

## Deployment Checklist

### Before Deployment
- [ ] Files copied to schmutz-controller/frontend/
- [ ] Route handlers updated
- [ ] Dockerfile updated (if needed)
- [ ] Builds successfully locally
- [ ] Works on localhost:8080

### During Deployment (Per Controller)
- [ ] SSH to controller
- [ ] Stop service
- [ ] Backup old binary
- [ ] Copy new binary
- [ ] Copy frontend files
- [ ] Start service
- [ ] Verify service running
- [ ] Test locally (curl http://localhost)

### After Deployment
- [ ] Test public endpoint (`https://controller.example.com/`)
- [ ] HTML loads
- [ ] CSS loads
- [ ] JavaScript loads
- [ ] Both skill flows work
- [ ] Installer script works
- [ ] Create test machine
- [ ] Enroll via web UI
- [ ] Check kontango status
- [ ] Verify machine in quarantine

---

## Success Criteria

✅ `https://controller.example.com/` returns HTML (enrollment UI)  
✅ Skill-level selection visible and functional  
✅ Simple flow works end-to-end  
✅ Advanced flow works end-to-end  
✅ Both flows produce identical payloads  
✅ CSS and JS load without errors  
✅ `https://controller.example.com/install` returns shell script  
✅ Users can run `curl ... | sudo sh` and enroll  
✅ Enrolled machines appear with correct stage level  
✅ All 3 controllers serving same UI  

---

## Timeline

- **Prepare files:** 10 minutes
- **Update routes:** 15 minutes
- **Test locally:** 10 minutes
- **Deploy to 3 controllers:** 30 minutes (10 min each)
- **Verify all 3:** 10 minutes
- **Total:** ~1.5 hours

---

## What This Achieves

Once UI is live on all controllers:

**Users can:**
1. Visit enrollment webpage
2. Select skill level
3. Answer questions about their machine
4. Get installer command
5. Run one curl command
6. Be enrolled in mesh

**All via:**
- Web UI (friendly, guided)
- Curl installer (fast, automated)
- CLI (advanced, full control)

**Same outcome:** Machine enrolled, in quarantine, ready to escalate trust

---

## The Complete Flow Then

```
User visits controller.example.com
    ↓
Sees enrollment UI (web)
    ↓
Simple: Chooses name + fields + confirms
Advanced: Reviews exact payload + toggles fields
    ↓
Gets installer command or download link
    ↓
curl https://controller.example.com/install | sudo sh
    ↓
SDK shows disclosure, collects fingerprint
    ↓
Sends to controller via /api/enroll/stream
    ↓
Controller verifies fingerprint, makes decision
    ↓
SDK receives identity, starts tunnel
    ↓
Machine in quarantine, ready to use
```

---

## Next: Make It Happen

1. **Copy files** → Update routes → Test locally
2. **Deploy** → All 3 controllers
3. **Verify** → UI and curl commands working
4. **Done** → Users can enroll via web + CLI

This is the final piece to make the system fully functional.
