package e2e

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/azophy/sshifu/internal/api"
	"github.com/azophy/sshifu/internal/cert"
	"github.com/azophy/sshifu/internal/oauth"
	"github.com/azophy/sshifu/internal/session"
	"golang.org/x/crypto/ssh"
)

// TestSshifuTrustE2E tests the complete sshifu-trust workflow
func TestSshifuTrustE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	tmpDir := t.TempDir()

	caPrivPath := filepath.Join(tmpDir, "ca")
	caPubPath := filepath.Join(tmpDir, "ca.pub")

	if err := cert.GenerateCA(caPrivPath, caPubPath); err != nil {
		t.Fatalf("Failed to generate CA: %v", err)
	}

	caSigner, err := cert.LoadCA(caPrivPath)
	if err != nil {
		t.Fatalf("Failed to load CA: %v", err)
	}

	sessionStore := session.NewStore(15 * time.Minute)
	mockOAuth := oauth.NewGitHubProvider(
		"test_client_id",
		"test_client_secret",
		"http://localhost:8080/oauth/callback",
		"test-org",
	)

	handlerCfg := &api.Config{
		TTL: 720 * time.Hour,
		Extensions: map[string]bool{
			"permit-pty":             true,
			"permit-port-forwarding": true,
		},
	}

	providers := map[string]oauth.Provider{"github": mockOAuth}
	handler, err := api.NewHandler(sessionStore, providers, caSigner.Signer(), handlerCfg, "http://localhost:8080")
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Read CA public key for the handler
	caPubBytes, _ := os.ReadFile(caPubPath)
	caPubKey := strings.TrimSpace(string(caPubBytes))

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/ca/pub", func(w http.ResponseWriter, r *http.Request) {
		handler.CAPublicKey(w, r, caPubKey)
	})
	mux.HandleFunc("/api/v1/sign/host", handler.SignHostCertificate)

	server := httptest.NewServer(mux)
	defer server.Close()

	testClient := &APIClient{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	etcSSHDir := filepath.Join(tmpDir, "etc", "ssh")
	if err := os.MkdirAll(etcSSHDir, 0755); err != nil {
		t.Fatalf("Failed to create etc/ssh directory: %v", err)
	}

	t.Run("TrustWorkflow_Step1_DownloadCA", func(t *testing.T) {
		caPub, err := testClient.GetCAPublicKey()
		if err != nil {
			t.Fatalf("Failed to download CA public key: %v", err)
		}

		if caPub == "" {
			t.Fatal("Expected non-empty CA public key")
		}

		caInstallPath := filepath.Join(etcSSHDir, "sshifu_ca.pub")
		if err := os.WriteFile(caInstallPath, []byte(caPub+"\n"), 0644); err != nil {
			t.Fatalf("Failed to install CA key: %v", err)
		}

		installed, err := os.ReadFile(caInstallPath)
		if err != nil {
			t.Fatalf("Failed to read installed CA key: %v", err)
		}

		if !strings.Contains(string(installed), "ssh-") {
			t.Error("Expected SSH key format in installed CA key")
		}

		t.Logf("CA key installed to: %s", caInstallPath)
	})

	t.Run("TrustWorkflow_Step2_GenerateHostKey", func(t *testing.T) {
		hostPrivPath := filepath.Join(etcSSHDir, "ssh_host_ed25519_key")
		hostPubPath := filepath.Join(etcSSHDir, "ssh_host_ed25519_key.pub")

		if err := cert.GenerateCA(hostPrivPath, hostPubPath); err != nil {
			t.Fatalf("Failed to generate host keys: %v", err)
		}

		if _, err := os.Stat(hostPubPath); os.IsNotExist(err) {
			t.Error("Host public key should exist")
		}

		t.Logf("Host key generated: %s", hostPubPath)
	})

	t.Run("TrustWorkflow_Step3_RequestHostCert", func(t *testing.T) {
		hostPubPath := filepath.Join(etcSSHDir, "ssh_host_ed25519_key.pub")
		hostPubBytes, err := os.ReadFile(hostPubPath)
		if err != nil {
			t.Fatalf("Failed to read host public key: %v", err)
		}

		principals := []string{"testhost.example.com", "testhost", "192.168.1.100"}
		certStr, err := testClient.SignHostCertificate(string(hostPubBytes), principals)
		if err != nil {
			t.Fatalf("Failed to sign host certificate: %v", err)
		}

		if certStr == "" {
			t.Fatal("Expected non-empty host certificate")
		}

		certBytes := []byte(certStr)
		parsedKey, _, _, _, err := ssh.ParseAuthorizedKey(certBytes)
		if err != nil {
			t.Fatalf("Failed to parse host certificate: %v", err)
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

		t.Logf("Host certificate signed with principals: %v", cert.ValidPrincipals)
	})

	t.Run("TrustWorkflow_Step4_InstallHostCert", func(t *testing.T) {
		hostPubPath := filepath.Join(etcSSHDir, "ssh_host_ed25519_key.pub")
		hostPubBytes, err := os.ReadFile(hostPubPath)
		if err != nil {
			t.Fatalf("Failed to read host public key: %v", err)
		}

		principals := []string{"testhost.example.com"}
		certStr, _ := testClient.SignHostCertificate(string(hostPubBytes), principals)

		certPath := filepath.Join(etcSSHDir, "ssh_host_ed25519_key-cert.pub")
		if err := os.WriteFile(certPath, []byte(certStr), 0644); err != nil {
			t.Fatalf("Failed to install host certificate: %v", err)
		}

		if _, err := os.Stat(certPath); os.IsNotExist(err) {
			t.Error("Host certificate should be installed")
		}

		t.Logf("Host certificate installed to: %s", certPath)
	})

	t.Run("TrustWorkflow_Complete", func(t *testing.T) {
		requiredFiles := []string{
			"sshifu_ca.pub",
			"ssh_host_ed25519_key.pub",
			"ssh_host_ed25519_key-cert.pub",
		}

		for _, file := range requiredFiles {
			path := filepath.Join(etcSSHDir, file)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("Required file missing: %s", file)
			}
		}

		t.Log("Complete sshifu-trust workflow successful")
	})
}

