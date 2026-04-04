// Package agent implements the universal Kontango machine agent.
//
// The agent has no local config. The Ziti identity is the only thing on disk.
// Everything else comes live from the controller on every connection.
//
// Two services, agent always dials:
//
//	nats.tango   — NATS broker; agent publishes telemetry events here
//	config.tango — agent sends its machine ID as a handshake, controller
//	                  pushes newline-delimited JSON instructions down the pipe
//
// The controller is the source of truth. The agent just sends data and
// listens for instructions. Nothing is cached locally.
package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	natsgo "github.com/nats-io/nats.go"
	"github.com/openziti/sdk-golang/ziti"
)

// BootConfig is the only config the agent needs locally.
// Everything else comes from the controller.
type BootConfig struct {
	// IdentityPath is the Ziti identity JSON written by kontango enroll.
	// Default: /opt/kontango/identity.json
	IdentityPath string

	// FallbackURL is the enroll API base URL used when Ziti is not yet reachable.
	// Example: "https://your-controller.example"
	FallbackURL string

	// TelemetryService overrides the Ziti service name for NATS. Default: "nats.tango"
	TelemetryService string

	// ConfigService overrides the Ziti service name. Default: "config.tango"
	ConfigService string

	// DefaultInterval is the heartbeat interval used before the controller
	// sends one. Default: 60s
	DefaultInterval time.Duration
}

func (c *BootConfig) telemetrySvc() string {
	if c.TelemetryService != "" {
		return c.TelemetryService
	}
	return "nats.tango"
}

func (c *BootConfig) configSvc() string {
	if c.ConfigService != "" {
		return c.ConfigService
	}
	return "config.tango"
}

func (c *BootConfig) defaultInterval() time.Duration {
	if c.DefaultInterval > 0 {
		return c.DefaultInterval
	}
	return 60 * time.Second
}

// Heartbeat is pushed to nats.tango on every tick.
// Short field names — target ~150 bytes over the wire.
type Heartbeat struct {
	MachineID  string  `json:"mid"`
	Hostname   string  `json:"host"`
	OS         string  `json:"os"`
	Arch       string  `json:"arch"`
	UptimeSecs int64   `json:"up"`
	CPUCores   int     `json:"cpu"`
	MemoryMB   int64   `json:"mem,omitempty"`
	LoadAvg1   float64 `json:"load,omitempty"`
	Nickname   string  `json:"nick,omitempty"`
	State      string  `json:"state,omitempty"`
	Profile    string  `json:"profile,omitempty"`
	Timestamp  int64   `json:"ts"`
}

// Instruction is pushed from the controller down the config.tango connection.
type Instruction struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// helloPayload is what the controller sends on connect and in heartbeat responses.
// Contains whatever the controller knows about this machine that's operationally useful.
// Ziti role attributes and policies are controller-only — never sent here.
type helloPayload struct {
	Interval int    `json:"interval"`
	Nickname string `json:"nickname,omitempty"`
	State    string `json:"state,omitempty"`
	Profile  string `json:"profile,omitempty"`
	OSRef    string `json:"os_ref,omitempty"`
	OSVer    string `json:"os_ver,omitempty"`
	Arch     string `json:"arch,omitempty"`
	CPU      string `json:"cpu,omitempty"`
}

// state holds the latest hello payload received from the controller.
// Written on hello, read when building heartbeats.
var state struct {
	mu       sync.Mutex
	hello    helloPayload
	hasHello bool
}

