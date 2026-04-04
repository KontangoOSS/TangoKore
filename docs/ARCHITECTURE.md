# Architecture

## Overview

TangoKore is a composable mesh platform. Each component is independent, versioned separately, and communicates through the OpenZiti overlay network. There is no shared database, no central bus outside the mesh, and no open ports.

## Layers

```
┌─────────────────────────────────────────────────────────┐
│  Applications                                           │
│  ticketarr, konfig, wanda, your apps...                 │
│  Deployed as profiles. Secrets from Bao. Config from git│
├─────────────────────────────────────────────────────────┤
│  Platform                                               │
│  kontango SDK, kontango-controller, schmutz             │
│  Enrollment, deploy, telemetry, edge security           │
├─────────────────────────────────────────────────────────┤
│  Infrastructure                                         │
│  OpenZiti, OpenBao, NATS, Caddy                         │
│  Overlay network, secrets, messaging, routing           │
├─────────────────────────────────────────────────────────┤
│  Machines                                               │
│  Linux, macOS, Windows, containers, bare metal          │
│  Anywhere Go compiles and a service manager runs        │
└─────────────────────────────────────────────────────────┘
```

## Machine Lifecycle

```
bare machine
    │
    ▼
enroll (curl | sh)
    │  ├── collect fingerprint
    │  ├── stream to controller (SSE)
    │  ├── verify pipeline (hostname, OS, ban check, fingerprint match)
    │  ├── decision: quarantine / approved / rejected
    │  └── issue Ziti identity (mTLS cert)
    │
    ▼
unclaimed (quarantine)
    │  ├── tunnel running (overlay connected)
    │  ├── agent running (heartbeating specs)
    │  ├── minimal mesh privileges (#quarantine role)
    │  └── portal showing docs (portal mode)
    │
    ▼
claimed
    │  ├── portal mode: user signs in via OIDC → machine bound to account
    │  └── service mode: controller pushes profile via NATS → machine assigned role
    │
    ▼
running
    │  ├── bao-agent authenticated (AppRole over tunnel)
    │  ├── secrets injected as env vars (nothing on disk)
    │  ├── application started (docker compose / binary / anything)
    │  ├── caddy routing traffic
    │  └── telemetry flowing to controller
    │
    ▼
update (new config, rotated secret, new version)
    │  ├── controller pushes via NATS
    │  ├── agent applies profile
    │  ├── bao-agent re-renders
    │  └── application restarts with new values
    │
    ▼
revoke
       ├── Ziti identity revoked
       ├── Bao AppRole revoked
       └── machine drops off mesh
```

## Network Model

Every connection travels through the OpenZiti overlay. There are no open ports on any machine. The overlay handles:

- **Authentication**: Every connection is mTLS. The identity was issued at enrollment.
- **Authorization**: Ziti service policies control who can dial/bind which services. A quarantined machine can only reach `nats.tango` and `config.tango`.
- **Encryption**: End-to-end. The controller nodes route traffic but can't read it.
- **Discovery**: Services are named (e.g., `ticketarr.tango`), not addressed by IP. DNS is resolved through the tunnel.

## Data Flows

### Telemetry (agent → controller)
```
agent → NATS publish (tango.telemetry.<machineID>) → Ziti overlay → controller NATS (JetStream)
```
- Unidirectional: agent pushes, controller subscribes
- JetStream retains 1 hour on controller, 5 minutes buffered on edge
- Compressed (zlib) JSON with short keys (~150 bytes per heartbeat)

### Config (controller → agent)
```
controller → NATS publish / config.tango TCP → Ziti overlay → agent
```
- Unidirectional: controller pushes, agent applies
- Instruction types: hello, config, apply, set_interval, reload
- Config stays in memory until replaced by next push

### Secrets (Bao → application)
```
bao-agent → Bao API (over Ziti tunnel) → OpenBao cluster
bao-agent → env vars → application process
```
- Agent authenticates with one-time AppRole secret_id
- Secrets rendered as environment variables (never written to disk)
- Auto-rotation: bao-agent detects changes, restarts application

### User traffic (browser → application)
```
browser → schmutz edge gateway → Ziti overlay → caddy on machine → application
```
- Schmutz reads TLS ClientHello, fingerprints with JA4
- Known clients routed through overlay
- Unknown clients see nothing (connection dropped)

## Component Interaction

```
kontango SDK                        kontango-controller
┌──────────────┐                    ┌──────────────────┐
│ enroll       │───enrollment────▶ │ verify pipeline  │
│ agent        │───telemetry─────▶ │ NATS + JetStream │
│ agent        │◀──config push──── │ deploy API       │
│ portal       │   (via NATS)      │ Bao management   │
│ bao-agent    │───secrets───────▶ │ OpenBao cluster  │
│ caddy        │   (via tunnel)    │ Ziti controller  │
│ ziti tunnel  │◀──overlay────────▶│ Ziti routers     │
└──────────────┘                    └──────────────────┘

schmutz (edge gateway)
┌──────────────┐
│ TLS peek     │  Sits in front of public-facing services.
│ JA4 classify │  Reads the handshake. Routes friends.
│ HP system    │  Ghosts strangers. Self-heals under attack.
│ Ziti relay   │
└──────────────┘
```

## Security Model

1. **Zero ports open**: No machine listens on any public port. All connectivity is outbound to Ziti routers.
2. **Identity-based access**: Machines authenticate by certificate, not IP. Service policies define who can reach what.
3. **Quarantine by default**: New machines get minimal privileges. Promotion requires explicit claim.
4. **Secrets never in git**: Templates live in git. Secrets live in OpenBao. They meet at runtime in memory.
5. **One-time credentials**: AppRole secret_ids are single-use, issued at deploy time, burned after first auth.
6. **Edge classification**: Schmutz fingerprints every TLS handshake before routing. Bots and scanners are caught at the hello, before any HTTP traffic flows.
