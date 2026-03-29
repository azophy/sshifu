package config

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestGenerateCAKeys(t *testing.T) {
	t.Run("generates valid keypair", func(t *testing.T) {
		tmpDir := t.TempDir()
		privateKeyPath := filepath.Join(tmpDir, "ca")
		publicKeyPath := filepath.Join(tmpDir, "ca.pub")

		if err := GenerateCAKeys(privateKeyPath, publicKeyPath); err != nil {
			t.Fatalf("GenerateCAKeys() error = %v", err)
		}

		// Verify private key file exists with correct permissions
		privInfo, err := os.Stat(privateKeyPath)
		if err != nil {
			t.Errorf("Private key file not created: %v", err)
		} else if privInfo.Mode().Perm()&0077 != 0 {
			t.Errorf("Private key permissions = %o, want restricted (no group/other access)", privInfo.Mode().Perm())
		}

		// Verify public key file exists
		if _, err := os.Stat(publicKeyPath); err != nil {
			t.Errorf("Public key file not created: %v", err)
		}

		// Verify private key can be loaded
		privData, err := os.ReadFile(privateKeyPath)
		if err != nil {
			t.Fatalf("Failed to read private key: %v", err)
		}

		_, err = ssh.ParsePrivateKey(privData)
		if err != nil {
			t.Errorf("Failed to parse private key: %v", err)
		}

		// Verify public key can be loaded
		pubData, err := os.ReadFile(publicKeyPath)
		if err != nil {
			t.Fatalf("Failed to read public key: %v", err)
		}

		_, _, _, _, err = ssh.ParseAuthorizedKey(pubData)
		if err != nil {
			t.Errorf("Failed to parse public key: %v", err)
		}
	})

	t.Run("fails for invalid path", func(t *testing.T) {
		err := GenerateCAKeys("/nonexistent/dir/ca", "/nonexistent/dir/ca.pub")
		if err == nil {
			t.Error("GenerateCAKeys() expected error for invalid path, got nil")
		}
	})
}

func TestFileExists(t *testing.T) {
	t.Run("returns true for existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")

		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		if !fileExists(testFile) {
			t.Error("fileExists() = false, want true for existing file")
		}
	})

	t.Run("returns false for non-existing file", func(t *testing.T) {
		if fileExists("/nonexistent/file.txt") {
			t.Error("fileExists() = true, want false for non-existing file")
		}
	})

	t.Run("returns false for directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		if fileExists(tmpDir) {
			t.Error("fileExists() = true, want false for directory")
		}
	})
}

func TestWizardResult(t *testing.T) {
	t.Run("WizardResult struct fields", func(t *testing.T) {
		result := &WizardResult{
			Config:       &Config{},
			GeneratedCA:  true,
			CAKeyPath:    "/path/to/ca",
			CAPublicPath: "/path/to/ca.pub",
		}

		if result.Config == nil {
			t.Error("WizardResult.Config is nil")
		}
		if !result.GeneratedCA {
			t.Error("WizardResult.GeneratedCA should be true")
		}
		if result.CAKeyPath != "/path/to/ca" {
			t.Errorf("WizardResult.CAKeyPath = %q, want %q", result.CAKeyPath, "/path/to/ca")
		}
		if result.CAPublicPath != "/path/to/ca.pub" {
			t.Errorf("WizardResult.CAPublicPath = %q, want %q", result.CAPublicPath, "/path/to/ca.pub")
		}
	})
}

func TestSaveWithCommentedAlternative(t *testing.T) {
	t.Run("saves config with commented OIDC alternative", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yml")

		cfg := &Config{}
		cfg.Server.Listen = ":8080"
		cfg.Server.PublicURL = "https://auth.example.com"
		cfg.CA.PrivateKey = "./ca"
		cfg.CA.PublicKey = "./ca.pub"
		cfg.Cert.TTL = "8h"
		cfg.Cert.Extensions = map[string]bool{
			"permit-pty":             true,
			"permit-port-forwarding": true,
		}
		cfg.Auth.Providers = []Provider{
			{
				Name:       "github",
				Type:       "github",
				ClientID:   "test-client-id",
				ClientSecret: "test-client-secret",
				AllowedOrg: "test-org",
			},
		}

		if err := SaveWithCommentedAlternative(cfg, configPath, "oidc"); err != nil {
			t.Fatalf("SaveWithCommentedAlternative() error = %v", err)
		}

		// Read and verify the config file
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("Failed to read config file: %v", err)
		}

		content := string(data)
		t.Logf("Generated config:\n%s", content)

		// Check for GitHub provider (active)
		if !contains(content, "- name: github") {
			t.Error("Config missing active GitHub provider")
		}
		if !contains(content, "client_id: test-client-id") {
			t.Error("Config missing GitHub client_id")
		}

		// Check for commented OIDC alternative (all lines commented)
		if !contains(content, "# Optional: OIDC provider") {
			t.Error("Config missing commented OIDC alternative header")
		}
		if !contains(content, "# - name: oidc") {
			t.Error("Config missing commented OIDC name line")
		}
		if !contains(content, "#   type: oidc") {
			t.Error("Config missing commented OIDC type line")
		}
	})

	t.Run("saves config with commented GitHub alternative", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yml")

		cfg := &Config{}
		cfg.Server.Listen = ":8080"
		cfg.Server.PublicURL = "https://auth.example.com"
		cfg.CA.PrivateKey = "./ca"
		cfg.CA.PublicKey = "./ca.pub"
		cfg.Cert.TTL = "8h"
		cfg.Cert.Extensions = map[string]bool{
			"permit-pty":             true,
			"permit-port-forwarding": true,
		}
		cfg.Auth.Providers = []Provider{
			{
				Name:        "oidc",
				Type:        "oidc",
				ClientID:    "test-oidc-client-id",
				ClientSecret: "test-oidc-client-secret",
				Issuer:      "https://accounts.google.com",
				PrincipalOAuthFieldName: "preferred_username",
			},
		}

		if err := SaveWithCommentedAlternative(cfg, configPath, "github"); err != nil {
			t.Fatalf("SaveWithCommentedAlternative() error = %v", err)
		}

		// Read and verify the config file
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("Failed to read config file: %v", err)
		}

		content := string(data)
		t.Logf("Generated config:\n%s", content)

		// Check for OIDC provider (active)
		if !contains(content, "- name: oidc") {
			t.Error("Config missing active OIDC provider")
		}
		if !contains(content, "client_id: test-oidc-client-id") {
			t.Error("Config missing OIDC client_id")
		}

		// Check for commented GitHub alternative (all lines commented)
		if !contains(content, "# Optional: GitHub provider") {
			t.Error("Config missing commented GitHub alternative header")
		}
		if !contains(content, "# - name: github") {
			t.Error("Config missing commented GitHub name line")
		}
		if !contains(content, "#   type: github") {
			t.Error("Config missing commented GitHub type line")
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
