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
	"regexp"
	"runtime"
	"strings"
	"time"
)

var (
	version            = "0.0.0-dev"
	caInstallPath      = "/etc/ssh/sshifu_ca.pub"
	hostCertPath       = "/etc/ssh/ssh_host_ed25519_key-cert.pub"
	hostKeyPath        = "/etc/ssh/ssh_host_ed25519_key.pub"
	sshdConfigPath     = "/etc/ssh/sshd_config"
	defaultHTTPTimeout = 30 * time.Second
	defaultCertValidity = "720h" // 30 days for host certificates
)

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

	server := os.Args[1]

	if err := run(server); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✓ SSH server configured successfully!")
	fmt.Println("  SSH daemon has been restarted.")
}

func printVersion() {
	fmt.Printf("sshifu-trust version %s\n", version)
}

func printUsage() {
	fmt.Printf("sshifu-trust version %s\n", version)
	fmt.Println()
	fmt.Println("Usage: sudo sshifu-trust <sshifu-server>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  help, -h, --help     Show this help message")
	fmt.Println("  version, -v, --version  Show version information")
	fmt.Println()
	fmt.Println("Description:")
	fmt.Println("  Configure SSH server to trust sshifu Certificate Authority.")
	fmt.Println()
	fmt.Println("Arguments:")
	fmt.Println("  <sshifu-server>  URL or hostname of the sshifu server")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("  sudo sshifu-trust auth.example.com")
}

func run(server string) error {
	// Validate server URL format
	baseURL, err := normalizeServerURL(server)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}

	fmt.Println("SSH Server Trust Configuration")
	fmt.Println("==============================")
	fmt.Printf("sshifu-server: %s\n\n", baseURL)

	// Step 1: Download CA public key
	fmt.Println("1. Downloading CA public key...")
	caPubKey, err := downloadCAPublicKey(baseURL)
	if err != nil {
		return fmt.Errorf("failed to download CA public key: %w", err)
	}
	fmt.Printf("   CA public key downloaded (%d bytes)\n", len(caPubKey))

	// Step 2: Install CA key
	fmt.Println("2. Installing CA public key...")
	if err := installCAKey(caPubKey); err != nil {
		return fmt.Errorf("failed to install CA key: %w", err)
	}
	fmt.Printf("   CA key installed to %s\n", caInstallPath)

	// Step 3: Read host public key
	fmt.Println("3. Reading host public key...")
	hostPubKey, err := readHostPublicKey()
	if err != nil {
		return fmt.Errorf("failed to read host public key: %w", err)
	}
	fmt.Printf("   Host public key loaded (%d bytes)\n", len(hostPubKey))

	// Step 4: Get host principals
	principals, err := getHostPrincipals()
	if err != nil {
		return fmt.Errorf("failed to get host principals: %w", err)
	}
	fmt.Printf("   Host principals: %s\n", strings.Join(principals, ", "))

	// Step 5: Request host certificate
	fmt.Println("4. Requesting host certificate...")
	hostCert, err := requestHostCertificate(baseURL, hostPubKey, principals)
	if err != nil {
		return fmt.Errorf("failed to request host certificate: %w", err)
	}
	fmt.Printf("   Host certificate received (%d bytes)\n", len(hostCert))

	// Step 6: Install host certificate
	fmt.Println("5. Installing host certificate...")
	if err := installHostCertificate(hostCert); err != nil {
		return fmt.Errorf("failed to install host certificate: %w", err)
	}
	fmt.Printf("   Host certificate installed to %s\n", hostCertPath)

	// Step 7: Update sshd_config
	fmt.Println("6. Updating sshd_config...")
	if err := updateSSHDConfig(); err != nil {
		return fmt.Errorf("failed to update sshd_config: %w", err)
	}
	fmt.Println("   sshd_config updated")

	// Step 8: Restart sshd
	fmt.Println("7. Restarting SSH daemon...")
	if err := restartSSHD(); err != nil {
		return fmt.Errorf("failed to restart sshd: %w", err)
	}
	fmt.Println("   SSH daemon restarted")

	return nil
}

// normalizeServerURL converts a server argument to a proper HTTP URL
func normalizeServerURL(server string) (string, error) {
	// If no scheme provided, assume https
	if !strings.HasPrefix(server, "http://") && !strings.HasPrefix(server, "https://") {
		server = "https://" + server
	}

	u, err := url.Parse(server)
	if err != nil {
		return "", err
	}

	// Ensure we have a valid URL
	if u.Host == "" {
		return "", fmt.Errorf("invalid server URL: %s", server)
	}

	// Remove trailing slash
	baseURL := strings.TrimSuffix(u.String(), "/")
	return baseURL, nil
}

