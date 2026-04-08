package controller

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// stepPKI generates PKI certificates using self-signed approach
func stepPKI(cfg *Config) error {
	log.Println("setting up PKI certificates...")

	// Create root CA
	rootCA, rootKey, err := createRootCA(cfg)
	if err != nil {
		return fmt.Errorf("create root CA: %w", err)
	}
	log.Println("  ✓ root CA created")

	// Create intermediate cert
	intermediateCert, intermediateKey, err := createIntermediate(cfg, rootCA, rootKey)
	if err != nil {
		return fmt.Errorf("create intermediate: %w", err)
	}
	log.Println("  ✓ intermediate certificate created")

	// Create server cert
	serverCert, serverKey, err := createServerCert(cfg, intermediateCert, intermediateKey)
	if err != nil {
		return fmt.Errorf("create server cert: %w", err)
	}
	log.Println("  ✓ server certificate created")

	// Write certs to disk
	if err := writePEMFile(filepath.Join(cfg.PKIDir, "root-ca.pem"), rootCA); err != nil {
		return fmt.Errorf("write root CA: %w", err)
	}

	if err := writePEMFile(filepath.Join(cfg.PKIDir, "intermediate.pem"), intermediateCert); err != nil {
		return fmt.Errorf("write intermediate: %w", err)
	}

	if err := writePEMFile(filepath.Join(cfg.PKIDir, "server.crt"), serverCert); err != nil {
		return fmt.Errorf("write server cert: %w", err)
	}

	if err := writePEMFile(filepath.Join(cfg.PKIDir, "server.key"), serverKey); err != nil {
		return fmt.Errorf("write server key: %w", err)
	}

	// Create CA bundle (root + intermediate)
	caBundle := append([]byte{}, rootCA...)
	caBundle = append(caBundle, intermediateCert...)
	if err := os.WriteFile(filepath.Join(cfg.PKIDir, "ca-bundle.pem"), caBundle, 0644); err != nil {
		return fmt.Errorf("write ca-bundle: %w", err)
	}
	log.Println("  ✓ CA bundle created")

	// Create PKI role definition files (for Bao or Ziti)
	if err := createPKIRoles(cfg); err != nil {
		return fmt.Errorf("create PKI roles: %w", err)
	}
	log.Println("  ✓ PKI roles defined")

	return nil
}

// createPKIRoles generates PKI role definitions for layer-based certificates
func createPKIRoles(cfg *Config) error {
	rolesFile := filepath.Join(cfg.PKIDir, "roles.json")

	roles := map[string]interface{}{
		"device-base": map[string]interface{}{
			"allowed_domains":   []string{cfg.Domain},
			"allow_subdomains":  true,
			"max_ttl":           "8760h",
			"organization":      []string{"Kontango"},
			"country":           []string{"US"},
			"description":       "Base device identity (1 year, external resolvable)",
		},
		"device-web": map[string]interface{}{
			"allowed_domains":   []string{"web." + cfg.Domain},
			"allow_subdomains":  true,
			"max_ttl":           "24h",
			"organization":      []string{"Kontango"},
			"country":           []string{"US"},
			"description":       "Web testing (24h, quarantine devices, internal Ziti DNS)",
		},
		"device-temp": map[string]interface{}{
			"allowed_domains":   []string{"temp." + cfg.Domain},
			"allow_subdomains":  true,
			"max_ttl":           "168h", // 7 days
			"organization":      []string{"Kontango"},
			"country":           []string{"US"},
			"description":       "Staging devices (7d, internal Ziti DNS)",
		},
		"device-lab": map[string]interface{}{
			"allowed_domains":   []string{"lab." + cfg.Domain},
			"allow_subdomains":  true,
			"max_ttl":           "8760h",
			"organization":      []string{"Kontango"},
			"country":           []string{"US"},
			"description":       "Public internal services (1yr, resolvable via Ziti DNS)",
		},
		"device-tango": map[string]interface{}{
			"allowed_domains":   []string{"tango"}, // NO domain suffix - unresolvable
			"allow_subdomains":  true,
			"max_ttl":           "8760h",
			"organization":      []string{"Kontango"},
			"country":           []string{"US"},
			"description":       "Production devices (1yr, unresolvable, Ziti DNS only)",
		},
		"device-admin": map[string]interface{}{
			"allowed_domains":   []string{"admin"}, // NO domain suffix - unresolvable
			"allow_subdomains":  true,
			"max_ttl":           "8760h",
			"organization":      []string{"Kontango"},
			"country":           []string{"US"},
			"description":       "Admin devices (1yr, unresolvable, Ziti DNS only)",
		},
	}

	rolesJSON, err := json.Marshal(roles)
	if err != nil {
		return fmt.Errorf("marshal roles: %w", err)
	}

	if err := os.WriteFile(rolesFile, rolesJSON, 0644); err != nil {
		return fmt.Errorf("write roles file: %w", err)
	}

	return nil
}

// createRootCA creates a self-signed root CA
func createRootCA(cfg *Config) ([]byte, *rsa.PrivateKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   cfg.Domain,
			Organization: []string{"Kontango"},
			Country:      []string{"US"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10 years
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	return certPEM, key, nil
}

// createIntermediate creates an intermediate certificate
func createIntermediate(cfg *Config, rootCertPEM []byte, rootKey *rsa.PrivateKey) ([]byte, *rsa.PrivateKey, error) {
	// Parse root cert
	block, _ := pem.Decode(rootCertPEM)
	if block == nil {
		return nil, nil, fmt.Errorf("failed to parse root cert")
	}

	rootCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, err
	}

	// Generate intermediate key
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	// Create intermediate template
	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName:   "intermediate." + cfg.Domain,
			Organization: []string{"Kontango"},
			Country:      []string{"US"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(5, 0, 0), // 5 years
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, rootCert, &key.PublicKey, rootKey)
	if err != nil {
		return nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	return certPEM, key, nil
}

// createServerCert creates a server certificate for the controller
func createServerCert(cfg *Config, caCertPEM []byte, caKey *rsa.PrivateKey) ([]byte, []byte, error) {
	// Parse CA cert
	block, _ := pem.Decode(caCertPEM)
	if block == nil {
		return nil, nil, fmt.Errorf("failed to parse CA cert")
	}

	caCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, err
	}

	// Generate server key
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	// Create server cert template
	template := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject: pkix.Name{
			CommonName:   cfg.Name + "." + cfg.Domain,
			Organization: []string{"Kontango"},
			Country:      []string{"US"},
		},
		NotBefore:   time.Now().Add(-1 * time.Hour),
		NotAfter:    time.Now().AddDate(1, 0, 0), // 1 year
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames: []string{
			cfg.Name + "." + cfg.Domain,
			"localhost",
			"127.0.0.1",
			cfg.Domain,
			"*." + cfg.Domain,
		},
	}

	// Add stage domains - always use the configured domain
	template.DNSNames = append(template.DNSNames,
		"*.quarantine."+cfg.Domain,
		"*.members."+cfg.Domain,
		"*.lab."+cfg.Domain,
		"*.admin."+cfg.Domain,
	)

	// Add IP address
	template.IPAddresses = append(template.IPAddresses, net.ParseIP("127.0.0.1"))
	if cfg.NodePublicIP != "" {
		if ip := net.ParseIP(cfg.NodePublicIP); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		}
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, caCert, &key.PublicKey, caKey)
	if err != nil {
		return nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	return certPEM, keyPEM, nil
}

// writePEMFile writes a PEM-encoded byte slice to a file
func writePEMFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0600)
}
