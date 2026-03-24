package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/azophy/sshifu/internal/cert"
	"github.com/azophy/sshifu/internal/oauth"
	"github.com/azophy/sshifu/internal/session"
	"github.com/azophy/sshifu/embed"
	"golang.org/x/crypto/ssh"
)

// Handler manages HTTP request handling
type Handler struct {
	sessionStore  *session.Store
	oauthProviders map[string]oauth.Provider
	caSigner      ssh.Signer
	config        *Config
	publicURL     string
	loginTemplate *template.Template
}

// Config holds configuration for certificate signing
type Config struct {
	TTL        time.Duration
	Extensions map[string]bool
}

// NewHandler creates a new API handler
func NewHandler(store *session.Store, providers map[string]oauth.Provider, caSigner ssh.Signer, cfg *Config, publicURL string) (*Handler, error) {
	// Load template from embedded FS
	tmpl, err := embed.LoadLoginTemplate()
	if err != nil {
		return nil, err
	}

	return &Handler{
		sessionStore:   store,
		oauthProviders: providers,
		caSigner:       caSigner,
		config:         cfg,
		publicURL:      publicURL,
		loginTemplate:  tmpl,
	}, nil
}

// generateSessionID generates a random session ID
func generateSessionID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// LoginStart handles POST /api/v1/login/start
func (h *Handler) LoginStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sessionID, err := generateSessionID()
	if err != nil {
		log.Printf("Failed to generate session ID: %v", err)
		WriteError(w, http.StatusInternalServerError, "failed to generate session ID")
		return
	}

	h.sessionStore.Create(sessionID)

	loginURL := h.publicURL + "/login/" + sessionID

	WriteJSON(w, http.StatusOK, LoginStartResponse{
		SessionID: sessionID,
		LoginURL:  loginURL,
	})
}

// LoginStatus handles GET /api/v1/login/status/{session_id}
func (h *Handler) LoginStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract session ID from URL path
	sessionID := strings.TrimPrefix(r.URL.Path, "/api/v1/login/status/")
	if sessionID == "" {
		WriteError(w, http.StatusBadRequest, "session ID required")
		return
	}

	sess, exists := h.sessionStore.Get(sessionID)
	if !exists {
		WriteError(w, http.StatusNotFound, "session not found or expired")
		return
	}

	response := LoginStatusResponse{
		Status: string(sess.Status),
	}

	if sess.Status == session.StatusApproved {
		response.AccessToken = sess.AccessToken
	}

	WriteJSON(w, http.StatusOK, response)
}

