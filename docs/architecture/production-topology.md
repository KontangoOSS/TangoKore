# Production Topology

## Overview

The kontango platform runs on a 3-node controller cluster (DigitalOcean) connected to on-premises nodes (Proxmox hypervisors + LXC containers) through an OpenZiti overlay mesh. All inter-service communication uses `.tango` DNS names resolved through the Ziti network. Only ports 80 and 443 are exposed to the public internet.

## Cluster Layout

```
                    ┌──────────────────────────────────────────┐
                    │         DigitalOcean (Public Cloud)       │
                    │                                          │
                    │  ┌──────────┐ ┌──────────┐ ┌──────────┐ │
                    │  │  ctrl-1  │ │  ctrl-2  │ │  ctrl-3  │ │
                    │  │  .tango  │ │  .tango  │ │  .tango  │ │
                    │  │ region-1 │ │ region-2 │ │ region-3 │ │
                    │  └────┬─────┘ └────┬─────┘ └────┬─────┘ │
                    │       │            │            │        │
                    │       └──────┬─────┘────────────┘        │
                    │              │  Raft consensus (Ziti+Bao) │
                    └──────────────┼───────────────────────────┘
                                   │
                           Ziti overlay mesh
                            (port 3023 edge)
                                   │
                    ┌──────────────┼───────────────────────────┐
                    │              │     Proxmox LAN            │
                    │  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐    │
                    │  │ hank │ │slim1 │ │slim2 │ │ pve  │    │
                    │  │.tango│ │.tango│ │.tango│ │.tango│    │
                    │  └──┬───┘ └──┬───┘ └──┬───┘ └──┬───┘    │
                    │     │        │        │        │         │
                    │     └────────┴────────┴────────┘         │
                    │          10.x.x.0/24 LAN               │
                    │                                          │
                    │  ┌────────────┐                          │
                    │  │ forgejo LXC│  (10.x.x.30)           │
                    │  │   .tango   │                          │
                    │  └────────────┘                          │
                    └──────────────────────────────────────────┘
```

## Controller Nodes (DO)

Each controller runs these processes:

| Process | Binary | Identity | Port | Managed By |
|---------|--------|----------|------|------------|
| Ziti Controller | ziti controller run | server cert (CN=ctrl-N) | 1280 | nohup |
| Ziti Router | ziti router run | router cert | 3022, 3023 | nohup |
| Host Tunnel | ziti-edge-tunnel run-host | ctrl-N-host.tango | — | nohup |
| Caddy | caddy run | caddy-gateway.tango (Ziti transport) | 80, 443 | kontango-caddy.service |
| Kontango Controller | kontango-controller | — (env auth) | 3080 | kontango-controller.service |
| OpenBao | bao server | — | 8200, 8201 | systemd |
| 404rd | 404rd | — | 3404, 4222 | 404rd.service |

Key design: **Routers do NOT have `--tunneler-enabled`**. Service hosting is handled by separate `ziti-edge-tunnel run-host` processes with dedicated identities. This prevents routers from creating terminators for services that don't run on that node.

All binaries at `/opt/kontango/bin/` (Ziti, Caddy, kontango-controller) and `/opt/openziti/bin/` (ziti-edge-tunnel). Config at `/etc/kontango/`. PKI at `/etc/kontango/pki/`.

## LAN Nodes (Proxmox)

Each LAN node runs `ziti-edge-tunnel` in `run` mode (tproxy + DNS) via systemd:

| Node | IP | Service Unit | Identity | Role |
|------|----|-------------|----------|------|
| hank | 10.x.x.27 | ziti-edge-tunnel.service | hank.tango | lan-host, lan, private |
| slim1 | 10.x.x.213 | ziti-tunnel-tango.service | slim1.tango | lan-host, lan, private |
| slim2 | 10.x.x.230 | ziti-tunnel-tango.service | slim2.tango | lan-host, lan, private |
| pve | 10.x.x.90 | ziti-tunnel-tango.service | pve.tango | lan-host, lan, private |
| forgejo | 10.x.x.30 | ziti-tunnel-tango.service + run-host | forgejo.tango | lan-host, lan, private |

The `run` mode provides:
- DNS resolution for `.tango` domains (via tun device, 100.64.0.0/10)
- Transparent proxying of intercepted traffic
- Service dialing for all services the identity can access

The `run-host` mode (forgejo LXC) additionally:
- Hosts (binds) services with matching host.v1 configs
- Creates `tunnel` type terminators

## Identity Roles

| Role | Purpose | Who |
|------|---------|-----|
| `controller-host` | Binds infrastructure + telemetry on DO controllers | ctrl-N-host.tango |
| `lan-host` | Binds web-services on LAN nodes | forgejo.tango, hank-tunnel, slim*.tango, pve.tango |
| `admin` | Dials all services | admin-workstation.tango |
| `workstation` | Dials infrastructure + telemetry + web | admin-workstation.tango |
| `gateway` | Dials web-services via public routers only (Caddy) | caddy-gateway.tango |
| `lan` | General LAN node identifier | All LAN identities |
| `private` | Non-public node | All LAN identities |

## Services

