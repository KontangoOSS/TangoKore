# PKI Architecture

## Certificate Hierarchy

```
Kontango Root CA (EC P-256, self-signed, 10-year)
  └── Kontango Intermediate CA (EC P-256, 5-year)
        ├── Controller server certificates (per-node)
        ├── Router enrollment certificates (per-router, issued by Ziti)
        └── Identity enrollment certificates (per-device, issued by Ziti)
```

The root and intermediate CAs are generated once during leader bootstrap. The intermediate CA key is used by the Ziti controller to sign enrollment certificates for routers and identities.

## Certificate Locations

### Controllers (`/etc/kontango/pki/`)

| File | Purpose |
|------|---------|
| `ca-bundle.pem` | Root CA + Intermediate CA (distributed to all clients) |
| `server.crt` | Node's server certificate |
| `server.key` | Node's server private key |
| `signing-chain.crt` | Intermediate cert chain (used by Ziti for enrollment signing) |
| `intermediate.key` | Intermediate CA private key (used by Ziti for enrollment signing) |

### Routers (`/opt/kontango/ziti/<router-name>/`)

| File | Purpose |
|------|---------|
| `client.crt` | Router's client certificate (issued during enrollment) |
| `client.key` | Router's client private key |
| `server.crt` | Router's server certificate (for edge listeners) |
| `cas.crt` | CA bundle (matches controller's ca-bundle.pem) |

## Server Certificate SANs

Controller certificates include these Subject Alternative Names:

- `ctrl-N` (short name)
- `*.prod.example.com` (wildcard for Caddy SNI routing)
- `ctrl-N.prod.example.com` (node-specific)
- `ctrl-N.tango` (overlay address)
- Node public IP address
- `127.0.0.1`
- SPIFFE URI: `spiffe://prod.example.com/controller/ctrl-N`

## Trust Model

### Controller → Controller
Controllers trust each other through the shared CA bundle. Raft communication (Ziti on 1280, Bao on 8201) uses mTLS with the same server certificate.

### Router → Controller
Routers connect to controllers using the CA bundle received during enrollment. The controller endpoint is specified as a hostname that matches the server certificate SANs.

### Client → Controller
CLI tools (like `ziti edge login`) must provide the CA bundle via `--ca` flag. The CA file is stored locally at `~/.kontango/certs/ziti-ca.pem`.

### Identity → Services
Device identities receive certificates during enrollment. These certificates are signed by the Ziti controller using the intermediate CA. The identity file (JSON) contains the certificate, key, and CA bundle inline.

## Key Operations

### Generate PKI (leader only)
```
ziti pki create ca --pki-root <dir> --ca-name root-ca --trust-domain <domain>
ziti pki create intermediate --ca-name root-ca --intermediate-name intermediate
ziti pki create server --intermediate-name intermediate --server-name <node>
```

### Enroll Router
```
ziti edge create edge-router <name> --tunneler-enabled --role-attributes <attrs>
ziti router enroll <config.yaml> --jwt <token.jwt>
```

### Enroll Identity
```
ziti edge create identity <name> --role-attributes <attrs> -o <token.jwt>
ziti edge enroll -j <token.jwt> -o <identity.json>
```

## Rotation

Certificate rotation is handled by re-enrolling the router or identity. The controller issues a new certificate from the same intermediate CA. No manual PKI regeneration is needed unless the intermediate CA itself expires.

Bao stores a backup of all issued certificates for audit and recovery purposes.
