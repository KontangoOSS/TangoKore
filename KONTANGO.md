# Kontango Ecosystem

TangoKore is the core SDK for the Kontango zero-trust mesh platform.

## Repositories

### Core
| Repo | Purpose |
|------|---------|
| **TangoKore** (this repo) | SDK binary, project docs, platform matrix |
| [schmutz](https://github.com/KontangoOSS/schmutz) | L4 edge gateway (TLS fingerprinting, SNI classification) |
| schmutz-controller | Enrollment API, deploy pipeline, NATS bus |

### Infrastructure
| Repo | Purpose |
|------|---------|
| [OpenZiti](https://github.com/openziti/ziti) | Zero-trust overlay network |
| [OpenBao](https://github.com/openbao/openbao) | Secrets management |
| [Caddy](https://github.com/caddyserver/caddy) | Reverse proxy with Ziti transport |
| [NATS](https://github.com/nats-io/nats-server) | Embedded message bus |

### Applications

The full catalog of supported applications is at [example.com](https://example.com).

## Project Standards

All Kontango projects follow the [Templatarr](https://github.com/KontangoOSS/templatarr) standard for project structure and CI/CD.

## Version Compatibility

| TangoKore | Ziti | OpenBao | Caddy | Schmutz |
|-----------|------|---------|-------|---------|
| v1.0.0 | v2.0.0-pre5 | v2.5.2 | v2.11.1 | v1.0.0 |
