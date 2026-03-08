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