// Run starts the agent and blocks until ctx is cancelled.
//
// The agent tries Ziti (NATS telemetry + config channel) as the primary
// transport. If Ziti isn't available yet — e.g. on a freshly enrolled machine
// where the tunnel hasn't connected — it falls back to HTTP heartbeating
// against the same controller endpoint. Both paths hit the same API; the
// transport is transparent. The agent continuously retries Ziti in the
// background and promotes to overlay when available.
func Run(ctx context.Context, boot BootConfig, logger *slog.Logger) error {
	if boot.IdentityPath == "" {
		boot.IdentityPath = defaultIdentityPath()
	}

	machineID, err := loadMachineID(boot.IdentityPath)
	if err != nil {
		return fmt.Errorf("machine record not found — enrolled? (%w)", err)
	}

	logger = logger.With("mid", machineID)

	intervalCh := make(chan time.Duration, 1)
	eventCh := make(chan []byte, 256)

	// Start collectors — they run for the lifetime of ctx regardless of transport.
	go (&pulseCollector{intervalCh: intervalCh, initial: boot.defaultInterval()}).collect(ctx, machineID, eventCh)
	go (&netCollector{interval: 5 * time.Minute}).collect(ctx, machineID, eventCh)
	go (&logCollector{}).collect(ctx, machineID, eventCh)

	// Start the local pulse API so applications can emit their own tagged KV events.
	if ln, err := startPulseAPI(machineID, eventCh, "127.0.0.1:8801"); err != nil {
		logger.Warn("pulse API failed to start", "error", err)
	} else {
		defer ln.Close()
		logger.Info("pulse API listening", "addr", "127.0.0.1:8801")
	}

	// Edge buffer holds events when neither NATS nor HTTP is available.
	edgeBuf := newEventBuffer(5 * time.Minute)

	// Try to connect via Ziti. If it works, run the overlay path.
	// If not, start HTTP fallback and keep retrying Ziti in the background.
	for {
		if ctx.Err() != nil {
			return nil
		}

		zitiCtx, err := ziti.NewContextFromFile(boot.IdentityPath)
		if err != nil {
			logger.Info("ziti identity not available, using HTTP fallback", "error", err)
			if boot.FallbackURL == "" {
				logger.Error("no fallback URL configured, waiting for overlay")
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(30 * time.Second):
					continue
				}
			}
			runFallbackUntilOverlay(ctx, boot, machineID, eventCh, edgeBuf, intervalCh, logger)
			continue
		}

		// Verify the overlay actually works by trying to dial NATS.
		// NewContextFromFile only loads the identity — it doesn't connect.
		logger.Info("ziti identity loaded, verifying overlay connectivity...")
		testConn, testErr := zitiCtx.Dial(boot.telemetrySvc())
		if testErr != nil {
			zitiCtx.Close()
			logger.Warn("ziti overlay not reachable, using HTTP fallback", "error", testErr)
			if boot.FallbackURL == "" {
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(30 * time.Second):
					continue
				}
			}
			runFallbackUntilOverlay(ctx, boot, machineID, eventCh, edgeBuf, intervalCh, logger)
			continue
		}
		testConn.Close()
		logger.Info("ziti overlay verified, switching to NATS telemetry")

		// Run NATS telemetry + config subscription — both on the same NATS connection.
		// Telemetry publishes to tango.telemetry.<machineID>
		// Config subscribes to tango.config.<machineID>
		runTelemetryLoop(ctx, zitiCtx, boot, machineID, eventCh, intervalCh, logger)
		zitiCtx.Close()

		if ctx.Err() != nil {
			return nil
		}

		logger.Warn("overlay disconnected, falling back to HTTP")
		// Loop back — try Ziti again, fall back to HTTP if needed.
	}
}

// runFallbackUntilOverlay sends heartbeats via HTTP while periodically
// trying to establish a Ziti connection. Returns when Ziti becomes available
// or ctx is cancelled.
func runFallbackUntilOverlay(ctx context.Context, boot BootConfig, machineID string, eventCh chan []byte, edgeBuf *eventBuffer, intervalCh chan time.Duration, logger *slog.Logger) {
	interval := boot.defaultInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Try Ziti every 30 seconds alongside HTTP heartbeating.
	zitiRetry := time.NewTicker(30 * time.Second)
	defer zitiRetry.Stop()

	beat := func() {
		hb := buildHeartbeat(machineID)
		ev, _ := encodeEvent(machineID, "hb", hb)
		if cmd, _ := httpHeartbeat(ctx, boot.FallbackURL, machineID, ev); cmd != nil {
			handleCmd(cmd, intervalCh)
		}
		// Also drain any buffered events via HTTP.
		for _, payload := range edgeBuf.drain() {
			httpHeartbeat(ctx, boot.FallbackURL, machineID, payload)
		}
		// Drain collector events.
		for {
			select {
			case payload := <-eventCh:
				httpHeartbeat(ctx, boot.FallbackURL, machineID, payload)
			default:
				return
			}
		}
	}

	beat()

	for {
		select {
		case <-ctx.Done():
			return
		case d := <-intervalCh:
			interval = d
			ticker.Reset(d)
		case <-ticker.C:
			beat()
		case <-zitiRetry.C:
			// Try Ziti — if it works, return so the main loop promotes to overlay.
			if zitiCtx, err := ziti.NewContextFromFile(boot.IdentityPath); err == nil {
				zitiCtx.Close()
				logger.Info("ziti became available, promoting to overlay")
				return
			}
		}
	}
}

