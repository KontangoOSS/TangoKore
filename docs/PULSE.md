# Pulse Telemetry

Pulse is how machines and applications report telemetry on the Kontango mesh. It's designed to be tiny, fast, and trivial to use.

## How it works

Every enrolled machine runs a **NATS leaf node** on `localhost:4222`. Applications publish pulses to the local NATS — zero network latency. The leaf syncs upstream to the controller's NATS hub through the Ziti overlay automatically.

The NATS subject encodes the source:

```
tango.telemetry.<machineID>.<slug>
```

- `machineID` — the enrolled machine's unique ID
- `slug` — the source: `system` for OS metrics, or an application slug like `kore/ticketarr`

The payload is MessagePack-encoded. System pulses use positional arrays (33 bytes). Application pulses use `map[string]string` (schema-free).

**Telemetry is outbound only** — machines push to the controller. Config updates flow inbound — the controller pushes to machines via `tango.config.<machineID>`.

## Wire format

```
NATS subject: tango.telemetry.23c0c89b.kore/ticketarr
NATS payload: [msgpack map] {"status":"healthy","version":"2.1.0"}
              44 bytes on the wire
```

For comparison, the same data as JSON would be 56 bytes. For a single KV pair (`{"status":"healthy"}`), msgpack is 16 bytes vs JSON's 20.

## Sending pulses from an application

### NATS (direct, any language)

Every machine runs a NATS leaf node on `localhost:4222`. Connect with any NATS client library and publish:

```sh
# Using the NATS CLI
nats pub tango.telemetry.local.kore/ticketarr '{"status":"healthy","version":"2.1.0"}'
```

```go
// From Go
nc, _ := nats.Connect("nats://localhost:4222")
nc.Publish("tango.telemetry.local.kore/ticketarr", data)
```

```python
# From Python
import nats
nc = await nats.connect("nats://localhost:4222")
await nc.publish("tango.telemetry.local.kore/ticketarr", data)
```

The leaf handles upstream sync to the controller through Ziti. Your app never touches the network.

### HTTP API (simple alternative)

A local HTTP pulse API also runs on `127.0.0.1:8801` for apps that don't want a NATS client:

```sh
curl -X POST http://localhost:8801/pulse \
  -d '{"slug":"kore/ticketarr","kv":{"status":"healthy","version":"2.1.0"}}'
```

### From a Docker container

```yaml
# compose.yml
services:
  myapp:
    image: myapp:latest
    # The host's pulse API is accessible via the Docker bridge
    extra_hosts:
      - "pulse:host-gateway"
    environment:
      - PULSE_URL=http://pulse:8801/pulse
```

```sh
# Inside the container
curl -X POST $PULSE_URL -d '{"slug":"kore/myapp","kv":{"status":"ok","requests":"42"}}'
```

### From Go (using the Kontango SDK)

```go
import "github.com/KontangoOSS/TangoKore/internal/agent"

// The agent handles serialization and transport
kv := agent.KV{"status": "healthy", "version": "2.1.0"}
data, _ := agent.encodeKV(kv)
// publish to NATS subject tango.telemetry.<machineID>.kore/ticketarr
```

## Key conventions

System keys (emitted by the agent automatically):

| Key | Example | Description |
|-----|---------|-------------|
| `hostname` | `mcphub` | Machine hostname |
| `os` | `linux` | Operating system |
| `arch` | `amd64` | CPU architecture |
| `cpus` | `2` | Core count |
| `load` | `1.42` | 1-minute load average |
| `mem_mb` | `193280` | Total memory in MB |
| `up` | `500000` | Uptime in seconds |

Application keys are namespaced by slug in the NATS subject. The KV payload itself uses plain keys:

```sh
# Subject: tango.telemetry.<id>.kore/ticketarr
# Payload KV:
{"status":"healthy","version":"2.1.0","requests":"1547"}
```

On the controller, app keys are stored prefixed: `kore/ticketarr.status`, `kore/ticketarr.version`.

## Subscribing to pulses

### From the controller API

```sh
# All machines, all metrics
curl https://join.kontango.net/api/pulse/live

# Specific machine
curl https://join.kontango.net/api/pulse/<machineID>
```

### From BrowZer (browser via Ziti overlay)

The browser subscribes to `tango.telemetry.>` through the Ziti BrowZer SDK and receives raw msgpack events. Decode with any msgpack library.

### From NATS directly (on the mesh)

```
nats sub "tango.telemetry.>"                              # all machines, all sources
nats sub "tango.telemetry.23c0c89b.system"                # one machine, system only
nats sub "tango.telemetry.*.kore/ticketarr"               # all machines, ticketarr only
```

NATS does server-side filtering — you only receive messages matching your subscription pattern.

## Size reference

| Pulse type | KV pairs | MsgPack bytes | JSON bytes |
|-----------|----------|---------------|------------|
| Single metric | 1 | 16 | 20 |
| App health | 3 | 44 | 56 |
| System heartbeat | 7 | 78 | 106 |

At 100 pulses/minute with 7 KV pairs each: **~8 KB/min** over the wire.
