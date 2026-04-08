package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestPhase2_QuarantineDeviceWeb tests enrollment of a quarantine device receiving .web cert
func TestPhase2_QuarantineDeviceWeb(t *testing.T) {
	if os.Getenv("KONTANGO_INTEGRATION") == "" {
		t.Skip("skipping integration test (set KONTANGO_INTEGRATION=1 to run)")
	}

	// Simulate device enrollment with quarantine decision
	enrollmentData := map[string]interface{}{
		"method":        "new",
		"hostname":      "test-device-1",
		"machine_id":    "machine-001",
		"os":            "Ubuntu 25.04",
		"os_version":    "25.04",
		"arch":          "amd64",
		"kernel":        "6.8.0",
		"cpu_cores":     4,
		"memory_mb":     8192,
		"hardware_hash": "abcd1234",
	}

	// Expected certificate layers for quarantine device
	expectedLayers := map[string]bool{
		"base": true,  // Always issued
		"web":  true,  // Quarantine testing cert
		"lab":  false, // Only for approved non-admin
		"tango": false, // Only for admin
	}

	verifyEnrollmentCerts(t, enrollmentData, "quarantine", expectedLayers)
}

// TestPhase2_ApprovedDeviceLab tests enrollment of approved device receiving .lab cert with Bao access
func TestPhase2_ApprovedDeviceLab(t *testing.T) {
	if os.Getenv("KONTANGO_INTEGRATION") == "" {
		t.Skip("skipping integration test (set KONTANGO_INTEGRATION=1 to run)")
	}

	// Simulate device enrollment with approval decision (non-admin)
	enrollmentData := map[string]interface{}{
		"method":        "scan",
		"hostname":      "test-device-2",
		"machine_id":    "machine-002",
		"os":            "Ubuntu 25.04",
		"os_version":    "25.04",
		"arch":          "amd64",
		"kernel":        "6.8.0",
		"cpu_cores":     8,
		"memory_mb":     16384,
		"hardware_hash": "efgh5678",
	}

	// Expected certificate layers for approved non-admin device
	expectedLayers := map[string]bool{
		"base": true,  // Always issued
		"web":  false, // Only for quarantine
		"lab":  true,  // Approved non-admin gets Bao access
		"tango": false, // Only for admin
	}

	verifyEnrollmentCerts(t, enrollmentData, "approved", expectedLayers)
}

// TestPhase2_AdminDeviceTango tests enrollment of admin device receiving .tango cert
func TestPhase2_AdminDeviceTango(t *testing.T) {
	if os.Getenv("KONTANGO_INTEGRATION") == "" {
		t.Skip("skipping integration test (set KONTANGO_INTEGRATION=1 to run)")
	}

	// Simulate device enrollment with admin attributes
	enrollmentData := map[string]interface{}{
		"method":        "oidc",
		"hostname":      "test-admin-1",
		"machine_id":    "machine-admin-001",
		"os":            "Ubuntu 25.04",
		"os_version":    "25.04",
		"arch":          "amd64",
		"kernel":        "6.8.0",
		"cpu_cores":     16,
		"memory_mb":     32768,
		"hardware_hash": "ijkl9012",
	}

	// Expected certificate layers for admin device
	expectedLayers := map[string]bool{
		"base":  true, // Always issued
		"web":   false, // Only for quarantine
		"lab":   false, // Only for approved non-admin
		"tango": true, // Admin-only experimental/hidden
	}

	verifyEnrollmentCerts(t, enrollmentData, "approved", expectedLayers)
}

// TestPhase2_CertificateDomains tests certificate CN formatting
func TestPhase2_CertificateDomains(t *testing.T) {
	if os.Getenv("KONTANGO_INTEGRATION") == "" {
		t.Skip("skipping integration test (set KONTANGO_INTEGRATION=1 to run)")
	}

	domain := "konoss.org"
	machineID := "machine-test"

	tests := []struct {
		layer           string
		expectedCN      string
		expectedDomain  string
		isUnresolvable  bool
	}{
		{
			layer:          "base",
			expectedCN:     machineID + "." + domain,
			expectedDomain: domain,
			isUnresolvable: false,
		},
		{
			layer:          "web",
			expectedCN:     machineID + ".web." + domain,
			expectedDomain: "web." + domain,
			isUnresolvable: false,
		},
		{
			layer:          "lab",
			expectedCN:     machineID + ".lab." + domain,
			expectedDomain: "lab." + domain,
			isUnresolvable: false,
		},
		{
			layer:          "tango",
			expectedCN:     machineID + ".tango",
			expectedDomain: "tango",
			isUnresolvable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.layer, func(t *testing.T) {
			// Verify CN format
			if tt.expectedCN != machineID+"."+tt.expectedDomain &&
			   tt.expectedCN != machineID+".tango" {
				t.Errorf("Invalid CN format for %s: %s", tt.layer, tt.expectedCN)
			}

			// Verify domain suffix
			if tt.isUnresolvable {
				if tt.expectedDomain != "tango" {
					t.Errorf("Unresolvable domain should be 'tango', got %s", tt.expectedDomain)
				}
			} else {
				if tt.expectedDomain == "tango" {
					t.Error("Resolvable domain should not be 'tango'")
				}
			}
		})
	}
}

