# Service Mesh Design

## Principles

1. **Every service gets a `.tango` hostname** — No raw IPs in configs. Services are addressed by name.
2. **host.v2 for all host configs** — Supports multi-terminator routing and smartrouting.
3. **tproxy mode on all edge routers** — Provides transparent DNS resolution and traffic interception.
4. **Least-privilege policies** — Services are grouped by attribute. Identities get the minimum access needed.
5. **Caddy for public ingress only** — Internal traffic stays on the Ziti overlay. Caddy handles TLS termination for external clients.

## Service Registration

### Config Types

Each service requires two configs:

**intercept.v1** — Client-side. Defines what hostname and port clients use to reach the service.
```json
{
  "protocols": ["tcp"],
  "addresses": ["grafana.tango"],
  "portRanges": [{"low": 3000, "high": 3000}]
}
```

**host.v2** — Server-side. Defines where traffic is forwarded. Uses terminators for multi-host routing.
```json
{
  "terminators": [{
    "address": "grafana.tango",
    "port": 3000,
    "protocol": "tcp"
  }]
}
```

When the host address is a `.tango` name, the router resolves it through the Ziti overlay. When it's a LAN IP, the router forwards directly on the local network.

### Adding a New Service

1. Create the host config (host.v2)
2. Create the intercept config (intercept.v1)
3. Create the service with both configs and the appropriate attribute
4. The service is immediately available on all routers with matching bind policies

No new policies are needed unless the service belongs to a new attribute group.

### Service Attributes

| Attribute | Description | Bound by | Dialed by |
|-----------|-------------|----------|-----------|
| `#infrastructure` | Core platform services | `#controller` | `#admin`, `#workstation` |
| `#telemetry` | Monitoring and metrics | `#controller`, `#telemetry` | `#admin`, `#workstation`, `#device-base` |

## Router Configuration

### Controller Routers (DO nodes)

Controller routers run in **host** mode. They bind services but do not intercept local traffic — the controller processes access services directly via localhost.

- Link listeners on port 3022 (inter-router mesh)
- Edge listeners on port 3023 (client connections)
- Tunnel binding in host mode

### Edge Routers (edge nodes)

Edge routers run in **tproxy** mode. They both bind and intercept services, providing:

- DNS resolution: `.tango` hostnames resolve to Ziti IPs (`100.64.0.0/10`)
- Transparent interception: Traffic to Ziti IPs is captured and routed through the overlay
- Local hosting: Services running on the node are terminated locally

**DNS requirement:** The node's `/etc/resolv.conf` must list `127.0.0.1` as the first nameserver so the router's DNS server handles `.tango` queries.

### Router Identity Attributes

| Attribute | Purpose |
|-----------|---------|
| `#controller` | Can bind `#infrastructure` services |
| `#telemetry` | Can bind `#telemetry` services |
| `#lan` | Identifies LAN-connected routers |

## Smartrouting

When multiple routers bind the same service, Ziti's smartrouting selects the optimal terminator based on:

- Latency to each terminator
- Current load and dynamic cost
- Router proximity in the mesh

This means a client on `node-1` dialing `bao.tango` will prefer the terminator on `node-1` (local) over one on `ctrl-1` (remote), unless the local terminator is unhealthy.

## DNS Flow

```
1. Application resolves grafana.tango
2. Query hits 127.0.0.1:53 (Ziti router DNS)
3. Router returns 100.64.0.X (Ziti-assigned IP)
4. Application connects to 100.64.0.X:3000
5. tproxy intercepts the connection
6. Router dials grafana service through Ziti overlay
7. Smartrouting selects best terminator
8. Traffic delivered to backend
```

## Naming Conventions

| Pattern | Example | Usage |
|---------|---------|-------|
| `<service>.tango` | `grafana.tango` | Service intercept address |
| `<node>.tango` | `node-1.tango` | Node-specific service (e.g., hypervisor UI) |
| `<service>-host` | `grafana-host` | host.v2 config name |
| `<service>-intercept` | `grafana-intercept` | intercept.v1 config name |
| `<node>-pmx` | `node-1-pmx` | Per-node hypervisor UI service name |
