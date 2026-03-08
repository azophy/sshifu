package api

import (
	"crypto/rand"
	"encoding/hex"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/azophy/sshifu/internal/oauth"
	"github.com/azophy/sshifu/internal/session"
)

// Handler manages HTTP request handling
type Handler struct {
	sessionStore *session.Store
	oauthProvider oauth.Provider
	publicURL     string
	loginTemplate *template.Template
}

// NewHandler creates a new API handler
func NewHandler(store *session.Store, provider oauth.Provider, publicURL string) (*Handler, error) {
	// Try multiple paths for the template (handles different working directories)
	var tmpl *template.Template
	var err error
	
	// Try current directory first (for running from project root)
	tmpl, err = template.ParseFiles("web/login.html")
	if err != nil {
		// Try parent directory (for running from cmd/sshifu-server)
		tmpl, err = template.ParseFiles("../web/login.html")
	}
	if err != nil {
		// Try two levels up (for running tests from internal/api)
		tmpl, err = template.ParseFiles("../../web/login.html")
	}
	if err != nil {
		return nil, err
	}

	return &Handler{
		sessionStore:  store,
		oauthProvider: provider,
		publicURL:     publicURL,
		loginTemplate: tmpl,
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

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.loginTemplate.Execute(w, map[string]string{"SessionID": sessionID}); err != nil {
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

	sessionID := r.URL.Query().Get("state")
	if sessionID == "" {
		http.Error(w, "Missing state parameter", http.StatusBadRequest)
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

	// Exchange code for token
	ctx := r.Context()
	token, err := h.oauthProvider.Exchange(ctx, code)
	if err != nil {
		log.Printf("Failed to exchange code: %v", err)
		http.Error(w, "Failed to exchange authorization code", http.StatusInternalServerError)
		return
	}

	// Get username
	username, err := h.oauthProvider.GetUsername(ctx, token)
	if err != nil {
		log.Printf("Failed to get username: %v", err)
		http.Error(w, "Failed to retrieve user information", http.StatusInternalServerError)
		return
	}

	// Verify org membership
	if err := h.oauthProvider.VerifyMembership(ctx, token, username); err != nil {
		log.Printf("Membership verification failed: %v", err)
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

// OAuthInit handles GET /oauth/github/{session_id} - initiates OAuth flow
func (h *Handler) OAuthInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract session ID from URL path
	sessionID := strings.TrimPrefix(r.URL.Path, "/oauth/github/")
	if sessionID == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	// Verify session exists
	_, exists := h.sessionStore.Get(sessionID)
	if !exists {
		http.Error(w, "Session not found or expired", http.StatusNotFound)
		return
	}

	// Redirect to OAuth provider
	authURL := h.oauthProvider.AuthURL(sessionID)
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
	// This will be implemented in Milestone 6
	WriteError(w, http.StatusNotImplemented, "not implemented")
}

// SignHostCertificate handles POST /api/v1/sign/host
func (h *Handler) SignHostCertificate(w http.ResponseWriter, r *http.Request) {
	// This will be implemented in Milestone 8
	WriteError(w, http.StatusNotImplemented, "not implemented")
}
