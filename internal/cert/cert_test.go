package cert

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// TestGenerateCA tests CA keypair generation
func TestGenerateCA(t *testing.T) {
	tmpDir := t.TempDir()
	privateKeyPath := filepath.Join(tmpDir, "ca")
	publicKeyPath := filepath.Join(tmpDir, "ca.pub")

	err := GenerateCA(privateKeyPath, publicKeyPath)
	if err != nil {
		t.Fatalf("GenerateCA failed: %v", err)
	}

	// Verify private key file exists and has correct permissions
	privInfo, err := os.Stat(privateKeyPath)
	if err != nil {
		t.Fatalf("Private key file not found: %v", err)
	}
	if privInfo.Mode().Perm()&0077 != 0 {
		t.Errorf("Private key file has incorrect permissions: %o", privInfo.Mode().Perm())
	}

	// Verify public key file exists
	pubInfo, err := os.Stat(publicKeyPath)
	if err != nil {
		t.Fatalf("Public key file not found: %v", err)
	}
	if pubInfo.Size() == 0 {
		t.Error("Public key file is empty")
	}

	// Verify we can load the CA
	ca, err := LoadCA(privateKeyPath)
	if err != nil {
		t.Fatalf("LoadCA failed: %v", err)
	}

	if ca == nil {
		t.Fatal("LoadCA returned nil")
	}

	if ca.PublicKey() == nil {
		t.Error("CA public key is nil")
	}
}

// TestLoadCAInvalidKey tests loading an invalid CA key
func TestLoadCAInvalidKey(t *testing.T) {
	tmpDir := t.TempDir()
	invalidKeyPath := filepath.Join(tmpDir, "invalid")

	// Write invalid key data
	err := os.WriteFile(invalidKeyPath, []byte("not a valid key"), 0600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err = LoadCA(invalidKeyPath)
	if err == nil {
		t.Error("Expected error when loading invalid key, got nil")
	}
}

// TestLoadCANonexistentKey tests loading a nonexistent CA key
func TestLoadCANonexistentKey(t *testing.T) {
	_, err := LoadCA("/nonexistent/path/ca")
	if err == nil {
		t.Error("Expected error when loading nonexistent key, got nil")
	}
}

// TestSignUserKey tests user certificate signing
func TestSignUserKey(t *testing.T) {
	// Generate CA keypair
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate CA key: %v", err)
	}

	caSigner, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("Failed to create CA signer: %v", err)
	}

	// Generate user key
	_, userPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate user key: %v", err)
	}

	userPubKey, err := ssh.NewPublicKey(userPriv.Public())
	if err != nil {
		t.Fatalf("Failed to create user public key: %v", err)
	}

	// Sign user certificate
	ttl := 8 * time.Hour
	extensions := map[string]bool{
		"permit-pty":             true,
		"permit-port-forwarding": true,
	}

	certBytes, err := SignUserKey(caSigner, userPubKey, "testuser", ttl, extensions)
	if err != nil {
		t.Fatalf("SignUserKey failed: %v", err)
	}

	if len(certBytes) == 0 {
		t.Fatal("SignUserKey returned empty certificate")
	}

	// Parse and verify the certificate
	cert, _, _, _, err := ssh.ParseAuthorizedKey(certBytes)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}

	sshCert, ok := cert.(*ssh.Certificate)
	if !ok {
		t.Fatal("Parsed key is not a certificate")
	}

	// Verify certificate properties
	if sshCert.CertType != ssh.UserCert {
		t.Errorf("Expected UserCert, got %d", sshCert.CertType)
	}

	if len(sshCert.ValidPrincipals) != 1 || sshCert.ValidPrincipals[0] != "testuser" {
		t.Errorf("Expected principal 'testuser', got %v", sshCert.ValidPrincipals)
	}

	// Verify extensions
	if _, exists := sshCert.Permissions.Extensions["permit-pty"]; !exists {
		t.Error("Missing permit-pty extension")
	}
	if _, exists := sshCert.Permissions.Extensions["permit-port-forwarding"]; !exists {
		t.Error("Missing permit-port-forwarding extension")
	}
}

// TestSignUserKeyDefaultExtensions tests that default extensions are applied
func TestSignUserKeyDefaultExtensions(t *testing.T) {
	// Generate CA keypair
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate CA key: %v", err)
	}

	caSigner, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("Failed to create CA signer: %v", err)
	}

	// Generate user key
	_, userPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate user key: %v", err)
	}

	userPubKey, err := ssh.NewPublicKey(userPriv.Public())
	if err != nil {
		t.Fatalf("Failed to create user public key: %v", err)
	}

	// Sign with empty extensions (should use defaults)
	certBytes, err := SignUserKey(caSigner, userPubKey, "testuser", 8*time.Hour, nil)
	if err != nil {
		t.Fatalf("SignUserKey failed: %v", err)
	}

	cert, _, _, _, err := ssh.ParseAuthorizedKey(certBytes)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}

	sshCert, ok := cert.(*ssh.Certificate)
	if !ok {
		t.Fatal("Parsed key is not a certificate")
	}

	// Verify default extensions
	expectedExtensions := []string{
		"permit-pty",
		"permit-port-forwarding",
		"permit-agent-forwarding",
		"permit-x11-forwarding",
	}

	for _, ext := range expectedExtensions {
		if _, exists := sshCert.Permissions.Extensions[ext]; !exists {
			t.Errorf("Missing default extension: %s", ext)
		}
	}
}

