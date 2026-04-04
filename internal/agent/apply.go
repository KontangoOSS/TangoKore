package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/openziti/sdk-golang/ziti"
)

// ApplyPayload is the instruction sent by the controller to deploy a new
// config profile to a machine. The agent pulls the config bundle from the
// source service, verifies the checksum, writes the bao-agent config,
// delivers the secret_id, and restarts the bao-agent unit.
type ApplyPayload struct {
	// Profile is the name of the config profile (e.g. "web-prod").
	Profile string `json:"profile"`

	// Version is the version tag of the config bundle (e.g. "v1.4.2").
	Version string `json:"version"`

	// Source is the Ziti service + path to fetch the config bundle archive.
	// Example: "configs.tango/web-prod/v1.4.2.tar.gz"
	Source string `json:"source"`

	// Checksum is the expected SHA-256 of the downloaded archive.
	Checksum string `json:"checksum"`

	// SecretID is a one-time AppRole secret_id for Bao authentication.
	// Burned after first use by the Bao agent.
	SecretID string `json:"secret_id,omitempty"`

	// RoleID is the AppRole role_id for this profile. Written to disk
	// so the Bao agent can re-authenticate on restart.
	RoleID string `json:"role_id,omitempty"`
}

// ApplyResult is reported back via telemetry after an apply.
type ApplyResult struct {
	Profile string `json:"profile"`
	Version string `json:"version"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

const kontangoDir = "/opt/kontango"

// handleApply processes an "apply" instruction from the controller.
// It pulls the config bundle, verifies it, writes Bao agent credentials,
// and restarts the bao-agent systemd unit which in turn starts the app
// with secrets injected as environment variables.
func handleApply(ctx context.Context, zitiCtx ziti.Context, payload json.RawMessage, logger *slog.Logger) ApplyResult {
	var ap ApplyPayload
	if err := json.Unmarshal(payload, &ap); err != nil {
		return ApplyResult{Error: fmt.Sprintf("unmarshal: %v", err)}
	}

	result := ApplyResult{Profile: ap.Profile, Version: ap.Version}
	logger = logger.With("profile", ap.Profile, "version", ap.Version)

	// 1. Pull config bundle from source over Ziti.
	archive, err := fetchBundle(ctx, zitiCtx, ap.Source)
	if err != nil {
		result.Error = fmt.Sprintf("fetch: %v", err)
		logger.Error("apply: fetch failed", "error", err)
		return result
	}

	// 2. Verify checksum.
	if ap.Checksum != "" {
		actual := sha256sum(archive)
		if actual != ap.Checksum {
			result.Error = fmt.Sprintf("checksum mismatch: expected %s, got %s", ap.Checksum, actual)
			logger.Error("apply: checksum mismatch", "expected", ap.Checksum, "actual", actual)
			return result
		}
	}

	// 3. Extract the archive to a staging dir, then move into place.
	stageDir := filepath.Join(kontangoDir, ".stage-"+ap.Profile)
	os.RemoveAll(stageDir)
	if err := extractTarGz(archive, stageDir); err != nil {
		result.Error = fmt.Sprintf("extract: %v", err)
		logger.Error("apply: extract failed", "error", err)
		return result
	}
	defer os.RemoveAll(stageDir)

	// 4. Copy extracted files to /opt/kontango/ (bao-agent.hcl, templates, etc).
	profileDir := filepath.Join(kontangoDir, "profiles", ap.Profile)
	os.MkdirAll(profileDir, 0755)
	if err := copyDir(stageDir, profileDir); err != nil {
		result.Error = fmt.Sprintf("install: %v", err)
		logger.Error("apply: install failed", "error", err)
		return result
	}

	// 5. Write AppRole credentials for the Bao agent.
	if ap.RoleID != "" {
		writeFile(filepath.Join(kontangoDir, "role-id"), []byte(ap.RoleID), 0600)
	}
	if ap.SecretID != "" {
		writeFile(filepath.Join(kontangoDir, "secret-id"), []byte(ap.SecretID), 0600)
	}

	// 6. Symlink the active profile's bao-agent.hcl.
	agentHCL := filepath.Join(profileDir, "bao-agent.hcl")
	if _, err := os.Stat(agentHCL); err == nil {
		activeLink := filepath.Join(kontangoDir, "bao-agent.hcl")
		os.Remove(activeLink)
		os.Symlink(agentHCL, activeLink)
	}

	// 7. Restart the bao-agent systemd unit.
	if err := restartUnit("kontango-bao-agent"); err != nil {
		result.Error = fmt.Sprintf("restart: %v", err)
		logger.Error("apply: restart failed", "error", err)
		return result
	}

	logger.Info("apply: success")
	result.Success = true
	return result
}

// fetchBundle downloads a config bundle from a Ziti service via HTTP.
// The source format is "service-name/path" — the agent dials the service
// through the Ziti overlay and makes an HTTP GET using the SDK as transport.
func fetchBundle(ctx context.Context, zitiCtx ziti.Context, source string) ([]byte, error) {
	serviceName := source[:indexOf(source, '/')]
	path := source[indexOf(source, '/'):]

	conn, err := zitiCtx.Dial(serviceName)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", serviceName, err)
	}
	defer conn.Close()

	// Write an HTTP/1.1 GET request over the Ziti connection.
	req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", path, serviceName)
	if _, err := conn.Write([]byte(req)); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	data, err := io.ReadAll(conn)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	body := stripHTTPHeaders(data)
	if body == nil {
		return nil, fmt.Errorf("malformed HTTP response")
	}

	return body, nil
}

func stripHTTPHeaders(data []byte) []byte {
	for i := 0; i < len(data)-3; i++ {
		if data[i] == '\r' && data[i+1] == '\n' && data[i+2] == '\r' && data[i+3] == '\n' {
			return data[i+4:]
		}
	}
	return nil
}

func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return len(s)
}

func sha256sum(data []byte) string {
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:])
}

func writeFile(path string, data []byte, perm os.FileMode) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func restartUnit(unit string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "systemctl", "restart", unit+".service").Run()
}

// extractTarGz extracts a tar.gz archive into destDir.
func extractTarGz(data []byte, destDir string) error {
	os.MkdirAll(destDir, 0755)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "tar", "-xzf", "-", "-C", destDir)
	cmd.Stdin = bytesReader(data)
	return cmd.Run()
}

type bytesReaderWrapper struct{ *io.SectionReader }

func bytesReader(data []byte) io.Reader {
	return io.NewSectionReader(readerAt(data), 0, int64(len(data)))
}

type readerAt []byte

func (r readerAt) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(r)) {
		return 0, io.EOF
	}
	n := copy(p, r[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// copyDir copies all files from src to dst, preserving directory structure.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}
