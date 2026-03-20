package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/azophy/sshifu/internal/cert"
	"github.com/azophy/sshifu/internal/oauth"
	"github.com/azophy/sshifu/internal/session"
	"golang.org/x/crypto/ssh"
	"golang.org/x/oauth2"
)

// mockOAuthProvider is a mock OAuth provider for testing
type mockOAuthProvider struct {
	name           string
	authURL        string
	exchangeError  bool
	username       string
	usernameError  bool
	membershipError bool
}

func (m *mockOAuthProvider) Name() string                           { return m.name }
func (m *mockOAuthProvider) AuthURL(state string) string            { return m.authURL }
func (m *mockOAuthProvider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	if m.exchangeError {
		return nil, &url.Error{Op: "exchange", Err: &url.Error{}}
	}
	return &oauth2.Token{AccessToken: "mock-token"}, nil
}
func (m *mockOAuthProvider) GetUsername(ctx context.Context, token *oauth2.Token) (string, error) {
	if m.usernameError {
		return "", &url.Error{Op: "username", Err: &url.Error{}}
	}
	return m.username, nil
}
func (m *mockOAuthProvider) VerifyMembership(ctx context.Context, token *oauth2.Token, username string) error {
	if m.membershipError {
		return &url.Error{Op: "membership", Err: &url.Error{}}
	}
	return nil
}

// getTestCA returns a test CA signer for testing
func getTestCA() (ssh.Signer, error) {
	ca, err := cert.LoadCA("../../test_ca")
	if err != nil {
		if err := cert.GenerateCA("../../test_ca", "../../test_ca.pub"); err != nil {
			return nil, err
		}
		ca, err = cert.LoadCA("../../test_ca")
		if err != nil {
			return nil, err
		}
	}
	return ca.Signer(), nil
}

