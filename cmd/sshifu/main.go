package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/azophy/sshifu/internal/api"
	"github.com/azophy/sshifu/internal/ssh"
)

var version = "0.0.0-dev"

// Config holds the CLI configuration
type Config struct {
	ServerURL    string
	IdentityFile string
	SSHCmd       string
	SSHArgs      []string
}

// LoginStatus represents the login session status
type LoginStatus struct {
	Status      string `json:"status"`
	AccessToken string `json:"access_token,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Handle special commands
	switch os.Args[1] {
	case "-h", "-help", "--help", "help":
		printUsage()
		os.Exit(0)
	case "-v", "-version", "--version", "version":
		printVersion()
		os.Exit(0)
	}

	// Parse arguments
	config, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Run the main workflow
	if err := run(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printVersion() {
	fmt.Printf("sshifu version %s\n", version)
}

func printUsage() {
	fmt.Printf("sshifu version %s\n", version)
	fmt.Println()
	fmt.Println("Usage: sshifu <sshifu-server> [ssh arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  help, -h, --help     Show this help message")
	fmt.Println("  version, -v, --version  Show version information")
	fmt.Println()
	fmt.Println("Arguments:")
	fmt.Println("  <sshifu-server>      URL or hostname of the sshifu server")
	fmt.Println("  [ssh arguments]      Additional arguments to pass to SSH")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  sshifu auth.example.com                    # Connect to auth.example.com")
	fmt.Println("  sshifu auth.example.com target-server.com  # Connect to target-server.com")
	fmt.Println("  sshifu auth.example.com -i ~/.ssh/my_key target-server.com")
}

// parseArgs parses command-line arguments
func parseArgs(args []string) (*Config, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("sshifu-server URL is required")
	}

	config := &Config{
		ServerURL: args[0],
		SSHCmd:    "ssh",
	}

	// Parse remaining args for SSH options
	var sshArgs []string
	for i := 1; i < len(args); i++ {
		arg := args[i]

		// Check for -i option
		if arg == "-i" && i+1 < len(args) {
			config.IdentityFile = ssh.ExpandTilde(args[i+1])
			i++ // Skip next argument
		} else {
			sshArgs = append(sshArgs, arg)
		}
	}

	// If no SSH arguments provided, use server URL as the target
	if len(sshArgs) == 0 {
		sshArgs = []string{config.ServerURL}
	}

	config.SSHArgs = sshArgs
	return config, nil
}

// run executes the main CLI workflow
func run(config *Config) error {
	// Step 1: Find identity key
	identityKey, err := ssh.FindIdentityKey(config.IdentityFile)
	if err != nil {
		return fmt.Errorf("failed to find identity key: %w", err)
	}
	fmt.Printf("Using identity key: %s\n", identityKey)

	// Step 2: Check for existing valid certificate
	certPath := ssh.GetCertificatePath(identityKey)
	if valid, err := ssh.IsCertificateValid(certPath, ""); err == nil && valid {
		fmt.Println("Valid certificate found, skipping login")
		return execSSH(config, certPath, "")
	}

	// Step 3: Perform login flow
	fmt.Println("No valid certificate found, starting login flow...")

	accessToken, err := performLogin(config.ServerURL)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// Step 4: Request certificate
	fmt.Println("Requesting SSH certificate...")
	certificate, err := requestCertificate(config.ServerURL, identityKey, accessToken)
	if err != nil {
		return fmt.Errorf("failed to request certificate: %w", err)
	}

	// Step 5: Save certificate
	if err := saveCertificate(certPath, certificate); err != nil {
		return fmt.Errorf("failed to save certificate: %w", err)
	}
	fmt.Printf("Certificate saved to: %s\n", certPath)

	// Step 6: Download CA public key
	caKey, err := getCAKey(config.ServerURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to get CA key: %v\n", err)
	}

	// Step 7: Execute SSH
	return execSSH(config, certPath, caKey)
}

// performLogin performs the OAuth login flow
func performLogin(serverURL string) (string, error) {
	// Step 1: Start login session
	loginURL, sessionID, err := startLoginSession(serverURL)
	if err != nil {
		return "", fmt.Errorf("failed to start login session: %w", err)
	}

	fmt.Printf("\nPlease open this URL in your browser to authenticate:\n")
	fmt.Printf("  %s\n\n", loginURL)
	fmt.Println("Waiting for authentication...")

	// Step 2: Poll for status
	return pollLoginStatus(serverURL, sessionID)
}

// startLoginSession starts a new login session
func startLoginSession(serverURL string) (string, string, error) {
	apiURL := joinURL(serverURL, "/api/v1/login/start")

	resp, err := http.Post(apiURL, "application/json", nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to start login: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("login start failed with status: %d", resp.StatusCode)
	}

	var result api.LoginStartResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.LoginURL, result.SessionID, nil
}

// pollLoginStatus polls the login status until approved
func pollLoginStatus(serverURL, sessionID string) (string, error) {
	apiURL := joinURL(serverURL, "/api/v1/login/status/"+sessionID)

	// Poll every 2 seconds for up to 10 minutes
	maxDuration := 10 * time.Minute
	interval := 2 * time.Second
	startTime := time.Now()

	for time.Since(startTime) < maxDuration {
		time.Sleep(interval)

		resp, err := http.Get(apiURL)
		if err != nil {
			return "", fmt.Errorf("failed to get status: %w", err)
		}

		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			continue // Session might not be ready yet
		}

		var status LoginStatus
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			resp.Body.Close()
			return "", fmt.Errorf("failed to decode status: %w", err)
		}
		resp.Body.Close()

		if status.Status == "approved" {
			fmt.Println("Authentication successful!")
			return status.AccessToken, nil
		}

		fmt.Print(".")
	}

	return "", fmt.Errorf("login timeout")
}

// requestCertificate requests a user certificate from the server
func requestCertificate(serverURL, identityKeyPath, accessToken string) (string, error) {
	// Load public key
	pubKey, err := ssh.LoadPublicKey(identityKeyPath + ".pub")
	if err != nil {
		return "", fmt.Errorf("failed to load public key: %w", err)
	}

	pubKeyBytes := ssh.MarshalAuthorizedKey(pubKey)

	// Create request
	reqBody := api.SignRequest{
		PublicKey:   string(pubKeyBytes),
		AccessToken: accessToken,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send request
	apiURL := joinURL(serverURL, "/api/v1/sign/user")
	resp, err := http.Post(apiURL, "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to request certificate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("certificate request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result api.SignResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Certificate, nil
}

// saveCertificate saves the certificate to disk
func saveCertificate(certPath, certificate string) error {
	// Ensure .ssh directory exists
	dir := filepath.Dir(certPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create .ssh directory: %w", err)
	}

	// Write certificate
	if err := os.WriteFile(certPath, []byte(certificate), 0600); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	return nil
}

// getCAKey downloads the CA public key from the server
func getCAKey(serverURL string) (string, error) {
	apiURL := joinURL(serverURL, "/api/v1/ca/pub")

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("failed to get CA key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("CA key request failed with status: %d", resp.StatusCode)
	}

	var result api.CAPublicKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode CA key: %w", err)
	}

	return result.PublicKey, nil
}

// execSSH executes the SSH command with the certificate
func execSSH(config *Config, certPath, caKey string) error {
	fmt.Printf("Connecting via SSH...\n")

	// Build SSH command with certificate
	args := []string{
		"-o", "CertificateFile=" + certPath,
	}

	// Add CA key as a temporary known_hosts file if available
	if caKey != "" {
		// Create a temporary known_hosts file with the CA key
		tmpFile, err := createTempKnownHosts(caKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create temporary known_hosts: %v\n", err)
		} else {
			defer os.Remove(tmpFile) // Clean up temp file
			args = append(args, "-o", "UserKnownHostsFile="+tmpFile)
		}
	}

	args = append(args, config.SSHArgs...)

	cmd := exec.Command(config.SSHCmd, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("SSH command failed: %w", err)
	}

	return nil
}

// createTempKnownHosts creates a temporary known_hosts file with the CA key
func createTempKnownHosts(caKey string) (string, error) {
	// Create @cert-authority entry
	entry := fmt.Sprintf("@cert-authority * %s\n", caKey)

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "sshifu_known_hosts_*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(entry); err != nil {
		return "", fmt.Errorf("failed to write to temp file: %w", err)
	}

	return tmpFile.Name(), nil
}

// joinURL joins a base URL with a path
func joinURL(base, path string) string {
	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" {
		// If base is not a valid URL or has no scheme, assume it's a hostname
		u = &url.URL{
			Scheme: "https",
			Host:   base,
		}
	}
	return u.ResolveReference(&url.URL{Path: path}).String()
}