// -- Instruction handling ----------------------------------------------------

// handleInstruction processes a single instruction from the controller,
// whether it arrives via NATS subscription or the legacy TCP config channel.
func handleInstruction(instr Instruction, boot BootConfig, machineID string, zitiCtx ziti.Context, eventCh chan<- []byte, intervalCh chan<- time.Duration, logger *slog.Logger) {
	switch instr.Type {
	case "hello":
		var cfg helloPayload
		if err := json.Unmarshal(instr.Payload, &cfg); err == nil {
			state.mu.Lock()
			state.hello = cfg
			state.hasHello = true
			state.mu.Unlock()
			if cfg.Interval > 0 {
				select {
				case intervalCh <- time.Duration(cfg.Interval) * time.Second:
				default:
				}
			}
		}

	case "config":
		var cfg helloPayload
		if err := json.Unmarshal(instr.Payload, &cfg); err == nil {
			state.mu.Lock()
			state.hello = cfg
			state.hasHello = true
			state.mu.Unlock()
			if cfg.Interval > 0 {
				select {
				case intervalCh <- time.Duration(cfg.Interval) * time.Second:
				default:
				}
			}
			logger.Info("config updated", "nickname", cfg.Nickname, "interval", cfg.Interval)
		}

	case "apply":
		go func() {
			ctx := context.Background()
			result := handleApply(ctx, zitiCtx, instr.Payload, logger)
			if ev, err := encodeEvent(machineID, "apply", result); err == nil {
				select {
				case eventCh <- ev:
				default:
				}
			}
		}()

	case "set_interval":
		var p struct {
			Seconds int `json:"seconds"`
		}
		if err := json.Unmarshal(instr.Payload, &p); err == nil && p.Seconds > 0 {
			select {
			case intervalCh <- time.Duration(p.Seconds) * time.Second:
			default:
			}
		}

	case "reload":
		logger.Info("reload requested")
	}
}

// -- Config channel (legacy TCP, kept for backwards compat) ------------------

// runConfigChannel dials config.tango, sends the machine ID, and reads
// instructions pushed down by the controller. Reconnects on disconnect.
func runConfigChannel(ctx context.Context, zitiCtx ziti.Context, boot BootConfig, machineID string, eventCh chan<- []byte, intervalCh chan<- time.Duration, configReady chan struct{}, logger *slog.Logger) {
	backoff := 5 * time.Second
	for {
		if ctx.Err() != nil {
			return
		}

		conn, err := zitiCtx.Dial(boot.configSvc())
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
				if backoff < 5*time.Minute {
					backoff *= 2
				}
			}
			continue
		}
		backoff = 5 * time.Second

		readInstructions(ctx, conn, boot, machineID, zitiCtx, eventCh, intervalCh, configReady, logger)
		conn.Close()

		select {
		case <-ctx.Done():
			return
		case <-time.After(15 * time.Second):
		}
	}
}

