package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
)

// OIDCProvider implements OAuth for OIDC providers
type OIDCProvider struct {
	config              *oauth2.Config
	httpClient          *http.Client
	issuer              string
	principalFieldName  string
	userInfoEndpoint    string
}

// OIDCWellKnown represents the OIDC well-known configuration
type OIDCWellKnown struct {
	Issuer            string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint     string `json:"token_endpoint"`
	UserInfoEndpoint  string `json:"userinfo_endpoint"`
	JWKSURI           string `json:"jwks_uri"`
}

// NewOIDCProvider creates a new OIDC OAuth provider
func NewOIDCProvider(issuer, clientID, clientSecret, redirectURL, principalFieldName string) (*OIDCProvider, error) {
	// Fetch OIDC configuration from well-known endpoint
	wellKnownURL, err := url.JoinPath(issuer, "/.well-known/openid-configuration")
	if err != nil {
		return nil, fmt.Errorf("failed to build well-known URL: %w", err)
	}

	resp, err := http.Get(wellKnownURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OIDC configuration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OIDC configuration returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OIDC configuration: %w", err)
	}

	var wellKnown OIDCWellKnown
	if err := json.Unmarshal(body, &wellKnown); err != nil {
		return nil, fmt.Errorf("failed to parse OIDC configuration: %w", err)
	}

	// Use default principal field name if not specified
	if principalFieldName == "" {
		principalFieldName = "preferred_username"
	}

	return &OIDCProvider{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  wellKnown.AuthorizationEndpoint,
				TokenURL: wellKnown.TokenEndpoint,
			},
		},
		httpClient:         &http.Client{},
		issuer:             issuer,
		principalFieldName: principalFieldName,
		userInfoEndpoint:   wellKnown.UserInfoEndpoint,
	}, nil
}

func (p *OIDCProvider) Name() string {
	return "oidc"
}

func (p *OIDCProvider) AuthURL(state string) string {
	return p.config.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

func (p *OIDCProvider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return p.config.Exchange(ctx, code)
}

func (p *OIDCProvider) GetUsername(ctx context.Context, token *oauth2.Token) (string, error) {
	// Fetch user info from OIDC userinfo endpoint
	client := p.config.Client(ctx, token)
	resp, err := client.Get(p.userInfoEndpoint)
	if err != nil {
		return "", fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OIDC userinfo API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var userInfo map[string]interface{}
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return "", fmt.Errorf("failed to parse user info: %w", err)
	}

	// Extract username from configured field
	username, ok := userInfo[p.principalFieldName].(string)
	if !ok || username == "" {
		// Fallback to email if principal field not found
		if email, ok := userInfo["email"].(string); ok && email != "" {
			// Extract username part from email
			if idx := strings.Index(email, "@"); idx > 0 {
				return email[:idx], nil
			}
			return email, nil
		}
		return "", fmt.Errorf("field %q not found in user info", p.principalFieldName)
	}

	return username, nil
}

func (p *OIDCProvider) VerifyMembership(ctx context.Context, token *oauth2.Token, username string) error {
	// OIDC doesn't have a standard org membership check
	// This is a no-op for basic OIDC providers
	// Custom implementations can override this behavior
	return nil
}