// TestSignHostKey tests host certificate signing
func TestSignHostKey(t *testing.T) {
	// Generate CA keypair
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate CA key: %v", err)
	}

	caSigner, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("Failed to create CA signer: %v", err)
	}

	// Generate host key
	_, hostPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate host key: %v", err)
	}

	hostPubKey, err := ssh.NewPublicKey(hostPriv.Public())
	if err != nil {
		t.Fatalf("Failed to create host public key: %v", err)
	}

	// Sign host certificate
	ttl := 30 * 24 * time.Hour
	principals := []string{"server.example.com", "192.168.1.100"}

	certBytes, err := SignHostKey(caSigner, hostPubKey, principals, ttl)
	if err != nil {
		t.Fatalf("SignHostKey failed: %v", err)
	}

	if len(certBytes) == 0 {
		t.Fatal("SignHostKey returned empty certificate")
	}

	// Parse and verify the certificate
	cert, _, _, _, err := ssh.ParseAuthorizedKey(certBytes)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}

	sshCert, ok := cert.(*ssh.Certificate)
	if !ok {
		t.Fatal("Parsed key is not a certificate")
	}

	// Verify certificate properties
	if sshCert.CertType != ssh.HostCert {
		t.Errorf("Expected HostCert, got %d", sshCert.CertType)
	}

	if len(sshCert.ValidPrincipals) != 2 {
		t.Errorf("Expected 2 principals, got %d", len(sshCert.ValidPrincipals))
	}

	for _, p := range principals {
		found := false
		for _, vp := range sshCert.ValidPrincipals {
			if vp == p {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected principal %q not found in certificate", p)
		}
	}
}

// TestSignHostKeyEmptyPrincipals tests host certificate with empty principals
func TestSignHostKeyEmptyPrincipals(t *testing.T) {
	// Generate CA keypair
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate CA key: %v", err)
	}

	caSigner, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("Failed to create CA signer: %v", err)
	}

	// Generate host key
	_, hostPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate host key: %v", err)
	}

	hostPubKey, err := ssh.NewPublicKey(hostPriv.Public())
	if err != nil {
		t.Fatalf("Failed to create host public key: %v", err)
	}

	// Sign with empty principals
	certBytes, err := SignHostKey(caSigner, hostPubKey, []string{}, 8*time.Hour)
	if err != nil {
		t.Fatalf("SignHostKey failed: %v", err)
	}

	cert, _, _, _, err := ssh.ParseAuthorizedKey(certBytes)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}

	sshCert, ok := cert.(*ssh.Certificate)
	if !ok {
		t.Fatal("Parsed key is not a certificate")
	}

	if sshCert.CertType != ssh.HostCert {
		t.Errorf("Expected HostCert, got %d", sshCert.CertType)
	}
}

// TestCA_Methods tests the CA struct methods
func TestCA_Methods(t *testing.T) {
	tmpDir := t.TempDir()
	privateKeyPath := filepath.Join(tmpDir, "ca")
	publicKeyPath := filepath.Join(tmpDir, "ca.pub")

	// Generate CA
	err := GenerateCA(privateKeyPath, publicKeyPath)
	if err != nil {
		t.Fatalf("GenerateCA failed: %v", err)
	}

	// Load CA
	ca, err := LoadCA(privateKeyPath)
	if err != nil {
		t.Fatalf("LoadCA failed: %v", err)
	}

	// Test PublicKey method
	pubKey := ca.PublicKey()
	if pubKey == nil {
		t.Error("PublicKey returned nil")
	}

	// Generate a test key and sign it
	_, userPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate user key: %v", err)
	}

	userPubKey, err := ssh.NewPublicKey(userPriv.Public())
	if err != nil {
		t.Fatalf("Failed to create user public key: %v", err)
	}

	certBytes, err := ca.SignUserKey(userPubKey, "testuser", 8*time.Hour, nil)
	if err != nil {
		t.Fatalf("CA.SignUserKey failed: %v", err)
	}

	if len(certBytes) == 0 {
		t.Error("CA.SignUserKey returned empty certificate")
	}

	// Test SignHostKey via CA
	_, hostPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate host key: %v", err)
	}

	hostPubKey, err := ssh.NewPublicKey(hostPriv.Public())
	if err != nil {
		t.Fatalf("Failed to create host public key: %v", err)
	}

	certBytes, err = ca.SignHostKey(hostPubKey, []string{"host.example.com"}, 24*time.Hour)
	if err != nil {
		t.Fatalf("CA.SignHostKey failed: %v", err)
	}

	if len(certBytes) == 0 {
		t.Error("CA.SignHostKey returned empty certificate")
	}
}
