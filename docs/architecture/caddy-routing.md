# Caddy Routing

## Role

Caddy is the only service listening on public ports (80 and 443). It handles:

- TLS termination with Let's Encrypt certificates
- Reverse proxy to internal services
- Catch-all routing to the honeypot for unknown domains
- (Planned) L4 SNI passthrough for Ziti and Bao traffic

## Certificate Management

Caddy uses Cloudflare DNS-01 challenges for certificate issuance. This allows wildcard certificates without exposing any HTTP validation endpoints.

```
tls {
  dns cloudflare {env.CLOUDFLARE_API_TOKEN}
}
```

The Cloudflare API token is stored in `/opt/kontango/caddy/.env`.

## Route Structure

### Named Services

Each public service gets an explicit Caddy block with its domain:

- `join.<domain>` — Enrollment API (kontango-controller on port 3080)
- `ctrl.<domain>` — Management API (kontango-controller on port 3080)
- `bao.<domain>` — OpenBao vault (HTTPS reverse proxy to port 8200)
- `ziti.<domain>` — Ziti management console (reverse proxy to port 1280, basic auth protected)

### Catch-All

Any domain not explicitly matched is routed to the honeypot (404rd on port 3404). This covers all wildcard domains across all registered TLDs.

```
*.<domain> {
  reverse_proxy localhost:3404
}
```

### HTTP Redirect

Port 80 redirects all traffic to HTTPS:

```
:80 {
  redir https://{host}{uri} permanent
}
```

## L4 SNI Routing (Planned)

To fully close ports 1280, 8200, and 3023, Caddy's Layer 4 module will handle SNI-based TLS passthrough on port 443:

| SNI Match | Backend | Purpose |
|-----------|---------|---------|
| `*.prod.konoss.org` | localhost:1280 | Ziti controller (TLS passthrough) |
| `bao.*` | localhost:8200 | Bao API (TLS passthrough) |
| All other TLS | localhost:10443 | Caddy HTTPS handler (TLS termination) |

This requires:
1. Router configs updated to use `ctrl-N.prod.konoss.org:443` as the controller endpoint
2. Controller advertise address changed to `ctrl-N.prod.konoss.org:443`
3. Caddy L4 listener on port 443 with SNI matching before the HTTP handler

The Caddy binary already includes the `layer4` module with TLS SNI matchers.

## Snippets

Reusable configuration blocks:

**TLS with Cloudflare:**
```
(tls-cf) {
  tls {
    dns cloudflare {env.CLOUDFLARE_API_TOKEN}
  }
}
```

**Bao reverse proxy (TLS passthrough):**
```
(bao-proxy) {
  reverse_proxy localhost:8200 {
    transport http {
      tls
      tls_insecure_skip_verify
    }
  }
}
```

**Ziti reverse proxy (TLS passthrough):**
```
(ziti-proxy) {
  reverse_proxy localhost:1280 {
    transport http {
      tls
      tls_insecure_skip_verify
    }
  }
}
```

## Deployment

- Binary: `/opt/kontango/bin/caddy` (compiled with cloudflare-dns, layer4, nats modules)
- Config: `/opt/kontango/caddy/Caddyfile`
- Environment: `/opt/kontango/caddy/.env`
- Service: `kontango-caddy.service`
- All three controllers run identical Caddyfiles
