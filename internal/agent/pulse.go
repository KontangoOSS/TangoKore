// Package agent — pulse.go
//
// The pulse system is how machines and applications report telemetry.
//
// Wire format: MessagePack-encoded array of values.
// NATS subject: tango.telemetry.<machineID>.<slug>
//
// System pulses use a fixed key registry — position in the array defines
// the metric. Application pulses use msgpack map[string]string for flexibility.
//
// Size on the wire:
//   System heartbeat (7 fields): 33 bytes
//   App health check (3 fields): ~30 bytes
//   Single status pulse: 9 bytes
//
// Encode speed: 2.8M+ messages/sec on a single core.

package agent

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

// KV is a flat map for application pulses. Schema-free.
type KV map[string]string

// System key registry — position in the array defines the metric.
// Both agent and controller use the same registry.
// Changing the order is a breaking change — only append new fields.
const (
	SysHostname = iota // 0
	SysOS              // 1
	SysArch            // 2
	SysCPUs            // 3
	SysLoad            // 4: load * 100 as uint16 (2 decimal places)
	SysMemMB           // 5
	SysUptime          // 6
	SysNick            // 7
	SysState           // 8
	SysProfile         // 9
)

// encodeSystemPulse packs system metrics as a msgpack array.
// 33 bytes on the wire for 7 fields. 58% smaller than map[string]string.
func encodeSystemPulse() ([]byte, error) {
	h, _ := os.Hostname()
	loadVal := uint16(loadAvg1() * 100)
	memMB := uint32(memoryMB())
	uptime := uint32(uptimeSeconds())

	// Build array — use typed values for compactness
	arr := make([]interface{}, 10)
	arr[SysHostname] = h
	arr[SysOS] = runtime.GOOS
	arr[SysArch] = runtime.GOARCH
	arr[SysCPUs] = uint8(runtime.NumCPU())
	arr[SysLoad] = loadVal
	arr[SysMemMB] = memMB
	arr[SysUptime] = uptime

	state.mu.Lock()
	if state.hasHello {
		arr[SysNick] = state.hello.Nickname
		arr[SysState] = state.hello.State
		arr[SysProfile] = state.hello.Profile
	} else {
		arr[SysNick] = ""
		arr[SysState] = ""
		arr[SysProfile] = ""
	}
	state.mu.Unlock()

	return msgpack.Marshal(arr)
}

// encodeAppPulse packs application KV as a msgpack map.
// Apps are schema-free — use string keys for flexibility.
func encodeAppPulse(kv KV) ([]byte, error) {
	return msgpack.Marshal(kv)
}

// decodeKV deserialises a msgpack payload. Handles both arrays (system)
// and maps (application).
func decodeKV(data []byte) (KV, error) {
	// Try map first (app pulses)
	var kv KV
	if err := msgpack.Unmarshal(data, &kv); err == nil {
		return kv, nil
	}
	// Try array (system pulses)
	var arr []interface{}
	if err := msgpack.Unmarshal(data, &arr); err != nil {
		return nil, err
	}
	return decodeSystemArray(arr), nil
}

// decodeSystemArray converts the positional array back to a KV map.
func decodeSystemArray(arr []interface{}) KV {
	kv := make(KV)
	get := func(i int) string {
		if i >= len(arr) || arr[i] == nil {
			return ""
		}
		return fmt.Sprintf("%v", arr[i])
	}
	if v := get(SysHostname); v != "" { kv["hostname"] = v }
	if v := get(SysOS); v != "" { kv["os"] = v }
	if v := get(SysArch); v != "" { kv["arch"] = v }
	if v := get(SysCPUs); v != "" { kv["cpus"] = v }
	if v := get(SysLoad); v != "" {
		// Decode load * 100 back to float string
		if n, ok := arr[SysLoad].(uint16); ok {
			kv["load"] = fmt.Sprintf("%.2f", float64(n)/100)
		} else if n, ok := arr[SysLoad].(uint64); ok {
			kv["load"] = fmt.Sprintf("%.2f", float64(n)/100)
		} else {
			kv["load"] = v
		}
	}
	if v := get(SysMemMB); v != "" { kv["mem_mb"] = v }
	if v := get(SysUptime); v != "" { kv["up"] = v }
	if v := get(SysNick); v != "" { kv["nick"] = v }
	if v := get(SysState); v != "" { kv["state"] = v }
	if v := get(SysProfile); v != "" { kv["profile"] = v }
	return kv
}

// pulseMessage framing for the event channel.
// [1 byte slug len][slug][payload]
type pulseMessage struct{}

func encodePulseMessage(slug string, data []byte) ([]byte, error) {
	slugBytes := []byte(slug)
	if len(slugBytes) > 255 {
		slugBytes = slugBytes[:255]
	}
	buf := make([]byte, 1+len(slugBytes)+len(data))
	buf[0] = byte(len(slugBytes))
	copy(buf[1:], slugBytes)
	copy(buf[1+len(slugBytes):], data)
	return buf, nil
}

func decodePulseMessage(raw []byte) (slug string, data []byte, err error) {
	if len(raw) < 2 {
		return "", nil, fmt.Errorf("too short")
	}
	sLen := int(raw[0])
	if 1+sLen > len(raw) {
		return "", nil, fmt.Errorf("invalid slug length")
	}
	return string(raw[1 : 1+sLen]), raw[1+sLen:], nil
}

// pulseCollector emits system pulses as compact msgpack arrays.
type pulseCollector struct {
	intervalCh <-chan time.Duration
	initial    time.Duration
}

func (c *pulseCollector) collect(ctx interface{ Done() <-chan struct{} }, machineID string, out chan<- []byte) {
	interval := c.initial
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	emit := func() {
		data, err := encodeSystemPulse()
		if err != nil {
			return
		}
		b, err := encodePulseMessage("system", data)
		if err != nil {
			return
		}
		select {
		case out <- b:
		default:
		}
	}

	emit()
	for {
		select {
		case <-ctx.Done():
			return
		case d := <-c.intervalCh:
			interval = d
			ticker.Reset(d)
		case <-ticker.C:
			emit()
		}
	}
}
