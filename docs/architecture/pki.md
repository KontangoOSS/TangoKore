# PKI Architecture

## Principle

**Ziti issues all mesh certificates. Bao stores them for backup and distribution.**

Never issue Ziti internal certificates from an external PKI (Bao PKI engine, self-signed, etc.). The Ziti controller has its own signing chain that issues router and identity certificates during enrollment. External PKIs cause trust chain mismatches across the mesh.

## Certificate Chain

```
root-ca (self-signed, NetFoundry format)
  └── intermediate-ctrl-1 (signing authority)
        ├── Controller server certs
        │     CN: ctrl-N
        │     SANs: ctrl-N.tango, IP, 127.0.0.1
        │     URI: spiffe://tango/controller/ctrl-N
        ├── Router certs (issued during enrollment)
        └── Identity certs (issued during enrollment)
```

The signing chain (`intermediate-ctrl-1` + `root-ca`) lives at `/etc/kontango/pki/signing-chain.crt` and `/etc/kontango/pki/intermediate.key` on each controller.

## CA Bundle

The CA bundle at `/etc/kontango/pki/ca-bundle.pem` must contain ALL CAs that any component might present:

- Kontango Intermediate CA + Kontango Root CA
- Mesh Intermediate CA (intermediate.tango) + Mesh Root CA (tango)
- intermediate-ctrl-1 + root-ca (Ziti signing PKI)

This allows the controller to trust certs from all historical PKI generations.

## Host Config Format

**All host configs MUST use `host.v1` format.**

The C SDK `ziti-edge-tunnel` v1.12 `run-host` mode does not support `host.v2` for hosting. The `host.v2` format (with `terminators` array) is only understood by the Go SDK and the router's built-in tunneler.

```json
// host.v1 (REQUIRED for run-host)
{"address": "127.0.0.1", "port": 8200, "protocol": "tcp"}

// host.v2 (DO NOT USE for run-host)
{"terminators": [{"address": "127.0.0.1", "port": 8200, "protocol": "tcp"}]}
```

## Storage in Bao

All PKI artifacts stored in the `ziti/` namespace:

```
ziti/secret/
  admin              — username, password, cluster info
  pki/
    ca-bundle        — Full CA bundle (base64)
  controllers/
    ctrl-1           — Server cert, key, CA bundle
    ctrl-2           — Server cert, key, CA bundle
    ctrl-3           — Server cert, key, CA bundle
  routers/
    ctrl-1           — Router cert, key, CAs
    ctrl-2           — Router cert, key, CAs
    ctrl-3           — Router cert, key, CAs
    hank             — Router cert (if applicable)
```

## Certificate Lifecycle

1. **Controller init**: `ziti controller run` generates the signing chain on first boot
2. **Router enrollment**: `ziti router enroll -j <jwt>` gets a cert from the signing chain
3. **Identity enrollment**: `ziti-edge-tunnel enroll -j <jwt>` gets a cert from the signing chain
4. **Storage**: After enrollment, certs are backed up to Bao `ziti/` namespace
5. **Rotation**: Re-enroll by deleting the identity/router and recreating with a new JWT

## SPIFFE

Controller certs include SPIFFE URI SANs: `spiffe://tango/controller/ctrl-N`

The SPIFFE path segment must match the controller's node ID (derived from the cert CN). If CN is `ctrl-1`, the SPIFFE URI must be `spiffe://tango/controller/ctrl-1`.