func readInstructions(ctx context.Context, conn io.ReadWriteCloser, boot BootConfig, machineID string, zitiCtx ziti.Context, eventCh chan<- []byte, intervalCh chan<- time.Duration, configReady chan struct{}, logger *slog.Logger) {
	go func() { <-ctx.Done(); conn.Close() }()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var instr Instruction
		if err := json.Unmarshal(line, &instr); err != nil {
			continue
		}

		switch instr.Type {
		case "hello":
			// Controller pushes this immediately on connect and via heartbeat
			// response. Contains operational params only — interval, profile
			// name for logging. Ziti identity and role attributes are immutable
			// from the agent's perspective; the controller owns those.
			var cfg helloPayload
			if err := json.Unmarshal(instr.Payload, &cfg); err == nil {
				state.mu.Lock()
				state.hello = cfg
				state.hasHello = true
				state.mu.Unlock()
				if cfg.Interval > 0 {
					select {
					case intervalCh <- time.Duration(cfg.Interval) * time.Second:
					default:
					}
				}
			}
			// Signal that initial config has been received — unblocks telemetry start.
			select {
			case configReady <- struct{}{}:
			default:
			}

		case "config":
			// Full config replacement pushed by the controller. Replaces the
			// in-memory state and stays active until the next push. This is
			// the primary mechanism for updating agent configuration —
			// controller pushes a message, agent applies it immediately.
			var cfg helloPayload
			if err := json.Unmarshal(instr.Payload, &cfg); err == nil {
				state.mu.Lock()
				state.hello = cfg
				state.hasHello = true
				state.mu.Unlock()
				if cfg.Interval > 0 {
					select {
					case intervalCh <- time.Duration(cfg.Interval) * time.Second:
					default:
					}
				}
				logger.Info("config updated", "nickname", cfg.Nickname, "interval", cfg.Interval)
			}

		case "apply":
			// Deploy a new config profile. Pulls the bundle from git over
			// Ziti, writes the bao-agent config, delivers AppRole creds,
			// and restarts the bao-agent unit (which starts the app with
			// secrets injected as env vars — nothing on disk).
			go func() {
				result := handleApply(ctx, zitiCtx, instr.Payload, logger)
				if ev, err := encodeEvent(machineID, "apply", result); err == nil {
					select {
					case eventCh <- ev:
					default:
					}
				}
			}()

		case "set_interval":
			var p struct {
				Seconds int `json:"seconds"`
			}
			if err := json.Unmarshal(instr.Payload, &p); err == nil && p.Seconds > 0 {
				select {
				case intervalCh <- time.Duration(p.Seconds) * time.Second:
				default:
				}
			}

		case "reload":
			return
		}
	}
}

// -- Telemetry (NATS over Ziti) ----------------------------------------------

