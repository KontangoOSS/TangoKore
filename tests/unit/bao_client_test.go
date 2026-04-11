package unit_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KontangoOSS/TangoKore/internal/controller/clients"
)

// TestBaoClientInit verifies Bao initialization
func TestBaoClientInit(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/sys/init" && r.Method == "POST" {
			resp := map[string]interface{}{
				"keys":       []string{"key1"},
				"root_token": "test-token",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	client, _ := clients.NewBaoClient(server.URL, "", "")
	unsealKey, rootToken, err := client.Init(1, 1)

	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if rootToken != "test-token" {
		t.Errorf("got token %q, want test-token", rootToken)
	}
	if unsealKey != "key1" {
		t.Errorf("got key %q, want key1", unsealKey)
	}
}

// TestBaoClientCreatePKIRoleOnMount verifies role creation on arbitrary mount
func TestBaoClientCreatePKIRoleOnMount(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/pki_int/roles/device-base" && r.Method == "POST" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{})
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	client, _ := clients.NewBaoClient(server.URL, "test-token", "")
	err := client.CreatePKIRoleOnMount("pki_int", "device-base", []string{"test.com"}, "8760h")

	if err != nil {
		t.Fatalf("CreatePKIRoleOnMount failed: %v", err)
	}
}

// TestBaoClientPKIIssueCert verifies certificate issuance
func TestBaoClientPKIIssueCert(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/pki_int/issue/device-base" && r.Method == "POST" {
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"certificate": "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
					"private_key": "-----BEGIN PRIVATE KEY-----\nkey\n-----END PRIVATE KEY-----",
					"ca_chain":    "-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	client, _ := clients.NewBaoClient(server.URL, "test-token", "")
	cert, key, ca, err := client.PKIIssueCert("pki_int", "device-base", "test.com", "8760h", nil)

	if err != nil {
		t.Fatalf("PKIIssueCert failed: %v", err)
	}
	if cert == "" {
		t.Error("certificate is empty")
	}
	if key == "" {
		t.Error("private key is empty")
	}
	if ca == "" {
		t.Error("ca chain is empty")
	}
}
