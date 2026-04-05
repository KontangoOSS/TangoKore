# TangoKore: Architecture Clarification

## The System Architecture

TangoKore is **not just an SDK**. It's the complete zero-trust enrollment system.

```
┌─────────────────────────────────────────────────────────────┐
│                     TangoKore                               │
│  (User-facing SDK + Documentation + UI guidelines)          │
└──────────────────┬──────────────────────────────────────────┘
                   │
                   ├─ README.md (how to use)
                   ├─ docs/ (technical reference)
                   ├─ SDK (CLI + agent)
                   └─ UI Patterns (web/public/)
                   
┌──────────────────────────────────────────────────────────────┐
│                  schmutz-controller                          │
│  (Enrollment API server - runs on every controller)          │
├──────────────────────────────────────────────────────────────┤
│  ✓ /install — Serves installer script                       │
│  ✓ /api/enroll/stream — SSE enrollment                     │
│  ✓ /api/ws/enroll — WebSocket enrollment                   │
│  ✓ /join — Web UI for enrollment (NEW)                     │
│  ✓ /guide — Help/documentation                             │
│  ✓ Authentication & disclosure                             │
└──────────────────────────────────────────────────────────────┘
                   │
                   ├─ Ziti (mesh networking)
                   ├─ OpenBao (secrets management)
                   └─ Caddy (SSL/routing)
                   
┌──────────────────────────────────────────────────────────────┐
│                    schmutz                                   │
│  (Edge gateway - L4 firewall in the network)                │
└──────────────────────────────────────────────────────────────┘
```

## Where Things Live

### TangoKore Repository (`~/git/kore/TangoKore/`)

**Purpose:** User-facing SDK and documentation

**Contents:**
- `README.md` — Simple, beginner-friendly overview
- `docs/` — Technical reference (MkDocs)
- `cmd/kontango/` — CLI binary and TUI
- `internal/enroll/` — Enrollment library
- `tests/` — Test suite
- `web/public/` — **UI patterns for controllers**

**What the web/ folder is:**
- Template/reference UI for controllers to serve
- Shows decision tree pattern: simple vs advanced flow
- Material Design styling
- Can be embedded in controllers OR run standalone

### schmutz-controller Repository (`~/git/kore/schmutz-controller/`)

**Purpose:** The actual enrollment API server

**Current Contents:**
- `src/internal/controller/enroll/` — Enrollment endpoints
- `frontend/` — Web interface (currently: `join-index.html`, `guide.html`)
- Routes & handlers for `/install`, `/api/enroll/*`, etc.

**What should be here:**
- The web UI files FROM TangoKore
- Served at `/` (or `/join`) for users
- Every controller instance serves this

### schmutz Repository (`~/git/kore/schmutz/`)

**Purpose:** Edge gateway (L4 firewall)

**What it does:**
- Sits between untrusted network and mesh
- Enforces quarantine stage (stage-0) rules
- No enrollment logic needed

## The Real Issue

**The web UI should be:**
1. Designed in TangoKore (`web/public/`)
2. **Built and served by schmutz-controller**
3. Available at every controller's `https://controller/` or `https://controller/join`
4. Users can access via:
   - `https://join.kontango.net/` (load-balanced across controllers)
   - Their own controller: `https://their-controller.example/`

**Currently:**
- TangoKore has the UI design (just created)
- schmutz-controller already has `frontend/join-index.html` (old design)
- Controllers are NOT serving it to users (unknown why - service issue?)

## What Needs to Happen

### Immediate (This Week)

1. **Understand why controllers aren't serving the UI**
   - Check if service is running
   - Check Caddy routing
   - Check if frontend files are present

2. **Get `https://join.kontango.net/` responding**
   - Diagnose and fix schmutz-controller service
   - Verify frontend is being served
   - Test /install endpoint works

### Short Term (This Sprint)

3. **Update schmutz-controller's frontend**
   - Replace `join-index.html` with new design from TangoKore
   - Copy decision tree UI pattern
   - Integrate with existing handlers

4. **Ensure every controller serves it**
   - Update deployment to include web files
   - Route `/` or `/join` to frontend
   - Test on all 3 controllers

### Medium Term (Next Sprint)

5. **Make it customizable**
   - Allow operators to customize branding
   - Support different themes/styles
   - Keep decision tree consistent

## The Decision Tree Pattern

All entry points should offer the same choices:

**For Users:**
1. **Simple Path** — "Just get me connected"
   - Auto-generated machine ID
   - Checkbox: choose optional fields
   - Confirm and enroll

2. **Advanced Path** — "Show me everything"
   - Custom machine ID
   - Live payload preview
   - Toggle optional fields
   - Credentials input

**Both paths → Same payload → Same SDK → Same enrollment**

## Files & Locations

```
TangoKore/web/public/
├── index.html        ← Use this as basis for schmutz-controller UI
├── css/style.css     ← Material Design styling
└── js/main.js        ← Decision tree logic

schmutz-controller/frontend/
├── join-index.html   ← Replace with TangoKore version
├── guide.html        ← Keep or update
└── [CSS/JS files]    ← Add Material Design
```

## Next Steps for Engineers

1. **Understand the codebase:**
   - Read `schmutz-controller/src/internal/controller/enroll/`
   - See how it routes `/install` and frontend files

2. **Integrate TangoKore UI:**
   - Copy decision tree pattern from `web/public/index.html`
   - Update `schmutz-controller/frontend/join-index.html`
   - Ensure same HTML/CSS/JS pattern

3. **Test on controllers:**
   - Deploy updated controller
   - Visit `https://controller-ip/join`
   - Verify both simple and advanced flows work

4. **Fix serving issue:**
   - Why isn't `https://join.kontango.net/` responding?
   - Service down? Port issue? Routing?

## Philosophy

**Every controller is a doorway.**

Whether users join via:
- `https://join.kontango.net/` (public master)
- Their own controller
- A custom domain
- CLI, TUI, or web

They get the **same experience, same decision tree, same quality**.

The web UI should be **embedded in every deployment**, not external.

---

**Key Insight:** TangoKore is the *specification* and *SDK*. schmutz-controller is the *implementation*. The UI belongs in both places — the pattern in TangoKore, the actual served instance in schmutz-controller.
