# Service Mesh Design

## Principles

1. **Every service gets a `.tango` hostname** — No raw IPs in configs. Services are addressed by name.
2. **Ziti issues certs, Bao stores them** — Never issue Ziti certs externally.
3. **Routers route, tunnels host** — Routers handle mesh routing only (no `--tunneler-enabled`). Service hosting is done by `ziti-edge-tunnel run-host` with dedicated identities.
4. **host.v1 only** — The C SDK tunnel doesn't support host.v2 for hosting.
5. **Role-based access** — Services are grouped by attribute. Identities get access by role, not by name.

## Service Architecture

```
Client (admin-workstation.tango)
  │
  │ dial forgejo.tango:3000
  │
  ▼
Local tunnel (ziti-edge-tunnel run)
  │ intercepts .tango DNS → 100.64.0.x
  │
  ▼
Ziti Router (ctrl-1.tango)
  │ finds terminator for forgejo service
  │ routes circuit to hosting router
  │
  ▼
Ziti Router (ctrl-2.tango)
  │ delivers to tunnel terminator
  │
  ▼
Host tunnel (ziti-edge-tunnel run-host on forgejo LXC)
  │ forwards to 127.0.0.1:3000
  │
  ▼
Forgejo application
```

## Terminator Types

| Type | Created By | Use Case |
|------|-----------|----------|
| `tunnel` | ziti-edge-tunnel run-host | C SDK hosting — this is what we use |
| `edge` | Router with --tunneler-enabled | Router built-in hosting — DO NOT USE |

**Never use `--tunneler-enabled` on routers.** It creates `edge` terminators for ALL services the router identity can bind, regardless of whether the service actually runs on that node. This causes phantom terminators that route traffic to `127.0.0.1` on nodes where the service doesn't exist.

## Services (Live)

### Infrastructure (#infrastructure)

| Service | Intercept | Host | Hosted By |
|---------|----------|------|-----------|
| bao-api | bao.tango:8200 | 127.0.0.1:8200 | DO controllers |
| bao-cluster | bao.tango:8201 | 127.0.0.1:8201 | DO controllers |
| ziti-ctrl | ziti-ctrl.tango:1280 | 127.0.0.1:1280 | DO controllers |
| ssh-private | ssh.tango:22 | 127.0.0.1:22 | DO controllers |
| pmx-api | pmx.tango:8006 | 127.0.0.1:8006 | DO controllers |
| hank-pmx | hank.tango:8006 | 127.0.0.1:8006 | DO controllers* |
| slim1-pmx | slim1.tango:8006 | 127.0.0.1:8006 | DO controllers* |
| slim2-pmx | slim2.tango:8006 | 127.0.0.1:8006 | DO controllers* |
| pve-pmx | pve.tango:8006 | 127.0.0.1:8006 | DO controllers* |

*Node-specific PMX services should eventually be hosted by their respective LAN nodes.

### Telemetry (#telemetry)

| Service | Intercept | Host | Hosted By |
|---------|----------|------|-----------|
| nats-telemetry | nats.tango:4222 | 127.0.0.1:4222 | DO controllers |
| grafana | grafana.tango:3000 | 127.0.0.1:3000 | DO controllers |
| influxdb | influxdb.tango:8086 | 127.0.0.1:8086 | DO controllers |

### Web Services (#web-services)

| Service | Intercept | Host | Hosted By |
|---------|----------|------|-----------|
| forgejo | forgejo.tango:3000 | 127.0.0.1:3000 | forgejo LXC |

## Adding a New Service

1. Create intercept config: `name-intercept` (intercept.v1)
2. Create host config: `name-host` (**host.v1**, address `127.0.0.1`)
3. Create service with role attribute (`infrastructure`, `telemetry`, or `web-services`)
4. Attach both configs to the service
5. Existing bind/dial policies cover it — no new policy needed
6. Restart the `run-host` process on the hosting node to pick up the new service

## Public Access via Caddy

For services that need public web access, add a Caddy route:

```
# Direct proxy (service runs on controller)
bao.example.org {
  reverse_proxy localhost:8200 { transport http { tls; tls_insecure_skip_verify } }
}

# Ziti transport (service runs on LAN node)
git.example.org {
  reverse_proxy forgejo.tango:3000 {
    transport ziti { identity /opt/kontango/caddy/caddy-gateway.json }
  }
}
```

The Ziti transport uses the `caddy-gateway.tango` identity (role `#gateway`) which can only reach `#public` routers and can only dial `#web-services`.

## DNS Resolution

- `.tango` domains resolve through the Ziti tunnel's DNS interceptor (100.64.0.0/10 range)
- Public domains (`.example.org`) resolve through Cloudflare DNS
- OPNsense BIND should NOT have local overrides for any `.example.org` domains
- `/etc/hosts` should NOT have entries for `.example.org` domains (causes Ziti tunnel to cache stale IPs)