| Service | Intercept | Host | Attribute | Hosted By |
|---------|----------|------|-----------|-----------|
| bao-api | bao.tango:8200 | 127.0.0.1:8200 | infrastructure | controller-host |
| bao-cluster | bao.tango:8201 | 127.0.0.1:8201 | infrastructure | controller-host |
| ziti-ctrl | ziti-ctrl.tango:1280 | 127.0.0.1:1280 | infrastructure | controller-host |
| ssh-private | ssh.tango:22 | 127.0.0.1:22 | infrastructure | controller-host |
| pmx-api | pmx.tango:8006 | 127.0.0.1:8006 | infrastructure | controller-host |
| nats-telemetry | nats.tango:4222 | 127.0.0.1:4222 | telemetry | controller-host |
| grafana | grafana.tango:3000 | 127.0.0.1:3000 | telemetry | controller-host |
| influxdb | influxdb.tango:8086 | 127.0.0.1:8086 | telemetry | controller-host |
| forgejo | forgejo.tango:3000 | 127.0.0.1:3000 | web-services | lan-host (forgejo LXC) |

**All host configs MUST be `host.v1` format.** The C SDK `ziti-edge-tunnel` v1.12 `run-host` mode does not support `host.v2` for hosting.

## Service Policies

### Bind (who hosts services)

| Policy | Identity Role | Service Role |
|--------|--------------|-------------|
| infra-bind | #controller-host | #infrastructure |
| telemetry-bind | #controller-host | #telemetry |
| web-bind | #lan-host | #web-services |

### Dial (who consumes services)

| Policy | Identity Role | Service Role |
|--------|--------------|-------------|
| admin-dial-all | #admin | #all |
| workstation-dial-infra | #workstation | #infrastructure |
| workstation-dial-telemetry | #workstation | #telemetry |
| workstation-dial-web | #admin, #workstation | #web-services |
| gateway-dial-web | #gateway | #web-services |
| lan-host-dial-all | #lan-host | #all |
| device-dial-telemetry | #device-base | #telemetry |

### Edge Router Policies

| Policy | Router Role | Identity Role |
|--------|------------|--------------|
| lan-to-all | #all | #admin, #controller-host, #device-base, #lan, #lan-host, #tunnel-host, #workstation |
| gateway-to-public | #public | #gateway |

## Public Access (Caddy)

Caddy on each controller handles TLS termination with Let's Encrypt (Cloudflare DNS-01). Routes:

| Domain | Backend | Method |
|--------|---------|--------|
| ctrl.example.org | localhost:3080 | Direct proxy |
| bao.example.org | localhost:8200 | HTTPS passthrough |
| git.example.org | forgejo.tango:3000 | **Ziti transport plugin** |
| *.example.org | localhost:3404 | 404rd honeypot |
| *.yourdomain.io/net/org/us | localhost:3404 | 404rd honeypot |

The `git.example.org` route uses Caddy's Ziti transport plugin with the `caddy-gateway.tango` identity to dial the forgejo service through the overlay. This means the forgejo LXC is never directly exposed — traffic flows: `Internet → Caddy → Ziti mesh → forgejo LXC`.

## Firewall (DO Cloud Firewall)

| Port | Source | Purpose |
|------|--------|---------|
| 22 | Admin IP /24 | SSH management |
| 80 | 0.0.0.0/0 | HTTP redirect |
| 443 | 0.0.0.0/0 | All HTTPS via Caddy |
| 1280 | 0.0.0.0/0 | Ziti control plane |
| 3022 | Admin IP /24 + controllers | Router fabric links |
| 3023 | 0.0.0.0/0 | Ziti edge connections |
| 8200-8201 | Controllers only | Bao raft cluster |

## PKI

Ziti issues all mesh certificates. Bao stores them for backup.

```
Ziti Signing Chain (intermediate-ctrl-1 → root-ca)
  ├── Controller server certs (per-node)
  │     CN: ctrl-N
  │     SANs: ctrl-N.tango, IP, spiffe://tango/controller/ctrl-N
  ├── Router enrollment certs (per-router)
  └── Identity enrollment certs (per-device)
```

**Key rule: Ziti issues certs, Bao stores them. Never issue Ziti certs externally.**

All certs stored in Bao `ziti/` namespace:
- `secret/admin` — Ziti admin credentials
- `secret/pki/ca-bundle` — Full CA bundle
- `secret/controllers/ctrl-{1,2,3}` — Controller server certs
- `secret/routers/ctrl-{1,2,3}` — Router certs

## Bootstrap Sequence

TangoKore's `kontango controller install` runs:

1. **preflight** — System checks
2. **download** — Fetch ziti, bao, caddy binaries
3. **pki** — Generate root + intermediate CA, server certs
4. **bao-init** — Initialize OpenBao, unseal, store keys
5. **ziti** — Initialize Ziti controller, create admin user
6. **store-creds** — Save credentials to Bao KV
7. **caddy** — Generate Caddyfile, start reverse proxy
8. **kontango-controller** — Configure enrollment service
9. **identities** — Create Bao PKI roles and policies
10. **fabric** — Register Ziti services, policies
11. **acl** — Configure Bao AppRoles and cert auth
12. **verify** — End-to-end verification

For join nodes (ctrl-2, ctrl-3), steps 3-5 are replaced with:
- **pki-from-leader** — Fetch CA bundle from leader's Bao
- **bao-join** — Join existing Bao raft cluster
- **ziti-join** — Join existing Ziti raft cluster (requires `minClusterSize: 0` on joiner)

## Raft Cluster Notes

- **Ziti**: ctrl-2 is leader, ctrl-1 and ctrl-3 are voters. Joiner nodes need `minClusterSize: 0` in ctrl.yaml and must be added via `ziti agent cluster add` before they self-bootstrap.
- **Bao**: 3-node raft, ctrl-2 is leader. Root token and unseal key in `ziti:secret/admin` and `secret/bao/init`.
- Both raft clusters use direct IPs for peer communication (not DNS CNAMEs, which break raft).
