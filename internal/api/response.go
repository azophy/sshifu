package api

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// WriteError writes an error response
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, ErrorResponse{Error: message})
}

// LoginStartResponse is the response for login start endpoint
type LoginStartResponse struct {
	SessionID string `json:"session_id"`
	LoginURL  string `json:"login_url"`
}

// LoginStatusResponse is the response for login status endpoint
type LoginStatusResponse struct {
	Status      string `json:"status"`
	AccessToken string `json:"access_token,omitempty"`
}

// SignRequest is the request for certificate signing
type SignRequest struct {
	PublicKey   string `json:"public_key"`
	AccessToken string `json:"access_token"`
}

// HostSignRequest is the request for host certificate signing
type HostSignRequest struct {
	PublicKey   string   `json:"public_key"`
	Principals  []string `json:"principals"`
}

// SignResponse is the response for certificate signing
type SignResponse struct {
	Certificate string `json:"certificate"`
}

// CAPublicKeyResponse is the response for CA public key endpoint
type CAPublicKeyResponse struct {
	PublicKey string `json:"public_key"`
}