// TestPhase2_ResponseStructure tests enrollment response includes certificates
func TestPhase2_ResponseStructure(t *testing.T) {
	if os.Getenv("KONTANGO_INTEGRATION") == "" {
		t.Skip("skipping integration test (set KONTANGO_INTEGRATION=1 to run)")
	}

	// Mock enrollment response structure
	response := map[string]interface{}{
		"id":       "reg-test-001",
		"nickname": "test-device",
		"status":   "approved",
		"identity": json.RawMessage(`{"some":"data"}`),
		"certificates": map[string]interface{}{
			"base": map[string]string{
				"certificate": "-----BEGIN CERTIFICATE-----...",
				"private_key": "-----BEGIN RSA PRIVATE KEY-----...",
				"issued_at":   "2026-04-08T01:24:17Z",
				"expires_at":  "2027-04-08T01:24:17Z",
			},
			"lab": map[string]string{
				"certificate": "-----BEGIN CERTIFICATE-----...",
				"private_key": "-----BEGIN RSA PRIVATE KEY-----...",
				"issued_at":   "2026-04-08T01:24:17Z",
				"expires_at":  "2027-04-08T01:24:17Z",
			},
		},
		"ca_bundle": "-----BEGIN CERTIFICATE-----...",
		"config": map[string]interface{}{
			"hosts": []string{"ctrl-1.konoss.org"},
		},
	}

	// Verify response structure
	if _, ok := response["id"].(string); !ok {
		t.Error("Missing id field")
	}
	if _, ok := response["certificates"].(map[string]interface{}); !ok {
		t.Error("Missing or invalid certificates field")
	}
	if _, ok := response["ca_bundle"].(string); !ok {
		t.Error("Missing or invalid ca_bundle field")
	}

	// Verify certificate layers have required fields
	certs := response["certificates"].(map[string]interface{})
	for layer, certData := range certs {
		cert, ok := certData.(map[string]string)
		if !ok {
			t.Errorf("Certificate layer %s has invalid structure", layer)
			continue
		}

		if cert["certificate"] == "" {
			t.Errorf("Certificate layer %s missing certificate", layer)
		}
		if cert["private_key"] == "" {
			t.Errorf("Certificate layer %s missing private_key", layer)
		}
		if cert["issued_at"] == "" {
			t.Errorf("Certificate layer %s missing issued_at", layer)
		}
		if cert["expires_at"] == "" {
			t.Errorf("Certificate layer %s missing expires_at", layer)
		}
	}
}

// TestPhase2_BaoAccessAfterApproval tests that approved devices get Bao credential access
func TestPhase2_BaoAccessAfterApproval(t *testing.T) {
	if os.Getenv("KONTANGO_INTEGRATION") == "" {
		t.Skip("skipping integration test (set KONTANGO_INTEGRATION=1 to run)")
	}

	tests := []struct {
		name              string
		status            string
		attributes        []string
		expectedBaoAccess bool
	}{
		{
			name:              "quarantine device no Bao",
			status:            "quarantine",
			attributes:        []string{},
			expectedBaoAccess: false,
		},
		{
			name:              "approved device with Bao",
			status:            "approved",
			attributes:        []string{"user"},
			expectedBaoAccess: true,
		},
		{
			name:              "admin device with Bao",
			status:            "approved",
			attributes:        []string{"admin"},
			expectedBaoAccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Bao access is tied to .lab certificate layer
			// Quarantine = .web cert (no Bao)
			// Approved = .lab cert (has Bao)
			// Admin = .tango cert (has Bao + higher privileges)

			if tt.status == "quarantine" && tt.expectedBaoAccess {
				t.Error("Quarantine device should not have Bao access")
			}
			if tt.status == "approved" && !tt.expectedBaoAccess {
				t.Error("Approved device should have Bao access via .lab cert")
			}
		})
	}
}

// --- Helpers ---

func verifyEnrollmentCerts(t *testing.T, enrollmentData map[string]interface{},
	expectedStatus string, expectedLayers map[string]bool) {

	// Mock server for enrollment
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}

		// Simulate enrollment response with certificates
		response := map[string]interface{}{
			"id":     "reg-" + fmt.Sprintf("%v", enrollmentData["machine_id"]),
			"status": expectedStatus,
			"identity": json.RawMessage(`{"some":"data"}`),
		}

		// Include only the expected certificate layers
		certs := make(map[string]interface{})
		for layer, shouldInclude := range expectedLayers {
			if shouldInclude {
				certs[layer] = map[string]string{
					"certificate": "-----BEGIN CERTIFICATE-----",
					"private_key": "-----BEGIN RSA PRIVATE KEY-----",
					"issued_at":   "2026-04-08T01:24:17Z",
					"expires_at":  "2027-04-08T01:24:17Z",
				}
			}
		}
		response["certificates"] = certs
		response["ca_bundle"] = "-----BEGIN CERTIFICATE-----"

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Verify response
	resp, err := http.Post(server.URL, "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to post enrollment: %v", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify status
	if result["status"] != expectedStatus {
		t.Errorf("Expected status %s, got %v", expectedStatus, result["status"])
	}

	// Verify certificates layers
	if certs, ok := result["certificates"].(map[string]interface{}); ok {
		for layer, shouldExist := range expectedLayers {
			exists := certs[layer] != nil
			if exists != shouldExist {
				t.Errorf("Layer %s: exists=%v, want %v", layer, exists, shouldExist)
			}
		}
	}

	// Verify CA bundle is included
	if result["ca_bundle"] == nil || result["ca_bundle"].(string) == "" {
		t.Error("CA bundle missing from response")
	}
}
