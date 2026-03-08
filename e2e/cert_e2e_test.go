package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/azophy/sshifu/internal/cert"
	"golang.org/x/crypto/ssh"
)

// TestCertificateGenerationE2E tests end-to-end certificate generation
func TestCertificateGenerationE2E(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate CA
	caPrivPath := filepath.Join(tmpDir, "ca")
	caPubPath := filepath.Join(tmpDir, "ca.pub")

	if err := cert.GenerateCA(caPrivPath, caPubPath); err != nil {
		t.Fatalf("Failed to generate CA: %v", err)
	}

	// Verify CA files exist
	if _, err := os.Stat(caPrivPath); os.IsNotExist(err) {
		t.Error("CA private key should exist")
	}

	if _, err := os.Stat(caPubPath); os.IsNotExist(err) {
		t.Error("CA public key should exist")
	}

	// Load CA
	caSigner, err := cert.LoadCA(caPrivPath)
	if err != nil {
		t.Fatalf("Failed to load CA: %v", err)
	}

	// Generate user key
	userPrivPath := filepath.Join(tmpDir, "user_key")
	userPubPath := userPrivPath + ".pub"

	if err := cert.GenerateCA(userPrivPath, userPubPath); err != nil {
		t.Fatalf("Failed to generate user key: %v", err)
	}

	// Read user public key
	userPubBytes, err := os.ReadFile(userPubPath)
	if err != nil {
		t.Fatalf("Failed to read user public key: %v", err)
	}

	// Parse user public key
	userKey, _, _, _, err := ssh.ParseAuthorizedKey(userPubBytes)
	if err != nil {
		t.Fatalf("Failed to parse user public key: %v", err)
	}

	// Sign user certificate
	certBytes, err := cert.SignUserKey(
		caSigner.Signer(),
		userKey,
		"testuser",
		8*time.Hour,
		map[string]bool{
			"permit-pty":             true,
			"permit-port-forwarding": true,
		},
	)
	if err != nil {
		t.Fatalf("Failed to sign user certificate: %v", err)
	}

	// Verify certificate
	if len(certBytes) == 0 {
		t.Fatal("Expected non-empty certificate")
	}

	// Parse and verify certificate
	parsedKey, _, _, _, err := ssh.ParseAuthorizedKey(certBytes)
	if err != nil {
		t.Fatalf("Failed to parse signed certificate: %v", err)
	}

	cert, ok := parsedKey.(*ssh.Certificate)
	if !ok {
		t.Fatal("Parsed key is not a certificate")
	}

	if cert.CertType != ssh.UserCert {
		t.Errorf("Expected UserCert, got: %d", cert.CertType)
	}

	if len(cert.ValidPrincipals) != 1 || cert.ValidPrincipals[0] != "testuser" {
		t.Errorf("Expected principal 'testuser', got: %v", cert.ValidPrincipals)
	}

	t.Logf("User certificate generated successfully for: %v", cert.ValidPrincipals)
}