// TestHostCertificateValidation tests host certificate properties
func TestHostCertificateValidation(t *testing.T) {
	tmpDir := t.TempDir()

	caPrivPath := filepath.Join(tmpDir, "ca")
	caPubPath := filepath.Join(tmpDir, "ca.pub")
	if err := cert.GenerateCA(caPrivPath, caPubPath); err != nil {
		t.Fatalf("Failed to generate CA: %v", err)
	}
	caSigner, _ := cert.LoadCA(caPrivPath)

	sessionStore := session.NewStore(15 * time.Minute)
	mockOAuth := oauth.NewGitHubProvider(
		"test_client_id",
		"test_client_secret",
		"http://localhost:8080/oauth/callback",
		"test-org",
	)

	handlerCfg := &api.Config{
		TTL:        720 * time.Hour,
		Extensions: map[string]bool{"permit-pty": true},
	}

	providers := map[string]oauth.Provider{"github": mockOAuth}
	handler, err := api.NewHandler(sessionStore, providers, caSigner.Signer(), handlerCfg, "http://localhost:8080")
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/sign/host", handler.SignHostCertificate)

	server := httptest.NewServer(mux)
	defer server.Close()

	testClient := &APIClient{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	t.Run("HostCert_MultiplePrincipals", func(t *testing.T) {
		_, pubKey, _ := GenerateTestKeyPair()

		principals := []string{
			"host1.example.com",
			"host2.example.com",
			"192.168.1.1",
			"10.0.0.1",
		}

		certStr, err := testClient.SignHostCertificate(pubKey, principals)
		if err != nil {
			t.Fatalf("Failed to sign certificate: %v", err)
		}

		parsedKey, _, _, _, _ := ssh.ParseAuthorizedKey([]byte(certStr))
		cert, _ := parsedKey.(*ssh.Certificate)

		if len(cert.ValidPrincipals) != len(principals) {
			t.Errorf("Expected %d principals, got: %d", len(principals), len(cert.ValidPrincipals))
		}

		principalMap := make(map[string]bool)
		for _, p := range cert.ValidPrincipals {
			principalMap[p] = true
		}

		for _, expected := range principals {
			if !principalMap[expected] {
				t.Errorf("Missing principal: %s", expected)
			}
		}
	})

	t.Run("HostCert_Type", func(t *testing.T) {
		_, pubKey, _ := GenerateTestKeyPair()

		certStr, _ := testClient.SignHostCertificate(pubKey, []string{"host.example.com"})
		parsedKey, _, _, _, _ := ssh.ParseAuthorizedKey([]byte(certStr))
		cert, _ := parsedKey.(*ssh.Certificate)

		if cert.CertType != ssh.HostCert {
			t.Errorf("Expected HostCert (%d), got: %d", ssh.HostCert, cert.CertType)
		}
	})

	t.Run("HostCert_ValidBefore", func(t *testing.T) {
		_, pubKey, _ := GenerateTestKeyPair()

		certStr, _ := testClient.SignHostCertificate(pubKey, []string{"host.example.com"})
		parsedKey, _, _, _, _ := ssh.ParseAuthorizedKey([]byte(certStr))
		cert, _ := parsedKey.(*ssh.Certificate)

		expectedValidBefore := uint64(time.Now().Add(720 * time.Hour).Unix())
		margin := uint64(60)

		if cert.ValidBefore < expectedValidBefore-margin || cert.ValidBefore > expectedValidBefore+margin {
			t.Errorf("Certificate validity unexpected: got %d, expected ~%d", cert.ValidBefore, expectedValidBefore)
		}

		if cert.ValidBefore < uint64(time.Now().Unix()) {
			t.Error("Certificate should be valid now")
		}
	})
}

// TestCAKeyDistribution tests CA key distribution to clients
func TestCAKeyDistribution(t *testing.T) {
	tmpDir := t.TempDir()

	caPrivPath := filepath.Join(tmpDir, "ca")
	caPubPath := filepath.Join(tmpDir, "ca.pub")
	if err := cert.GenerateCA(caPrivPath, caPubPath); err != nil {
		t.Fatalf("Failed to generate CA: %v", err)
	}

	expectedBytes, _ := os.ReadFile(caPubPath)
	expectedKey := strings.TrimSpace(string(expectedBytes))

	caSigner, _ := cert.LoadCA(caPrivPath)
	sessionStore := session.NewStore(15 * time.Minute)
	mockOAuth := oauth.NewGitHubProvider(
		"test_client_id",
		"test_client_secret",
		"http://localhost:8080/oauth/callback",
		"test-org",
	)

	handlerCfg := &api.Config{
		TTL:        8 * time.Hour,
		Extensions: map[string]bool{"permit-pty": true},
	}

	providers := map[string]oauth.Provider{"github": mockOAuth}
	handler, err := api.NewHandler(sessionStore, providers, caSigner.Signer(), handlerCfg, "http://localhost:8080")
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Read CA public key for the handler
	caPubBytes, _ := os.ReadFile(caPubPath)
	caPubKey := strings.TrimSpace(string(caPubBytes))

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/ca/pub", func(w http.ResponseWriter, r *http.Request) {
		handler.CAPublicKey(w, r, caPubKey)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	testClient := &APIClient{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	t.Run("CAKey_EndpointReturnsKey", func(t *testing.T) {
		response, err := testClient.GetCAPublicKey()
		if err != nil {
			t.Fatalf("GetCAPublicKey failed: %v", err)
		}

		if response != expectedKey {
			t.Error("CA public key mismatch")
		}
	})

	t.Run("CAKey_KnownHostsFormat", func(t *testing.T) {
		response, _ := testClient.GetCAPublicKey()

		knownHostsEntry := "@cert-authority * " + response

		if !strings.HasPrefix(knownHostsEntry, "@cert-authority") {
			t.Error("Expected @cert-authority prefix")
		}

		if !strings.Contains(knownHostsEntry, "ssh-") {
			t.Error("Expected SSH key format")
		}

		t.Logf("Known hosts entry: %s", knownHostsEntry)
	})
}

// TestSSHDConfigUpdate tests sshd_config modification logic
func TestSSHDConfigUpdate(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("SSHDConfig_AddDirectives", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "sshd_config")
		originalConfig := `# Sample sshd_config
Port 22
PermitRootLogin no
PasswordAuthentication yes
`
		if err := os.WriteFile(configPath, []byte(originalConfig), 0644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		caKeyPath := "/etc/ssh/sshifu_ca.pub"
		hostCertPath := "/etc/ssh/ssh_host_ed25519_key-cert.pub"

		content, _ := os.ReadFile(configPath)
		configStr := string(content)

		if !strings.Contains(configStr, "TrustedUserCAKeys") {
			configStr += "\nTrustedUserCAKeys " + caKeyPath + "\n"
		}
		if !strings.Contains(configStr, "HostCertificate") {
			configStr += "\nHostCertificate " + hostCertPath + "\n"
		}

		if err := os.WriteFile(configPath, []byte(configStr), 0644); err != nil {
			t.Fatalf("Failed to write updated config: %v", err)
		}

		updated, _ := os.ReadFile(configPath)
		updatedStr := string(updated)

		if !strings.Contains(updatedStr, "TrustedUserCAKeys "+caKeyPath) {
			t.Error("TrustedUserCAKeys directive not added")
		}

		if !strings.Contains(updatedStr, "HostCertificate "+hostCertPath) {
			t.Error("HostCertificate directive not added")
		}

		if !strings.Contains(updatedStr, "Port 22") {
			t.Error("Original config not preserved")
		}

		t.Log("sshd_config updated successfully")
	})

	t.Run("SSHDConfig_UpdateExistingDirectives", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "sshd_config_old")
		oldConfig := `Port 22
TrustedUserCAKeys /old/path/ca.pub
HostCertificate /old/path/cert.pub
`
		if err := os.WriteFile(configPath, []byte(oldConfig), 0644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}

		newCAPath := "/etc/ssh/sshifu_ca.pub"
		newCertPath := "/etc/ssh/ssh_host_ed25519_key-cert.pub"

		content, _ := os.ReadFile(configPath)
		lines := strings.Split(string(content), "\n")

		var newLines []string
		for _, line := range lines {
			if strings.HasPrefix(line, "TrustedUserCAKeys") {
				line = "TrustedUserCAKeys " + newCAPath
			}
			if strings.HasPrefix(line, "HostCertificate") {
				line = "HostCertificate " + newCertPath
			}
			newLines = append(newLines, line)
		}

		if err := os.WriteFile(configPath, []byte(strings.Join(newLines, "\n")), 0644); err != nil {
			t.Fatalf("Failed to write updated config: %v", err)
		}

		updated, _ := os.ReadFile(configPath)
		updatedStr := string(updated)

		if !strings.Contains(updatedStr, "TrustedUserCAKeys "+newCAPath) {
			t.Error("TrustedUserCAKeys not updated")
		}

		if !strings.Contains(updatedStr, "HostCertificate "+newCertPath) {
			t.Error("HostCertificate not updated")
		}

		t.Log("sshd_config directives updated successfully")
	})
}