// downloadCAPublicKey fetches the CA public key from sshifu-server
func downloadCAPublicKey(baseURL string) (string, error) {
	client := &http.Client{Timeout: defaultHTTPTimeout}

	resp, err := client.Get(baseURL + "/api/v1/ca/pub")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		PublicKey string `json:"public_key"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.PublicKey, nil
}

// installCAKey writes the CA public key to the SSH directory
func installCAKey(caPubKey string) error {
	// Ensure /etc/ssh directory exists
	if err := os.MkdirAll(filepath.Dir(caInstallPath), 0755); err != nil {
		return err
	}

	// Write CA key with proper permissions
	if err := os.WriteFile(caInstallPath, []byte(caPubKey+"\n"), 0644); err != nil {
		return err
	}

	return nil
}

// readHostPublicKey reads the host's ED25519 public key
func readHostPublicKey() (string, error) {
	data, err := os.ReadFile(hostKeyPath)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

// getHostPrincipals returns the hostnames and IPs for the certificate
func getHostPrincipals() ([]string, error) {
	var principals []string

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	principals = append(principals, hostname)

	// Try to get additional hostnames from /etc/hosts
	hosts, err := readEtcHosts()
	if err == nil {
		for _, h := range hosts {
			if h != hostname && h != "localhost" && h != "localhost.localdomain" {
				principals = append(principals, h)
			}
		}
	}

	// Add localhost variants
	principals = append(principals, "localhost", "localhost.localdomain")

	return principals, nil
}

// readEtcHosts parses /etc/hosts and returns hostnames
func readEtcHosts() ([]string, error) {
	data, err := os.ReadFile("/etc/hosts")
	if err != nil {
		return nil, err
	}

	var hosts []string
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		// Skip non-IP addresses
		if !isValidIP(parts[0]) {
			continue
		}

		// Add all hostnames from this line
		for _, h := range parts[1:] {
			hosts = append(hosts, h)
		}
	}

	return hosts, nil
}

// isValidIP checks if a string looks like an IP address
func isValidIP(s string) bool {
	// Simple regex for IPv4 and IPv6
	ipv4 := regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)
	ipv6 := regexp.MustCompile(`^[0-9a-fA-F:]+$`)
	return ipv4.MatchString(s) || (ipv6.MatchString(s) && strings.Contains(s, ":"))
}

// requestHostCertificate requests a signed host certificate from sshifu-server
func requestHostCertificate(baseURL, hostPubKey string, principals []string) (string, error) {
	client := &http.Client{Timeout: defaultHTTPTimeout}

	reqBody := struct {
		PublicKey string   `json:"public_key"`
		Principals []string `json:"principals"`
		TTL       string   `json:"ttl,omitempty"`
	}{
		PublicKey: hostPubKey,
		Principals: principals,
		TTL:       defaultCertValidity,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := client.Post(baseURL+"/api/v1/sign/host", "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status: %d - %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		Certificate string `json:"certificate"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.Certificate, nil
}

// installHostCertificate writes the host certificate to disk
func installHostCertificate(cert string) error {
	// Write certificate with proper permissions
	if err := os.WriteFile(hostCertPath, []byte(cert), 0644); err != nil {
		return err
	}

	return nil
}

// updateSSHDConfig modifies sshd_config to trust the Sshifu CA
func updateSSHDConfig() error {
	// Read current config
	existing, err := os.ReadFile(sshdConfigPath)
	if err != nil {
		return err
	}

	content := string(existing)
	lines := strings.Split(content, "\n")

	// Track if we need to add directives
	hasTrustedCA := false
	hasHostCert := false

	// Update existing lines
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Check for existing TrustedUserCAKeys
		if strings.HasPrefix(trimmed, "TrustedUserCAKeys") {
			lines[i] = "TrustedUserCAKeys " + caInstallPath
			hasTrustedCA = true
		}
		
		// Check for existing HostCertificate
		if strings.HasPrefix(trimmed, "HostCertificate") {
			lines[i] = "HostCertificate " + hostCertPath
			hasHostCert = true
		}
	}

	// Add missing directives
	newLines := lines
	if !hasTrustedCA {
		newLines = append(newLines, "TrustedUserCAKeys "+caInstallPath)
	}
	if !hasHostCert {
		newLines = append(newLines, "HostCertificate "+hostCertPath)
	}

	// Write updated config
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(sshdConfigPath, []byte(newContent), 0600); err != nil {
		return err
	}

	return nil
}

// restartSSHD restarts the SSH daemon
func restartSSHD() error {
	// Detect init system and use appropriate command
	if isSystemd() {
		return runCommand("systemctl", "restart", "sshd")
	}

	// Fallback to service command
	return runCommand("service", "ssh", "restart")
}

// isSystemd checks if systemd is the init system
func isSystemd() bool {
	// Check if systemctl is available
	_, err := exec.LookPath("systemctl")
	return err == nil
}

// runCommand executes a shell command
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Helper functions for testing

// joinURL joins a base URL with a path
func joinURL(base, path string) string {
	base = strings.TrimSuffix(base, "/")
	path = strings.TrimPrefix(path, "/")
	return base + "/" + path
}

// getHostKeyPath returns the path to the host key file
func getHostKeyPath() string {
	return hostKeyPath
}

// getHostCertPath returns the path to the host certificate file
func getHostCertPath() string {
	return hostCertPath
}

// getCAInstallPath returns the path where CA key is installed
func getCAInstallPath() string {
	return caInstallPath
}

// getSSHDConfigPath returns the path to sshd_config
func getSSHDConfigPath() string {
	return sshdConfigPath
}

// detectOS returns the current operating system
func detectOS() string {
	return runtime.GOOS
}
