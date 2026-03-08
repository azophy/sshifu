package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// Provider handles OAuth authentication
type Provider interface {
	// Name returns the provider name
	Name() string
	// AuthURL returns the OAuth authorization URL
	AuthURL(state string) string
	// Exchange exchanges a code for a token
	Exchange(ctx context.Context, code string) (*oauth2.Token, error)
	// GetUsername retrieves the username from the OAuth token
	GetUsername(ctx context.Context, token *oauth2.Token) (string, error)
	// VerifyMembership verifies the user belongs to required org/group
	VerifyMembership(ctx context.Context, token *oauth2.Token, username string) error
}

// GitHubProvider implements OAuth for GitHub
type GitHubProvider struct {
	config       *oauth2.Config
	allowedOrg   string
	httpClient   *http.Client
}

// NewGitHubProvider creates a new GitHub OAuth provider
func NewGitHubProvider(clientID, clientSecret, redirectURL, allowedOrg string) *GitHubProvider {
	return &GitHubProvider{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"read:org"},
			Endpoint:     github.Endpoint,
		},
		allowedOrg: allowedOrg,
		httpClient: &http.Client{},
	}
}

func (p *GitHubProvider) Name() string {
	return "github"
}

func (p *GitHubProvider) AuthURL(state string) string {
	return p.config.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

func (p *GitHubProvider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return p.config.Exchange(ctx, code)
}

func (p *GitHubProvider) GetUsername(ctx context.Context, token *oauth2.Token) (string, error) {
	client := p.config.Client(ctx, token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return "", fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var user struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal(body, &user); err != nil {
		return "", fmt.Errorf("failed to parse user info: %w", err)
	}

	return user.Login, nil
}

func (p *GitHubProvider) VerifyMembership(ctx context.Context, token *oauth2.Token, username string) error {
	if p.allowedOrg == "" {
		return nil
	}

	client := p.config.Client(ctx, token)
	resp, err := client.Get(fmt.Sprintf("https://api.github.com/user/orgs"))
	if err != nil {
		return fmt.Errorf("failed to get orgs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var orgs []struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal(body, &orgs); err != nil {
		return fmt.Errorf("failed to parse orgs: %w", err)
	}

	for _, org := range orgs {
		if org.Login == p.allowedOrg {
			return nil
		}
	}

	return fmt.Errorf("user %s is not a member of org %s", username, p.allowedOrg)
}
