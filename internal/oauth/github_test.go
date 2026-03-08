package oauth

import (
	"context"
	"testing"

	"golang.org/x/oauth2"
)

func TestNewGitHubProvider(t *testing.T) {
	provider := NewGitHubProvider("client123", "secret123", "http://localhost/callback", "test-org")

	if provider.Name() != "github" {
		t.Errorf("expected name github, got %s", provider.Name())
	}

	if provider.config.ClientID != "client123" {
		t.Errorf("expected ClientID client123, got %s", provider.config.ClientID)
	}
	if provider.config.ClientSecret != "secret123" {
		t.Errorf("expected ClientSecret secret123, got %s", provider.config.ClientSecret)
	}
	if provider.config.RedirectURL != "http://localhost/callback" {
		t.Errorf("expected RedirectURL http://localhost/callback, got %s", provider.config.RedirectURL)
	}
	if provider.allowedOrg != "test-org" {
		t.Errorf("expected AllowedOrg test-org, got %s", provider.allowedOrg)
	}
}

func TestGitHubProviderAuthURL(t *testing.T) {
	provider := NewGitHubProvider("client123", "secret123", "http://localhost/callback", "test-org")

	authURL := provider.AuthURL("state123")
	if authURL == "" {
		t.Fatal("AuthURL returned empty string")
	}

	// URL should contain state parameter
	if !contains(authURL, "state=state123") {
		t.Error("AuthURL does not contain state parameter")
	}

	// URL should contain GitHub OAuth endpoint
	if !contains(authURL, "github.com") {
		t.Error("AuthURL does not contain github.com")
	}
}

func TestGitHubProviderExchangeInvalidCode(t *testing.T) {
	provider := NewGitHubProvider("invalid-client", "invalid-secret", "http://localhost/callback", "test-org")

	ctx := context.Background()
	_, err := provider.Exchange(ctx, "invalid-code")
	if err == nil {
		t.Error("Exchange should fail with invalid credentials")
	}
}

func TestGitHubProviderGetUsernameInvalidToken(t *testing.T) {
	provider := NewGitHubProvider("client123", "secret123", "http://localhost/callback", "test-org")

	ctx := context.Background()
	token := &oauth2.Token{
		AccessToken: "invalid-token",
	}

	_, err := provider.GetUsername(ctx, token)
	if err == nil {
		t.Error("GetUsername should fail with invalid token")
	}
}

func TestGitHubProviderVerifyMembershipNoOrg(t *testing.T) {
	// Provider with no org restriction
	provider := NewGitHubProvider("client123", "secret123", "http://localhost/callback", "")

	ctx := context.Background()
	token := &oauth2.Token{
		AccessToken: "any-token",
	}

	err := provider.VerifyMembership(ctx, token, "anyuser")
	if err != nil {
		t.Errorf("VerifyMembership should succeed when no org is required: %v", err)
	}
}

func TestGitHubProviderVerifyMembershipInvalidToken(t *testing.T) {
	provider := NewGitHubProvider("client123", "secret123", "http://localhost/callback", "test-org")

	ctx := context.Background()
	token := &oauth2.Token{
		AccessToken: "invalid-token",
	}

	err := provider.VerifyMembership(ctx, token, "anyuser")
	if err == nil {
		t.Error("VerifyMembership should fail with invalid token")
	}
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