// TestHostCertificateGenerationE2E tests end-to-end host certificate generation
func TestHostCertificateGenerationE2E(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate CA
	caPrivPath := filepath.Join(tmpDir, "ca")
	caPubPath := filepath.Join(tmpDir, "ca.pub")

	if err := cert.GenerateCA(caPrivPath, caPubPath); err != nil {
		t.Fatalf("Failed to generate CA: %v", err)
	}

	caSigner, err := cert.LoadCA(caPrivPath)
	if err != nil {
		t.Fatalf("Failed to load CA: %v", err)
	}

	// Generate host key
	hostPrivPath := filepath.Join(tmpDir, "ssh_host_ed25519_key")
	hostPubPath := hostPrivPath + ".pub"

	if err := cert.GenerateCA(hostPrivPath, hostPubPath); err != nil {
		t.Fatalf("Failed to generate host key: %v", err)
	}

	// Read host public key
	hostPubBytes, err := os.ReadFile(hostPubPath)
	if err != nil {
		t.Fatalf("Failed to read host public key: %v", err)
	}

	// Parse host public key
	hostKey, _, _, _, err := ssh.ParseAuthorizedKey(hostPubBytes)
	if err != nil {
		t.Fatalf("Failed to parse host public key: %v", err)
	}

	// Sign host certificate
	principals := []string{"host.example.com", "192.168.1.1"}
	certBytes, err := cert.SignHostKey(
		caSigner.Signer(),
		hostKey,
		principals,
		720*time.Hour,
	)
	if err != nil {
		t.Fatalf("Failed to sign host certificate: %v", err)
	}

	// Verify certificate
	if len(certBytes) == 0 {
		t.Fatal("Expected non-empty certificate")
	}

	// Parse and verify certificate
	parsedKey, _, _, _, err := ssh.ParseAuthorizedKey(certBytes)
	if err != nil {
		t.Fatalf("Failed to parse signed host certificate: %v", err)
	}

	cert, ok := parsedKey.(*ssh.Certificate)
	if !ok {
		t.Fatal("Parsed key is not a certificate")
	}

	if cert.CertType != ssh.HostCert {
		t.Errorf("Expected HostCert, got: %d", cert.CertType)
	}

	if len(cert.ValidPrincipals) != len(principals) {
		t.Errorf("Expected %d principals, got: %d", len(principals), len(cert.ValidPrincipals))
	}

	t.Logf("Host certificate generated successfully with principals: %v", cert.ValidPrincipals)
}

// TestCAKeyFormatE2E tests that CA keys are in correct format
func TestCAKeyFormatE2E(t *testing.T) {
	tmpDir := t.TempDir()

	caPrivPath := filepath.Join(tmpDir, "ca")
	caPubPath := filepath.Join(tmpDir, "ca.pub")

	if err := cert.GenerateCA(caPrivPath, caPubPath); err != nil {
		t.Fatalf("Failed to generate CA: %v", err)
	}

	// Read and verify private key format
	privBytes, err := os.ReadFile(caPrivPath)
	if err != nil {
		t.Fatalf("Failed to read CA private key: %v", err)
	}

	if !strings.Contains(string(privBytes), "-----BEGIN OPENSSH PRIVATE KEY-----") {
		t.Error("CA private key should be in OpenSSH format")
	}

	// Read and verify public key format
	pubBytes, err := os.ReadFile(caPubPath)
	if err != nil {
		t.Fatalf("Failed to read CA public key: %v", err)
	}

	pubStr := string(pubBytes)
	if !strings.HasPrefix(pubStr, "ssh-ed25519") {
		t.Errorf("CA public key should start with ssh-ed25519, got: %s", pubStr[:20])
	}

	t.Log("CA keys are in correct format")
}

// TestKnownHostsFormatE2E tests known_hosts CA entry format
func TestKnownHostsFormatE2E(t *testing.T) {
	tmpDir := t.TempDir()

	caPrivPath := filepath.Join(tmpDir, "ca")
	caPubPath := filepath.Join(tmpDir, "ca.pub")

	if err := cert.GenerateCA(caPrivPath, caPubPath); err != nil {
		t.Fatalf("Failed to generate CA: %v", err)
	}

	caPubBytes, err := os.ReadFile(caPubPath)
	if err != nil {
		t.Fatalf("Failed to read CA public key: %v", err)
	}

	// Format for known_hosts
	knownHostsEntry := "@cert-authority * " + strings.TrimSpace(string(caPubBytes))

	// Verify format
	if !strings.HasPrefix(knownHostsEntry, "@cert-authority") {
		t.Error("Expected @cert-authority prefix")
	}

	if !strings.Contains(knownHostsEntry, "ssh-ed25519") {
		t.Error("Expected SSH key format")
	}

	// Write to known_hosts file
	knownHostsPath := filepath.Join(tmpDir, "known_hosts")
	if err := os.WriteFile(knownHostsPath, []byte(knownHostsEntry+"\n"), 0644); err != nil {
		t.Fatalf("Failed to write known_hosts: %v", err)
	}

	// Verify file content
	content, err := os.ReadFile(knownHostsPath)
	if err != nil {
		t.Fatalf("Failed to read known_hosts: %v", err)
	}

	if !strings.Contains(string(content), "@cert-authority") {
		t.Error("known_hosts should contain @cert-authority entry")
	}

	t.Logf("Known hosts entry format: %s", knownHostsEntry)
}
