// Package agent — leaf.go
//
// Embeds a NATS leaf node in the kontango agent. Apps on the machine
// connect to localhost:4222 (or unix socket). The leaf syncs upstream
// to the central NATS hub on the controller through the Ziti overlay.
//
// This is the machine's local API. Any process can publish/subscribe.
// If the upstream is down, the leaf buffers locally via JetStream.
// When connectivity returns, it syncs automatically.

package agent

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/openziti/sdk-golang/ziti"
)

// LeafNode is an embedded NATS leaf server.
type LeafNode struct {
	srv      *server.Server
	nc       *nats.Conn // in-process client for the agent
	port     int
	listener net.Listener
}

// LeafOpts configures the embedded leaf node.
type LeafOpts struct {
	// ListenAddr for local clients. Default: "127.0.0.1:4222"
	ListenAddr string

	// StoreDir for JetStream persistence. Default: /opt/kontango/nats
	StoreDir string

	// MaxAge for local JetStream retention. Default: 5 minutes.
	MaxAge time.Duration

	// UpstreamService is the Ziti service name for the central NATS hub.
	// Default: "nats.tango"
	UpstreamService string
}

// StartLeaf starts an embedded NATS leaf node.
// It listens on a local port for apps and connects upstream to the
// controller's NATS hub through the Ziti overlay.
func StartLeaf(zitiCtx ziti.Context, opts *LeafOpts, logger *slog.Logger) (*LeafNode, error) {
	if opts == nil {
		opts = &LeafOpts{}
	}
	if opts.ListenAddr == "" {
		opts.ListenAddr = "127.0.0.1:4222"
	}
	if opts.StoreDir == "" {
		opts.StoreDir = "/opt/kontango/nats"
	}
	if opts.MaxAge == 0 {
		opts.MaxAge = 5 * time.Minute
	}
	if opts.UpstreamService == "" {
		opts.UpstreamService = "nats.tango"
	}

	os.MkdirAll(opts.StoreDir, 0755)

	// Start the Ziti bridge listener — the leaf connects upstream through this.
	// We create a local TCP listener, bridge it to the Ziti service, and tell
	// the NATS leaf to use it as its remote URL.
	bridgePort, bridgeLn, err := startBridge(zitiCtx, opts.UpstreamService)
	if err != nil {
		return nil, fmt.Errorf("bridge to %s: %w", opts.UpstreamService, err)
	}

	// Parse listen address
	host, port, _ := net.SplitHostPort(opts.ListenAddr)
	portNum := 4222
	if port != "" {
		fmt.Sscanf(port, "%d", &portNum)
	}
	if host == "" {
		host = "127.0.0.1"
	}

	srvOpts := &server.Options{
		Host:   host,
		Port:   portNum,
		NoLog:  true,
		NoSigs: true,

		// No auth — local machine only.

		// JetStream — local persistence for offline buffering.
		JetStream: true,
		StoreDir:  opts.StoreDir,

		// Leaf node — connect upstream to the hub through the Ziti bridge.
		LeafNode: server.LeafNodeOpts{
			Remotes: []*server.RemoteLeafOpts{
				{
					URLs: []*url.URL{parseURL(fmt.Sprintf("nats://127.0.0.1:%d", bridgePort))},
				},
			},
		},
	}

	srv, err := server.NewServer(srvOpts)
	if err != nil {
		bridgeLn.Close()
		return nil, fmt.Errorf("nats leaf: %w", err)
	}

	go srv.Start()
	if !srv.ReadyForConnections(10e9) {
		srv.Shutdown()
		bridgeLn.Close()
		return nil, fmt.Errorf("nats leaf did not start")
	}

	// In-process client for the agent itself.
	nc, err := nats.Connect("", nats.InProcessServer(srv))
	if err != nil {
		srv.Shutdown()
		bridgeLn.Close()
		return nil, fmt.Errorf("nats leaf client: %w", err)
	}

	logger.Info("nats leaf started",
		"listen", opts.ListenAddr,
		"upstream", opts.UpstreamService,
		"store", opts.StoreDir,
	)

	return &LeafNode{
		srv:      srv,
		nc:       nc,
		port:     portNum,
		listener: bridgeLn,
	}, nil
}

// Conn returns the in-process NATS client.
func (l *LeafNode) Conn() *nats.Conn { return l.nc }

// Shutdown stops the leaf node cleanly.
func (l *LeafNode) Shutdown() {
	if l.nc != nil {
		l.nc.Close()
	}
	if l.srv != nil {
		l.srv.Shutdown()
	}
	if l.listener != nil {
		l.listener.Close()
	}
}

// startBridge creates a local TCP listener that bridges connections to
// a Ziti service. Returns the local port and the listener.
func startBridge(zitiCtx ziti.Context, serviceName string) (int, net.Listener, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, nil, err
	}
	port := ln.Addr().(*net.TCPAddr).Port

	go func() {
		for {
			local, err := ln.Accept()
			if err != nil {
				return
			}
			go bridgeToZiti(local, zitiCtx, serviceName)
		}
	}()

	return port, ln, nil
}

func bridgeToZiti(local net.Conn, zitiCtx ziti.Context, service string) {
	defer local.Close()

	remote, err := zitiCtx.Dial(service)
	if err != nil {
		return
	}
	defer remote.Close()

	done := make(chan struct{}, 2)
	go func() { copyBytes(local, remote); done <- struct{}{} }()
	go func() { copyBytes(remote, local); done <- struct{}{} }()
	<-done
}

func copyBytes(dst, src net.Conn) {
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			dst.Write(buf[:n])
		}
		if err != nil {
			return
		}
	}
}

// parseURL parses a URL string, panics on error (only used for known-good URLs).
func parseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}