// runTelemetry starts all collectors and fans their output into a single NATS
// publisher on "tango.telemetry.<machineID>". Each collector runs independently
// in its own goroutine — net, logs, heartbeat — and sends as soon as data is
// ready. The NATS connection is restarted on failure; collectors keep running
// and buffering into the shared channel during the reconnect window.
//
// A local eventBuffer holds up to 5 minutes of events when NATS is unreachable.
// On reconnect the buffer is drained into NATS before resuming live publishing.
// JetStream publish is used when available for delivery acknowledgement.
func runTelemetryLoop(ctx context.Context, zitiCtx ziti.Context, boot BootConfig, machineID string, eventCh chan []byte, intervalCh chan time.Duration, logger *slog.Logger) {
	backoff := 5 * time.Second

	// Local buffer — holds events for up to 5 minutes when NATS is down.
	edgeBuf := newEventBuffer(5 * time.Minute)

	interval := boot.defaultInterval()
	for {
		if ctx.Err() != nil {
			return
		}

		nc, err := connectNATS(zitiCtx, boot.telemetrySvc())
		if err != nil {
			// NATS unavailable — buffer events locally and try HTTP fallback.
			if boot.FallbackURL != "" {
				fallbackTicker := time.NewTicker(interval)
			fallbackLoop:
				for {
					// Send heartbeat and flush any queued collector events.
					hb := buildHeartbeat(machineID)
					ev, _ := encodeEvent(machineID, "hb", hb)
					if cmd, _ := httpHeartbeat(ctx, boot.FallbackURL, machineID, ev); cmd != nil {
						handleCmd(cmd, intervalCh)
					}
					// Drain eventCh into the local buffer + HTTP (best-effort).
				drain:
					for {
						select {
						case payload := <-eventCh:
							edgeBuf.push(payload)
							httpHeartbeat(ctx, boot.FallbackURL, machineID, payload)
						default:
							break drain
						}
					}
					select {
					case <-ctx.Done():
						fallbackTicker.Stop()
						return
					case d := <-intervalCh:
						interval = d
						fallbackTicker.Reset(d)
					case <-fallbackTicker.C:
						// Try NATS again on each tick
						fallbackTicker.Stop()
						break fallbackLoop
					}
				}
			} else {
				// No fallback — just buffer locally.
				bufferFromChannel(eventCh, edgeBuf)
				select {
				case <-ctx.Done():
					return
				case <-time.After(backoff):
					if backoff < 5*time.Minute {
						backoff *= 2
					}
				}
			}
			continue
		}

		backoff = 5 * time.Second

		// Subscribe to config instructions on the same NATS connection.
		configSubject := "tango.config." + machineID
		configSub, subErr := nc.Subscribe(configSubject, func(msg *natsgo.Msg) {
			var instr Instruction
			if err := json.Unmarshal(msg.Data, &instr); err != nil {
				return
			}
			logger.Info("config received via NATS", "type", instr.Type)
			handleInstruction(instr, boot, machineID, zitiCtx, eventCh, intervalCh, logger)
		})
		if subErr != nil {
			logger.Warn("config subscription failed", "error", subErr)
		} else {
			logger.Info("subscribed to config", "subject", configSubject)
		}

		// Drain any buffered events from the edge buffer first.
		js, jsErr := nc.JetStream()
		if jsErr != nil {
			logger.Warn("jetstream unavailable, using core publish", "error", jsErr)
		}
		drainEdgeBuffer(edgeBuf, nc, js, machineID, logger)

		interval = publishEvents(ctx, nc, js, machineID, eventCh, edgeBuf, intervalCh, interval, logger)

		if configSub != nil {
			configSub.Unsubscribe()
		}
		nc.Close()

		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

// drainEdgeBuffer replays buffered events into NATS after a reconnect.
func drainEdgeBuffer(buf *eventBuffer, nc *natsgo.Conn, js natsgo.JetStreamContext, machineID string, logger *slog.Logger) {
	events := buf.drain()
	if len(events) == 0 {
		return
	}
	logger.Info("draining edge buffer", "count", len(events))
	for _, payload := range events {
		slug, data, err := decodePulseMessage(payload)
		if err != nil {
			slug = "unknown"
			data = payload
		}
		subject := "tango.telemetry." + machineID + "." + slug
		if js != nil {
			if _, err := js.Publish(subject, data); err != nil {
				nc.Publish(subject, data)
			}
		} else {
			nc.Publish(subject, data)
		}
	}
}

// bufferFromChannel drains any pending events from eventCh into the edge buffer.
func bufferFromChannel(eventCh <-chan []byte, buf *eventBuffer) {
	for {
		select {
		case payload := <-eventCh:
			buf.push(payload)
		default:
			return
		}
	}
}

// connectNATS dials the NATS service through Ziti and returns a connected
// *nats.Conn. The Ziti conn is used as the custom dialer — NATS sees a
// normal net.Conn underneath.
func connectNATS(zitiCtx ziti.Context, serviceName string) (*natsgo.Conn, error) {
	return natsgo.Connect("nats://nats.tango",
		natsgo.SetCustomDialer(&zitiDialer{ctx: zitiCtx, service: serviceName}),
		natsgo.MaxReconnects(0), // we handle reconnects ourselves
		natsgo.Timeout(10*time.Second),
	)
}

// zitiDialer implements nats.CustomDialer — dials through the Ziti overlay.
type zitiDialer struct {
	ctx     ziti.Context
	service string
}

func (d *zitiDialer) Dial(_, _ string) (net.Conn, error) {
	return d.ctx.Dial(d.service)
}

// publishEvents drains eventCh and publishes each pulse to NATS.
// Each pulse message is framed as [slug length][slug][msgpack data].
// The NATS subject is built from the machine ID and slug:
//   tango.telemetry.<machineID>.<slug>
func publishEvents(ctx context.Context, nc *natsgo.Conn, js natsgo.JetStreamContext, machineID string, eventCh <-chan []byte, edgeBuf *eventBuffer, intervalCh <-chan time.Duration, interval time.Duration, logger *slog.Logger) time.Duration {
	for {
		select {
		case <-ctx.Done():
			return interval
		case d := <-intervalCh:
			interval = d
		case payload, ok := <-eventCh:
			if !ok {
				return interval
			}
			if !nc.IsConnected() {
				edgeBuf.push(payload)
				return interval
			}

			// Extract slug from the framed pulse message to build the subject.
			slug, data, err := decodePulseMessage(payload)
			if err != nil {
				// Legacy format — publish to base subject
				slug = "unknown"
				data = payload
			}
			subject := "tango.telemetry." + machineID + "." + slug

			if js != nil {
				if _, err := js.Publish(subject, data); err != nil {
					edgeBuf.push(payload)
					if !nc.IsConnected() {
						return interval
					}
				}
			} else {
				nc.Publish(subject, data)
			}
		}
	}
}

// -- Fallback ----------------------------------------------------------------
// HTTP fallback is now handled by runFallbackUntilOverlay in the main Run loop.

// handleCmd acts on a command returned in a heartbeat response.
func handleCmd(cmd *HeartbeatCmd, intervalCh chan<- time.Duration) {
	switch cmd.Cmd {
	case "hello":
		var cfg helloPayload
		if err := json.Unmarshal(cmd.Payload, &cfg); err == nil {
			state.mu.Lock()
			state.hello = cfg
			state.hasHello = true
			state.mu.Unlock()
			if cfg.Interval > 0 {
				select {
				case intervalCh <- time.Duration(cfg.Interval) * time.Second:
				default:
				}
			}
		}
	case "enroll":
		if JoinFunc == nil {
			return
		}
		var p struct {
			URL      string `json:"url"`
			Token    string `json:"token"`
			RoleID   string `json:"role_id"`
			SecretID string `json:"secret_id"`
		}
		if err := json.Unmarshal(cmd.Payload, &p); err != nil || p.URL == "" {
			return
		}
		go JoinFunc(p.URL, p.Token, p.RoleID, p.SecretID)
	}
}

// JoinFunc is called when the agent receives an enroll command. Injected at
// startup so the agent package doesn't import the enroll command package.
var JoinFunc func(url, session, roleID, secretID string) error

// HeartbeatCmd is the command returned by the controller in a heartbeat response.
type HeartbeatCmd struct {
	Cmd     string          `json:"cmd"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

func httpHeartbeat(ctx context.Context, baseURL, machineID string, payload []byte) (*HeartbeatCmd, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/heartbeat", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Machine-ID", machineID)
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var cmd HeartbeatCmd
	if err := json.NewDecoder(resp.Body).Decode(&cmd); err != nil {
		return nil, nil
	}
	return &cmd, nil
}

// -- Payload -----------------------------------------------------------------

func buildHeartbeat(machineID string) Heartbeat {
	h, _ := os.Hostname()
	hb := Heartbeat{
		MachineID:  machineID,
		Hostname:   h,
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		UptimeSecs: uptimeSeconds(),
		CPUCores:   runtime.NumCPU(),
		MemoryMB:   memoryMB(),
		LoadAvg1:   loadAvg1(),
		Timestamp:  time.Now().Unix(),
	}
	state.mu.Lock()
	if state.hasHello {
		hb.Nickname = state.hello.Nickname
		hb.State = state.hello.State
		hb.Profile = state.hello.Profile
	}
	state.mu.Unlock()
	return hb
}

// -- Helpers -----------------------------------------------------------------

func loadMachineID(identityPath string) (string, error) {
	data, err := os.ReadFile(filepath.Join(filepath.Dir(identityPath), "machine.json"))
	if err != nil {
		return "", err
	}
	var rec struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &rec); err != nil || rec.ID == "" {
		return "", fmt.Errorf("id missing from machine.json")
	}
	return rec.ID, nil
}

func defaultIdentityPath() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("ProgramData"), "kontango", "identity.json")
	case "darwin":
		return "/usr/local/kontango/identity.json"
	default:
		return "/opt/kontango/identity.json"
	}
}
