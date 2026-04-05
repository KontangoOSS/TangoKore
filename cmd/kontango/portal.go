package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	iofs "io/fs"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

//go:embed web/*
var webFS embed.FS

// statusResponse is the public machine info returned by /api/status.
type statusResponse struct {
	Nickname  string `json:"nickname,omitempty"`
	Hostname  string `json:"hostname"`
	MachineID string `json:"machine_id,omitempty"`
	Mode      string `json:"mode,omitempty"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	Version   string `json:"version"`
	Uptime    int64  `json:"uptime_secs"`
	Status    string `json:"status"`
}

func cmdPortal(args []string) {
	flagSet := flag.NewFlagSet("portal", flag.ExitOnError)
	listen := flagSet.String("listen", ":8800", "portal listen address")
	identityPath := flagSet.String("identity", "/opt/kontango/identity.json", "path to identity for machine info")
	flagSet.Parse(args)

	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		hostname, _ := os.Hostname()
		resp := statusResponse{
			Hostname: hostname,
			OS:       runtime.GOOS,
			Arch:     runtime.GOARCH,
			Version:  version,
			Status:   "ok",
		}

		// Load machine info if available.
		if data, err := os.ReadFile(machineJSONPath(*identityPath)); err == nil {
			var rec struct {
				ID       string `json:"id"`
				Nickname string `json:"nickname"`
				Mode     string `json:"mode"`
			}
			if json.Unmarshal(data, &rec) == nil {
				resp.MachineID = rec.ID
				resp.Nickname = rec.Nickname
				resp.Mode = rec.Mode
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Static site — serve from embedded FS.
	webRoot, err := iofs.Sub(webFS, "web")
	if err != nil {
		fmt.Fprintf(os.Stderr, "portal: embedded fs: %v\n", err)
		os.Exit(1)
	}
	mux.Handle("/", http.FileServer(http.FS(webRoot)))

	srv := &http.Server{
		Addr:         *listen,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigCh; srv.Close() }()

	fmt.Printf("kontango portal on %s\n", *listen)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "portal: %v\n", err)
		os.Exit(1)
	}
}

func machineJSONPath(identityPath string) string {
	// machine.json lives next to identity.json
	dir := identityPath[:lastSlash(identityPath)]
	if dir == "" {
		dir = "."
	}
	return dir + "/machine.json"
}

func lastSlash(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return i
		}
	}
	return 0
}
