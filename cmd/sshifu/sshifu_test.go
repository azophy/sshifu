package main

import (
	"os"
	"path/filepath"
	"testing"

	intssh "github.com/azophy/sshifu/internal/ssh"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantServer    string
		wantIdentity  string
		wantSSHArgs   []string
		expectIdentity bool // whether to check identity file
		wantErr       bool
	}{
		{
			name:        "minimal args",
			args:        []string{"auth.example.com"},
			wantServer:  "auth.example.com",
			wantSSHArgs: []string{"auth.example.com"},
			wantErr:     false,
		},
		{
			name:        "with target host",
			args:        []string{"auth.example.com", "target-server.com"},
			wantServer:  "auth.example.com",
			wantSSHArgs: []string{"target-server.com"},
			wantErr:     false,
		},
		{
			name:        "with identity file",
			args:        []string{"auth.example.com", "-i", "~/.ssh/my_key", "target-server.com"},
			wantServer:  "auth.example.com",
			wantSSHArgs: []string{"target-server.com"},
			expectIdentity: true,
			wantErr:     false,
		},
		{
			name:        "with multiple ssh args",
			args:        []string{"auth.example.com", "-v", "target-server.com", "ls", "-la"},
			wantServer:  "auth.example.com",
			wantSSHArgs: []string{"-v", "target-server.com", "ls", "-la"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parseArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseArgs() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}

			if cfg.ServerURL != tt.wantServer {
				t.Errorf("ServerURL = %v, want %v", cfg.ServerURL, tt.wantServer)
			}
			if tt.expectIdentity && cfg.IdentityFile == "" {
				t.Errorf("IdentityFile is empty, expected non-empty")
			}
			if len(cfg.SSHArgs) != len(tt.wantSSHArgs) {
				t.Errorf("SSHArgs length = %v, want %v", len(cfg.SSHArgs), len(tt.wantSSHArgs))
			}
		})
	}
}

func TestJoinURL(t *testing.T) {
	tests := []struct {
		name string
		base string
		path string
		want string
	}{
		{
			name: "hostname only",
			base: "auth.example.com",
			path: "/api/v1/login/start",
			want: "https://auth.example.com/api/v1/login/start",
		},
		{
			name: "with https scheme",
			base: "https://auth.example.com",
			path: "/api/v1/login/start",
			want: "https://auth.example.com/api/v1/login/start",
		},
		{
			name: "with port",
			base: "http://localhost:8080",
			path: "/api/v1/ca/pub",
			want: "http://localhost:8080/api/v1/ca/pub",
		},
		{
			name: "base with trailing slash",
			base: "https://auth.example.com/",
			path: "/api/v1/sign/user",
			want: "https://auth.example.com/api/v1/sign/user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinURL(tt.base, tt.path)
			if got != tt.want {
				t.Errorf("joinURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateTempKnownHosts(t *testing.T) {
	// Use a sample CA key for testing
	caKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl"

	tmpFile, err := createTempKnownHosts(caKey)
	if err != nil {
		t.Fatalf("createTempKnownHosts() error = %v", err)
	}
	defer os.Remove(tmpFile)

	// Verify file exists
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Errorf("Temp file was not created")
	}

	// Verify content
	content, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	expected := "@cert-authority * " + caKey + "\n"
	if string(content) != expected {
		t.Errorf("Temp file content = %v, want %v", string(content), expected)
	}
}

func TestSaveCertificate(t *testing.T) {
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, ".ssh", "id_ed25519-cert.pub")
	certificate := "ssh-ed25519-cert-v01@openssh.com AAAAaW52YWxpZA== test@example.com"

	t.Run("save new certificate", func(t *testing.T) {
		err := saveCertificate(certPath, certificate)
		if err != nil {
			t.Fatalf("saveCertificate() error = %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(certPath); os.IsNotExist(err) {
			t.Errorf("Certificate file was not created")
		}

		// Verify content
		content, err := os.ReadFile(certPath)
		if err != nil {
			t.Fatalf("Failed to read certificate: %v", err)
		}
		if string(content) != certificate {
			t.Errorf("Certificate content mismatch")
		}

		// Verify permissions
		info, err := os.Stat(certPath)
		if err != nil {
			t.Fatalf("Failed to stat file: %v", err)
		}
		if info.Mode().Perm()&0600 != 0600 {
			t.Errorf("Certificate permissions too open: %o", info.Mode().Perm())
		}
	})
}

func TestGetCertificatePath(t *testing.T) {
	tests := []struct {
		keyPath string
		want    string
	}{
		{"/home/user/.ssh/id_ed25519", "/home/user/.ssh/id_ed25519-cert.pub"},
		{"~/.ssh/id_rsa", "~/.ssh/id_rsa-cert.pub"},
		{"./test_key", "./test_key-cert.pub"},
	}

	for _, tt := range tests {
		t.Run(tt.keyPath, func(t *testing.T) {
			got := intssh.GetCertificatePath(tt.keyPath)
			if got != tt.want {
				t.Errorf("GetCertificatePath() = %v, want %v", got, tt.want)
			}
		})
	}
}
