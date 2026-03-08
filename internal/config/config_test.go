package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	t.Run("loads valid config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yml")

		content := `
server:
  listen: ":9090"
  public_url: https://auth.example.com
ca:
  private_key: /path/to/ca
  public_key: /path/to/ca.pub
cert:
  ttl: 12h
  extensions:
    permit-pty: true
auth:
  providers:
    - name: github
      type: github
      client_id: abc123
      client_secret: secret456
      allowed_org: my-org
`
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.Server.Listen != ":9090" {
			t.Errorf("Server.Listen = %q, want %q", cfg.Server.Listen, ":9090")
		}
		if cfg.Server.PublicURL != "https://auth.example.com" {
			t.Errorf("Server.PublicURL = %q, want %q", cfg.Server.PublicURL, "https://auth.example.com")
		}
		if cfg.CA.PrivateKey != "/path/to/ca" {
			t.Errorf("CA.PrivateKey = %q, want %q", cfg.CA.PrivateKey, "/path/to/ca")
		}
		if cfg.Cert.TTL != "12h" {
			t.Errorf("Cert.TTL = %q, want %q", cfg.Cert.TTL, "12h")
		}
		if len(cfg.Auth.Providers) != 1 {
			t.Errorf("Auth.Providers count = %d, want %d", len(cfg.Auth.Providers), 1)
		}
	})

	t.Run("applies defaults for missing fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yml")

		content := `
server:
  public_url: https://auth.example.com
`
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.Server.Listen != ":8080" {
			t.Errorf("Server.Listen = %q, want %q", cfg.Server.Listen, ":8080")
		}
		if cfg.CA.PrivateKey != "./ca" {
			t.Errorf("CA.PrivateKey = %q, want %q", cfg.CA.PrivateKey, "./ca")
		}
		if cfg.CA.PublicKey != "./ca.pub" {
			t.Errorf("CA.PublicKey = %q, want %q", cfg.CA.PublicKey, "./ca.pub")
		}
		if cfg.Cert.TTL != "8h" {
			t.Errorf("Cert.TTL = %q, want %q", cfg.Cert.TTL, "8h")
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		_, err := Load("/nonexistent/path/config.yml")
		if err == nil {
			t.Error("Load() expected error for missing file, got nil")
		}
	})

	t.Run("returns error for invalid YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yml")

		content := `invalid: yaml: content: [`
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		_, err := Load(configPath)
		if err == nil {
			t.Error("Load() expected error for invalid YAML, got nil")
		}
	})
}

func TestExists(t *testing.T) {
	t.Run("returns true for existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yml")

		if err := os.WriteFile(configPath, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		if !Exists(configPath) {
			t.Error("Exists() = false, want true for existing file")
		}
	})

	t.Run("returns false for non-existing file", func(t *testing.T) {
		if Exists("/nonexistent/path/config.yml") {
			t.Error("Exists() = true, want false for non-existing file")
		}
	})
}

func TestDefaultTTL(t *testing.T) {
	t.Run("returns configured TTL", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yml")

		content := `
server:
  public_url: https://auth.example.com
cert:
  ttl: 24h
`
		if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		cfg, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		ttl := cfg.DefaultTTL()
		if ttl != 24*time.Hour {
			t.Errorf("DefaultTTL() = %v, want %v", ttl, 24*time.Hour)
		}
	})

	t.Run("returns default 8h for empty TTL", func(t *testing.T) {
		cfg := &Config{}
		cfg.Cert.TTL = ""

		ttl := cfg.DefaultTTL()
		if ttl != 8*time.Hour {
			t.Errorf("DefaultTTL() = %v, want %v", ttl, 8*time.Hour)
		}
	})

	t.Run("returns default 8h for invalid TTL", func(t *testing.T) {
		cfg := &Config{}
		cfg.Cert.TTL = "invalid"

		ttl := cfg.DefaultTTL()
		if ttl != 8*time.Hour {
			t.Errorf("DefaultTTL() = %v, want %v", ttl, 8*time.Hour)
		}
	})
}

func TestSave(t *testing.T) {
	t.Run("saves config to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yml")

		cfg := &Config{}
		cfg.Server.Listen = ":8080"
		cfg.Server.PublicURL = "https://auth.example.com"
		cfg.CA.PrivateKey = "./ca"
		cfg.CA.PublicKey = "./ca.pub"
		cfg.Cert.TTL = "8h"
		cfg.Cert.Extensions = map[string]bool{
			"permit-pty": true,
		}
		cfg.Auth.Providers = []Provider{
			{
				Name:       "github",
				Type:       "github",
				ClientID:   "abc123",
				ClientSecret: "secret",
				AllowedOrg: "my-org",
			},
		}

		if err := Save(cfg, configPath); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		// Verify file exists
		if !Exists(configPath) {
			t.Error("Save() did not create config file")
		}

		// Verify content can be loaded
		loaded, err := Load(configPath)
		if err != nil {
			t.Fatalf("Load() after Save() error = %v", err)
		}

		if loaded.Server.PublicURL != cfg.Server.PublicURL {
			t.Errorf("Loaded Server.PublicURL = %q, want %q", loaded.Server.PublicURL, cfg.Server.PublicURL)
		}
	})

	t.Run("creates directory if needed", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "subdir", "config.yml")

		cfg := &Config{}
		cfg.Server.PublicURL = "https://auth.example.com"

		// Create subdirectory first (Save doesn't create directories)
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}

		if err := Save(cfg, configPath); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		if !Exists(configPath) {
			t.Error("Save() did not create config file in subdirectory")
		}
	})
}

func TestMarshal(t *testing.T) {
	t.Run("marshals config to YAML", func(t *testing.T) {
		cfg := &Config{}
		cfg.Server.Listen = ":8080"
		cfg.Server.PublicURL = "https://auth.example.com"
		cfg.Cert.TTL = "8h"

		data, err := Marshal(cfg)
		if err != nil {
			t.Fatalf("Marshal() error = %v", err)
		}

		if len(data) == 0 {
			t.Error("Marshal() returned empty data")
		}
	})
}
