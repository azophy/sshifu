package e2e

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
)

// TestServer holds the running test server state
type TestServer struct {
	BaseURL    string
	ServerURL  string
	Config     *TestConfig
	HTTPClient *http.Client
}

// TestConfig holds test configuration
type TestConfig struct {
	ListenAddr   string
	PublicURL    string
	CAPrivateKey string
	CAPublicKey  string
	GitHubOrg    string
}

// APIClient provides helper methods for API calls
type APIClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// StartTestServer starts a test instance of sshifu-server
func StartTestServer(cfg *TestConfig) (*TestServer, error) {
	// Generate CA keys if not provided
	if cfg.CAPrivateKey == "" || cfg.CAPublicKey == "" {
		if err := generateTestCAKeys(cfg); err != nil {
			return nil, fmt.Errorf("failed to generate CA keys: %w", err)
		}
	}

	// Create test config file
	configData := fmt.Sprintf(`
server:
  listen: %s
  public_url: %s

ca:
  private_key: %s
  public_key: %s

cert:
  ttl: 1h
  extensions:
    permit-pty: true
    permit-port-forwarding: true

auth:
  providers:
    - name: github
      type: github
      client_id: test_client_id
      client_secret: test_client_secret
      allowed_org: %s
`, cfg.ListenAddr, cfg.PublicURL, cfg.CAPrivateKey, cfg.CAPublicKey, cfg.GitHubOrg)

	configPath := filepath.Join(os.TempDir(), fmt.Sprintf("sshifu-test-config-%d.yml", time.Now().UnixNano()))
	if err := os.WriteFile(configPath, []byte(configData), 0600); err != nil {
		return nil, fmt.Errorf("failed to write config: %w", err)
	}

	// Note: In a real e2e test, we would start the actual server here
	// For now, we'll use mock testing approach
	// The full integration test would require:
	// 1. Building sshifu-server binary
	// 2. Starting it as a subprocess
	// 3. Waiting for it to be ready
	// 4. Cleaning up on test completion

	return &TestServer{
		BaseURL:   "http://" + cfg.ListenAddr,
		ServerURL: cfg.PublicURL,
		Config:    cfg,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// generateTestCAKeys generates ED25519 keys for testing
func generateTestCAKeys(cfg *TestConfig) error {
	// Generate ED25519 key pair
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}

	// Create temp files
	privPath := filepath.Join(os.TempDir(), fmt.Sprintf("test-ca-priv-%d", time.Now().UnixNano()))
	pubPath := privPath + ".pub"

	// Write private key (OpenSSH format)
	privBlock, err := ssh.MarshalPrivateKey(priv, "ca-test")
	if err != nil {
		return err
	}
	privPEM := pem.EncodeToMemory(privBlock)
	if err := os.WriteFile(privPath, privPEM, 0600); err != nil {
		return err
	}

	// Write public key
	pubKey, err := ssh.NewPublicKey(pub)
	if err != nil {
		return err
	}
	pubLine := ssh.MarshalAuthorizedKey(pubKey)
	if err := os.WriteFile(pubPath, append(pubLine, '\n'), 0644); err != nil {
		return err
	}

	cfg.CAPrivateKey = privPath
	cfg.CAPublicKey = pubPath

	return nil
}

// NewAPIClient creates a new API client
func NewAPIClient(baseURL string) *APIClient {
	return &APIClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// LoginStart calls the login start endpoint
func (c *APIClient) LoginStart() (sessionID, loginURL string, err error) {
	req, err := http.NewRequest("POST", c.BaseURL+"/api/v1/login/start", nil)
	if err != nil {
		return "", "", err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("login start failed: %s", resp.Status)
	}

	var result struct {
		SessionID string `json:"session_id"`
		LoginURL  string `json:"login_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}

	return result.SessionID, result.LoginURL, nil
}

// LoginStatus checks the login status
func (c *APIClient) LoginStatus(sessionID string) (status string, accessToken string, err error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/api/v1/login/status/" + sessionID)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", "", fmt.Errorf("session not found")
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("login status failed: %s", resp.Status)
	}

	var result struct {
		Status      string `json:"status"`
		AccessToken string `json:"access_token,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}

	return result.Status, result.AccessToken, nil
}

// GetCAPublicKey fetches the CA public key
func (c *APIClient) GetCAPublicKey() (string, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/api/v1/ca/pub")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get CA public key failed: %s", resp.Status)
	}

	var result struct {
		PublicKey string `json:"public_key"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.PublicKey, nil
}

// SignUserCertificate requests a user certificate
func (c *APIClient) SignUserCertificate(publicKey, accessToken string) (string, error) {
	reqBody := map[string]string{
		"public_key":   publicKey,
		"access_token": accessToken,
	}

	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/api/v1/sign/user", bytes.NewReader(reqBodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("sign user certificate failed: %s - %s", resp.Status, string(body))
	}

	var result struct {
		Certificate string `json:"certificate"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Certificate, nil
}

// SignHostCertificate requests a host certificate
func (c *APIClient) SignHostCertificate(publicKey string, principals []string) (string, error) {
	reqBody := map[string]interface{}{
		"public_key":  publicKey,
		"principals":  principals,
	}

	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", c.BaseURL+"/api/v1/sign/host", bytes.NewReader(reqBodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("sign host certificate failed: %s - %s", resp.Status, string(body))
	}

	var result struct {
		Certificate string `json:"certificate"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Certificate, nil
}

// GenerateTestKeyPair generates an SSH key pair for testing
func GenerateTestKeyPair() (privKey, pubKey string, err error) {
	// Generate RSA key for testing
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	// Create public key
	pubSSH, err := ssh.NewPublicKey(&priv.PublicKey)
	if err != nil {
		return "", "", err
	}

	pubLine := ssh.MarshalAuthorizedKey(pubSSH)

	// Create private key in OpenSSH format
	privBlock, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return "", "", err
	}

	var privBuf bytes.Buffer
	if err := pem.Encode(&privBuf, privBlock); err != nil {
		return "", "", err
	}

	return privBuf.String(), string(pubLine), nil
}

// PollLoginStatus polls the login status until approved or timeout
func (c *APIClient) PollLoginStatus(sessionID string, timeout time.Duration) (string, error) {
	start := time.Now()
	for time.Since(start) < timeout {
		status, token, err := c.LoginStatus(sessionID)
		if err != nil {
			return "", err
		}

		if status == "approved" {
			return token, nil
		}

		time.Sleep(1 * time.Second)
	}

	return "", fmt.Errorf("login polling timed out")
}
