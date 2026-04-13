# Production Topology

## Overview

The kontango platform runs on a 3-node controller cluster (DigitalOcean) connected to a Proxmox cluster (on-premises) through an OpenZiti overlay mesh. All inter-service communication uses `.tango` DNS names resolved through the Ziti network. Only ports 80 and 443 are exposed to the public internet.

## Cluster Layout

```
                    ┌─────────────────────────────────────┐
                    │         DigitalOcean (Public)        │
                    │                                     │
                    │  ┌─────────┐ ┌─────────┐ ┌─────────┐
                    │  │ ctrl-1  │ │ ctrl-2  │ │ ctrl-3  │
                    │  │ .tango  │ │ .tango  │ │ .tango  │
                    │  └────┬────┘ └────┬────┘ └────┬────┘
                    │       │           │           │     │
                    │       └─────┬─────┘───────────┘     │
                    │             │  Raft consensus        │
                    └─────────────┼───────────────────────┘
                                  │
                          Ziti overlay mesh
                                  │
                    ┌─────────────┼───────────────────────┐
                    │             │    Proxmox (LAN)       │
                    │  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐
                    │  │ hank │ │ pve  │ │slim1 │ │slim2 │
                    │  │.tango│ │.tango│ │.tango│ │.tango│
                    │  └──┬───┘ └──┬───┘ └──┬───┘ └──┬───┘
                    │     │        │        │        │    │
                    │     └────────┴────────┴────────┘    │
                    │          10.11.30.0/16 LAN          │
                    │     LXC containers, VMs, services   │
                    └─────────────────────────────────────┘
```

## Controller Nodes

Each controller runs identical services:

| Service | Systemd Unit | Port | Purpose |
|---------|-------------|------|---------|
| Ziti Controller | kontango-ziti-controller | 1280 | Overlay control plane (raft) |
| Ziti Router | kontango-ziti-router | 3022, 3023 | Mesh routing + edge listeners |
| OpenBao | kontango-bao | 8200, 8201 | Secrets, PKI, credential storage (raft) |
| Caddy | kontango-caddy | 80, 443 | TLS termination, reverse proxy, L4 SNI |
| Controller API | kontango-controller-api | 3080 | Enrollment, management, web UI |

All binaries live at `/opt/kontango/bin/`. Config files at `/etc/kontango/`. PKI at `/etc/kontango/pki/`.

## Proxmox Router Nodes

Each Proxmox node runs a single service:

| Service | Systemd Unit | Mode | Purpose |
|---------|-------------|------|---------|
| Ziti Router | kontango-ziti-router | tproxy | Mesh routing + DNS resolution + service hosting |

Router mode is **tproxy**, which provides:
- Full DNS resolution for `.tango` domains (via `127.0.0.1:53`)
- Transparent proxying of intercepted traffic
- Service hosting (bind) for all services with matching policies
- Service dialing for all services the identity has access to

## Service Mesh

### DNS Resolution

Every `.tango` hostname resolves to a Ziti IP in the `100.64.0.0/10` range. Resolution happens locally on each node through the router's built-in DNS server.

```
Application → grafana.tango → 100.64.0.X → Ziti overlay → terminator → backend
```

No external DNS is involved. `.tango` is a Ziti-internal domain.

### Service Registration Pattern

Each service has two configs:

1. **intercept.v1** — Defines the `.tango` hostname and port that clients dial
2. **host.v2** — Defines where the traffic is forwarded (terminators)

```
Service: grafana
  Intercept: grafana.tango:3000 (what clients connect to)
  Host:      grafana.tango:3000 (where traffic goes via overlay)
```

For node-specific services (Proxmox UI per node):

```
Service: hank-pmx
  Intercept: hank.tango:8006 (clients dial this)
  Host:      10.11.30.27:8006 (specific LAN backend)
```

### Service Attributes and Policies

Services are grouped by attribute. Policies grant dial/bind access by identity attribute.

**Service Attributes:**
- `#infrastructure` — Bao, Ziti controller, SSH, Proxmox UIs
- `#telemetry` — Grafana, InfluxDB, NATS

**Identity Attributes:**
- `#admin` — Full access to all services
- `#workstation` — Dial infrastructure + telemetry
- `#controller` — Bind infrastructure/telemetry, dial all
- `#device-base` — Dial telemetry only

**Policy Matrix:**

| Identity | Can Dial | Can Bind |
|----------|----------|----------|
| #admin | #all | — |
| #workstation | #infrastructure, #telemetry | — |
| #controller | #all | #infrastructure, #telemetry |
| #device-base | #telemetry | — |

## Public Access (Caddy)

Caddy runs on each controller, listening on ports 80 and 443. It handles:

- **TLS termination** with Let's Encrypt certificates (Cloudflare DNS-01 challenge)
- **Reverse proxy** to local services (controller API, Bao, Ziti management)
- **Catch-all routing** to the honeypot (404rd) for unknown domains

Public endpoints:
- `join.<domain>` → Controller API (enrollment)
- `ctrl.<domain>` → Controller API (management)
- `bao.<domain>` → OpenBao (HTTPS passthrough)
- `ziti.<domain>` → Ziti management (basic auth)
- `*.<domain>` → Honeypot (404rd)

## Firewall

Only three port ranges are publicly accessible:

| Port | Access | Purpose |
|------|--------|---------|
| 22 | Admin IP only | SSH management |
| 80 | Public | HTTP → HTTPS redirect |
| 443 | Public | All HTTPS services via Caddy |

All other ports (1280, 3022, 3023, 8200, 8201) are restricted to cluster-internal IPs only.

## PKI

```
Kontango Root CA (self-signed, EC P-256)
  └── Kontango Intermediate CA
        ├── Controller server certs (per-node)
        │     SANs: ctrl-N, *.prod.example.com, ctrl-N.tango, IP, SPIFFE URI
        ├── Router enrollment certs (per-router)
        └── Identity enrollment certs (per-device)
```

Ziti generates its own certs during enrollment using the intermediate CA as the signing authority. The signing chain and intermediate key are stored at `/etc/kontango/pki/` on each controller.

## Bootstrap Sequence

TangoKore's `kontango controller install` command runs these steps:

1. **preflight** — System checks (OS, memory, disk, ports)
2. **download** — Fetch ziti, bao, caddy binaries
3. **pki** — Generate root + intermediate CA, server certs
4. **bao-init** — Initialize OpenBao, unseal, store keys
5. **ziti** — Initialize Ziti controller, create admin user
6. **store-creds** — Save credentials to Bao KV
7. **caddy** — Generate Caddyfile, start reverse proxy
8. **schmutz** — Configure enrollment service
9. **identities** — Create Bao PKI roles and policies
10. **fabric** — Register Ziti services, policies, router/edge-router policies
11. **acl** — Configure Bao AppRoles and cert auth
12. **verify** — End-to-end verification

For join nodes (ctrl-2, ctrl-3), steps 3-5 are replaced with:
- **pki-from-leader** — Fetch CA bundle from leader's Bao
- **bao-join** — Join existing Bao raft cluster
- **ziti-join** — Join existing Ziti raft cluster