func TestLoginStart(t *testing.T) {
	store := session.NewStore(15 * time.Minute)
	provider := &mockOAuthProvider{name: "github", authURL: "https://github.com/oauth"}

	caSigner, err := getTestCA()
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	handlerCfg := &Config{
		TTL:        8 * time.Hour,
		Extensions: map[string]bool{},
	}

	providers := map[string]oauth.Provider{"github": provider}
	handler, err := NewHandler(store, providers, caSigner, handlerCfg, "http://localhost:8080")
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/login/start", nil)
	w := httptest.NewRecorder()

	handler.LoginStart(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp LoginStartResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.SessionID == "" {
		t.Error("SessionID is empty")
	}
	if resp.LoginURL == "" {
		t.Error("LoginURL is empty")
	}
	if !strings.HasPrefix(resp.LoginURL, "http://localhost:8080/login/") {
		t.Errorf("unexpected LoginURL: %s", resp.LoginURL)
	}
}

func TestLoginStartWrongMethod(t *testing.T) {
	store := session.NewStore(15 * time.Minute)
	provider := &mockOAuthProvider{name: "github", authURL: "https://github.com/oauth"}
	
	caSigner, err := getTestCA()
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	
	handlerCfg := &Config{
		TTL:        8 * time.Hour,
		Extensions: map[string]bool{},
	}
	
	handler, err := NewHandler(store, map[string]oauth.Provider{"github": provider}, caSigner, handlerCfg, "http://localhost:8080")
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/login/start", nil)
	w := httptest.NewRecorder()

	handler.LoginStart(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestLoginStatus(t *testing.T) {
	store := session.NewStore(15 * time.Minute)
	provider := &mockOAuthProvider{name: "github", authURL: "https://github.com/oauth"}
	
	caSigner, err := getTestCA()
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	
	handlerCfg := &Config{
		TTL:        8 * time.Hour,
		Extensions: map[string]bool{},
	}
	
	handler, err := NewHandler(store, map[string]oauth.Provider{"github": provider}, caSigner, handlerCfg, "http://localhost:8080")
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	// Create a session
	store.Create("test-session-123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/login/status/test-session-123", nil)
	w := httptest.NewRecorder()

	handler.LoginStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp LoginStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Status != "pending" {
		t.Errorf("expected status pending, got %s", resp.Status)
	}
}

func TestLoginStatusApproved(t *testing.T) {
	store := session.NewStore(15 * time.Minute)
	provider := &mockOAuthProvider{name: "github", authURL: "https://github.com/oauth"}
	
	caSigner, err := getTestCA()
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	
	handlerCfg := &Config{
		TTL:        8 * time.Hour,
		Extensions: map[string]bool{},
	}
	
	handler, err := NewHandler(store, map[string]oauth.Provider{"github": provider}, caSigner, handlerCfg, "http://localhost:8080")
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	// Create and approve a session
	store.Create("test-session-456")
	store.Approve("test-session-456", "testuser", "access-token-123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/login/status/test-session-456", nil)
	w := httptest.NewRecorder()

	handler.LoginStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp LoginStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Status != "approved" {
		t.Errorf("expected status approved, got %s", resp.Status)
	}
	if resp.AccessToken != "access-token-123" {
		t.Errorf("expected access_token access-token-123, got %s", resp.AccessToken)
	}
}

func TestLoginStatusNotFound(t *testing.T) {
	store := session.NewStore(15 * time.Minute)
	provider := &mockOAuthProvider{name: "github", authURL: "https://github.com/oauth"}
	
	caSigner, err := getTestCA()
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	
	handlerCfg := &Config{
		TTL:        8 * time.Hour,
		Extensions: map[string]bool{},
	}
	
	handler, err := NewHandler(store, map[string]oauth.Provider{"github": provider}, caSigner, handlerCfg, "http://localhost:8080")
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/login/status/non-existent", nil)
	w := httptest.NewRecorder()

	handler.LoginStatus(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestCAPublicKey(t *testing.T) {
	store := session.NewStore(15 * time.Minute)
	provider := &mockOAuthProvider{name: "github", authURL: "https://github.com/oauth"}
	
	caSigner, err := getTestCA()
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	
	handlerCfg := &Config{
		TTL:        8 * time.Hour,
		Extensions: map[string]bool{},
	}
	
	handler, err := NewHandler(store, map[string]oauth.Provider{"github": provider}, caSigner, handlerCfg, "http://localhost:8080")
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ca/pub", nil)
	w := httptest.NewRecorder()

	handler.CAPublicKey(w, req, "ssh-ed25519 AAAAC3NzaC1lZG1OdAAA")

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp CAPublicKeyResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.PublicKey != "ssh-ed25519 AAAAC3NzaC1lZG1OdAAA" {
		t.Errorf("expected specific public key, got %s", resp.PublicKey)
	}
}

func TestLoginTemplateExists(t *testing.T) {
	// Check if login.html exists (try multiple paths)
	templateExists := false
	if _, err := os.Stat("web/login.html"); err == nil {
		templateExists = true
	} else if _, err := os.Stat("../web/login.html"); err == nil {
		templateExists = true
	} else if _, err := os.Stat("../../web/login.html"); err == nil {
		templateExists = true
	}

	if !templateExists {
		t.Skip("web/login.html not found, skipping template test")
	}

	store := session.NewStore(15 * time.Minute)
	provider := &mockOAuthProvider{name: "github", authURL: "https://github.com/oauth"}
	
	caSigner, err := getTestCA()
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}
	
	handlerCfg := &Config{
		TTL:        8 * time.Hour,
		Extensions: map[string]bool{},
	}
	
	handler, err := NewHandler(store, map[string]oauth.Provider{"github": provider}, caSigner, handlerCfg, "http://localhost:8080")
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	store.Create("template-test-session")

	req := httptest.NewRequest(http.MethodGet, "/login/template-test-session", nil)
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	if !strings.Contains(w.Body.String(), "Sshifu Authentication") {
		t.Error("response does not contain expected content")
	}
}

// TestSignUserCertificate tests the user certificate signing endpoint
func TestSignUserCertificate(t *testing.T) {
	store := session.NewStore(15 * time.Minute)
	provider := &mockOAuthProvider{name: "github", authURL: "https://github.com/oauth"}
	
	caSigner, err := getTestCA()
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	handlerCfg := &Config{
		TTL: 8 * time.Hour,
		Extensions: map[string]bool{
			"permit-pty": true,
		},
	}
	
	handler, err := NewHandler(store, map[string]oauth.Provider{"github": provider}, caSigner, handlerCfg, "http://localhost:8080")
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	// Create and approve a session
	store.Create("test-session-sign")
	store.Approve("test-session-sign", "testuser", "access-token-sign")

	// Generate a test key pair for the user
	userKey, err := generateTestKey()
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	// Create request body
	reqBody := SignRequest{
		PublicKey:   userKey,
		AccessToken: "access-token-sign",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sign/user", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	handler.SignUserCertificate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp SignResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Certificate == "" {
		t.Error("Certificate is empty")
	}
	if !strings.HasPrefix(resp.Certificate, "ssh-ed25519-cert-v01@openssh.com") {
		t.Errorf("expected certificate format, got %s", resp.Certificate)
	}
}

// TestSignUserCertificateInvalidToken tests with an invalid access token
func TestSignUserCertificateInvalidToken(t *testing.T) {
	store := session.NewStore(15 * time.Minute)
	provider := &mockOAuthProvider{name: "github", authURL: "https://github.com/oauth"}
	
	caSigner, err := getTestCA()
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	handlerCfg := &Config{
		TTL:        8 * time.Hour,
		Extensions: map[string]bool{},
	}
	
	handler, err := NewHandler(store, map[string]oauth.Provider{"github": provider}, caSigner, handlerCfg, "http://localhost:8080")
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	// Generate a test key pair
	userKey, err := generateTestKey()
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	// Create request body with invalid token
	reqBody := SignRequest{
		PublicKey:   userKey,
		AccessToken: "invalid-token",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sign/user", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	handler.SignUserCertificate(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d: %s", w.Code, w.Body.String())
	}
}

// TestSignUserCertificateMissingFields tests with missing required fields
func TestSignUserCertificateMissingFields(t *testing.T) {
	store := session.NewStore(15 * time.Minute)
	provider := &mockOAuthProvider{name: "github", authURL: "https://github.com/oauth"}
	
	caSigner, err := getTestCA()
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	handlerCfg := &Config{
		TTL:        8 * time.Hour,
		Extensions: map[string]bool{},
	}
	
	handler, err := NewHandler(store, map[string]oauth.Provider{"github": provider}, caSigner, handlerCfg, "http://localhost:8080")
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	// Create request body with missing public key
	reqBody := SignRequest{
		AccessToken: "some-token",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sign/user", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	handler.SignUserCertificate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestSignHostCertificate tests the host certificate signing endpoint
func TestSignHostCertificate(t *testing.T) {
	store := session.NewStore(15 * time.Minute)
	provider := &mockOAuthProvider{name: "github", authURL: "https://github.com/oauth"}
	
	caSigner, err := getTestCA()
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	handlerCfg := &Config{
		TTL:        8 * time.Hour,
		Extensions: map[string]bool{},
	}
	
	handler, err := NewHandler(store, map[string]oauth.Provider{"github": provider}, caSigner, handlerCfg, "http://localhost:8080")
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	// Generate a test key pair for the host
	hostKey, err := generateTestKey()
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	// Create request body
	reqBody := HostSignRequest{
		PublicKey:  hostKey,
		Principals: []string{"server1.example.com", "192.168.1.100"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sign/host", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	handler.SignHostCertificate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp SignResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Certificate == "" {
		t.Error("Certificate is empty")
	}
	if !strings.HasPrefix(resp.Certificate, "ssh-ed25519-cert-v01@openssh.com") {
		t.Errorf("expected certificate format, got %s", resp.Certificate)
	}
}

// TestSignHostCertificateMissingFields tests with missing required fields
func TestSignHostCertificateMissingFields(t *testing.T) {
	store := session.NewStore(15 * time.Minute)
	provider := &mockOAuthProvider{name: "github", authURL: "https://github.com/oauth"}
	
	caSigner, err := getTestCA()
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	handlerCfg := &Config{
		TTL:        8 * time.Hour,
		Extensions: map[string]bool{},
	}
	
	handler, err := NewHandler(store, map[string]oauth.Provider{"github": provider}, caSigner, handlerCfg, "http://localhost:8080")
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	// Create request body with missing principals
	reqBody := HostSignRequest{
		PublicKey: "ssh-ed25519 AAAAC3NzaC1lZG1OdAAA",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/sign/host", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	handler.SignHostCertificate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

// TestSignHostCertificateWrongMethod tests with wrong HTTP method
func TestSignHostCertificateWrongMethod(t *testing.T) {
	store := session.NewStore(15 * time.Minute)
	provider := &mockOAuthProvider{name: "github", authURL: "https://github.com/oauth"}
	
	caSigner, err := getTestCA()
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	handlerCfg := &Config{
		TTL:        8 * time.Hour,
		Extensions: map[string]bool{},
	}
	
	handler, err := NewHandler(store, map[string]oauth.Provider{"github": provider}, caSigner, handlerCfg, "http://localhost:8080")
	if err != nil {
		t.Skipf("Skipping test: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sign/host", nil)
	w := httptest.NewRecorder()

	handler.SignHostCertificate(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

// generateTestKey generates a test SSH public key
func generateTestKey() (string, error) {
	// For testing, we'll use a pre-generated valid-looking key
	// In real scenarios, this would be generated by the client
	return "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl test@example.com", nil
}