// Login handles GET /login/{session_id} - displays login page
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract session ID from URL path
	sessionID := strings.TrimPrefix(r.URL.Path, "/login/")
	if sessionID == "" {
		WriteError(w, http.StatusBadRequest, "session ID required")
		return
	}

	// Verify session exists
	_, exists := h.sessionStore.Get(sessionID)
	if !exists {
		http.Error(w, "Session not found or expired", http.StatusNotFound)
		return
	}

	// Build provider list with their init URLs
	providers := make([]map[string]string, 0, len(h.oauthProviders))
	for name := range h.oauthProviders {
		providers = append(providers, map[string]string{
			"name": name,
			"url":  h.publicURL + "/oauth/init/" + name + "/" + sessionID,
		})
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.loginTemplate.Execute(w, map[string]interface{}{
		"SessionID": sessionID,
		"Providers": providers,
	}); err != nil {
		log.Printf("Failed to execute template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// OAuthCallback handles GET /oauth/callback
func (h *Handler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	state := r.URL.Query().Get("state")
	if state == "" {
		http.Error(w, "Missing state parameter", http.StatusBadRequest)
		return
	}

	// Decode provider name and session ID from state
	// Format: provider_name:session_id
	stateParts := strings.SplitN(state, ":", 2)
	if len(stateParts) != 2 {
		http.Error(w, "Invalid state parameter format", http.StatusBadRequest)
		return
	}

	providerName := stateParts[0]
	sessionID := stateParts[1]

	// Get the specific provider
	provider, exists := h.oauthProviders[providerName]
	if !exists {
		http.Error(w, "Unknown OAuth provider: "+providerName, http.StatusNotFound)
		return
	}

	// Verify session exists
	sess, exists := h.sessionStore.Get(sessionID)
	if !exists {
		http.Error(w, "Session not found or expired", http.StatusNotFound)
		return
	}

	// Check if session is already approved
	if sess.Status == session.StatusApproved {
		http.Error(w, "Session already approved", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	// Exchange code for token using the correct provider
	ctx := r.Context()
	token, err := provider.Exchange(ctx, code)
	if err != nil {
		log.Printf("Failed to exchange code with provider %s: %v", providerName, err)
		http.Error(w, "Failed to exchange authorization code", http.StatusInternalServerError)
		return
	}

	// Get username
	username, err := provider.GetUsername(ctx, token)
	if err != nil {
		log.Printf("Failed to get username from provider %s: %v", providerName, err)
		http.Error(w, "Failed to retrieve user information", http.StatusInternalServerError)
		return
	}

	// Verify membership
	if err := provider.VerifyMembership(ctx, token, username); err != nil {
		log.Printf("Membership verification failed with provider %s: %v", providerName, err)
		http.Error(w, "Access denied: not a member of required organization", http.StatusForbidden)
		return
	}

	// Approve session
	if !h.sessionStore.Approve(sessionID, username, token.AccessToken) {
		http.Error(w, "Failed to approve session", http.StatusInternalServerError)
		return
	}

	// Show success page
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Authentication Successful</title></head>
<body style="font-family: sans-serif; text-align: center; padding: 50px;">
<h1>✓ Authentication Successful!</h1>
<p>You can close this window and return to the CLI.</p>
<script>
	if (window.opener) { window.close(); }
</script>
</body>
</html>`))
}

// OAuthInit handles GET /oauth/init/{provider_name}/{session_id} - initiates OAuth flow
func (h *Handler) OAuthInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract provider name and session ID from URL path
	// Path format: /oauth/init/{provider_name}/{session_id}
	path := strings.TrimPrefix(r.URL.Path, "/oauth/init/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		http.Error(w, "Invalid path. Expected: /oauth/init/{provider_name}/{session_id}", http.StatusBadRequest)
		return
	}

	providerName := parts[0]
	sessionID := parts[1]

	// Get the provider
	provider, exists := h.oauthProviders[providerName]
	if !exists {
		http.Error(w, "Unknown OAuth provider: "+providerName, http.StatusNotFound)
		return
	}

	// Verify session exists
	_, exists = h.sessionStore.Get(sessionID)
	if !exists {
		http.Error(w, "Session not found or expired", http.StatusNotFound)
		return
	}

	// Encode provider name in state for callback identification
	// Format: provider_name:session_id
	state := providerName + ":" + sessionID

	// Redirect to OAuth provider with encoded state
	authURL := provider.AuthURL(state)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// CAPublicKey handles GET /api/v1/ca/pub
func (h *Handler) CAPublicKey(w http.ResponseWriter, r *http.Request, caPubKey string) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	WriteJSON(w, http.StatusOK, CAPublicKeyResponse{
		PublicKey: caPubKey,
	})
}

// SignUserCertificate handles POST /api/v1/sign/user
func (h *Handler) SignUserCertificate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Parse request body
	var req SignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request: %v", err)
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate public key
	if req.PublicKey == "" {
		WriteError(w, http.StatusBadRequest, "public_key required")
		return
	}

	// Validate access token
	if req.AccessToken == "" {
		WriteError(w, http.StatusBadRequest, "access_token required")
		return
	}

	// Find session by access token
	var username string
	found := false
	h.sessionStore.Range(func(id string, sess *session.LoginSession) bool {
		if sess.Status == session.StatusApproved && sess.AccessToken == req.AccessToken {
			username = sess.Username
			found = true
			return false
		}
		return true
	})

	if !found {
		WriteError(w, http.StatusUnauthorized, "invalid or expired access token")
		return
	}

	// Parse user public key
	userKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(req.PublicKey))
	if err != nil {
		log.Printf("Failed to parse public key: %v", err)
		WriteError(w, http.StatusBadRequest, "invalid public key")
		return
	}

	// Sign the certificate
	certBytes, err := cert.SignUserKey(
		h.caSigner,
		userKey,
		username,
		h.config.TTL,
		h.config.Extensions,
	)
	if err != nil {
		log.Printf("Failed to sign certificate: %v", err)
		WriteError(w, http.StatusInternalServerError, "failed to sign certificate")
		return
	}

	WriteJSON(w, http.StatusOK, SignResponse{
		Certificate: string(certBytes),
	})
}

// SignHostCertificate handles POST /api/v1/sign/host
func (h *Handler) SignHostCertificate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Parse request body
	var req HostSignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request: %v", err)
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate host public key
	if req.PublicKey == "" {
		WriteError(w, http.StatusBadRequest, "public_key required")
		return
	}

	// Validate principals
	if len(req.Principals) == 0 {
		WriteError(w, http.StatusBadRequest, "principals required")
		return
	}

	// Parse host public key
	hostKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(req.PublicKey))
	if err != nil {
		log.Printf("Failed to parse host public key: %v", err)
		WriteError(w, http.StatusBadRequest, "invalid public key")
		return
	}

	// Sign the host certificate
	certBytes, err := cert.SignHostKey(
		h.caSigner,
		hostKey,
		req.Principals,
		h.config.TTL,
	)
	if err != nil {
		log.Printf("Failed to sign host certificate: %v", err)
		WriteError(w, http.StatusInternalServerError, "failed to sign host certificate")
		return
	}

	WriteJSON(w, http.StatusOK, SignResponse{
		Certificate: string(certBytes),
	})
}
