package agent

import (
	"encoding/json"
	"net"
	"net/http"
)

// pulseAPI runs a local HTTP server for applications to emit pulses.
//
// POST /pulse with JSON:
//
//	{"slug":"myorg/myapp","kv":{"status":"healthy","version":"2.1.0"}}
//
// The slug identifies the application. The kv map is the data.
// On the wire it becomes a msgpack payload on NATS subject
// tango.telemetry.<machineID>.<slug>
type pulseAPI struct {
	machineID string
	out       chan<- []byte
}

type pulseRequest struct {
	Slug string `json:"slug"` // e.g. "myorg/myapp"
	KV   KV     `json:"kv"`   // arbitrary key-value pairs
}

func startPulseAPI(machineID string, out chan<- []byte, listen string) (net.Listener, error) {
	if listen == "" {
		listen = "127.0.0.1:8801"
	}

	api := &pulseAPI{machineID: machineID, out: out}

	mux := http.NewServeMux()
	mux.HandleFunc("/pulse", api.handlePulse)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	ln, err := net.Listen("tcp", listen)
	if err != nil {
		return nil, err
	}

	go http.Serve(ln, mux)
	return ln, nil
}

func (a *pulseAPI) handlePulse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req pulseRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if req.Slug == "" || len(req.KV) == 0 {
		http.Error(w, "slug and kv required", http.StatusBadRequest)
		return
	}

	data, _ := encodeAppPulse(req.KV)
	if b, err := encodePulseMessage(req.Slug, data); err == nil {
		select {
		case a.out <- b:
		default:
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"accepted":true}`))
}
