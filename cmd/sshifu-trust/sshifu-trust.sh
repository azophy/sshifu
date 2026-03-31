#!/usr/bin/env bash
#
# sshifu-trust - Configure SSH server to trust sshifu Certificate Authority
#
# Usage: 
#   sudo sshifu-trust.sh <sshifu-server>
#   SSHIFU_SERVER=auth.example.com sudo sshifu-trust.sh
#   curl -fsSL <url> | sudo bash -s -- auth.example.com
#
# Example: curl -fsSL https://raw.githubusercontent.com/azophy/sshifu/main/cmd/sshifu-trust/sshifu-trust.sh | sudo bash -s -- auth.example.com
#

set -e

VERSION="0.7.7"
CA_INSTALL_PATH="/etc/ssh/sshifu_ca.pub"
HOST_CERT_PATH="/etc/ssh/ssh_host_ed25519_key-cert.pub"
HOST_KEY_PATH="/etc/ssh/ssh_host_ed25519_key.pub"
SSHD_CONFIG_PATH="/etc/ssh/sshd_config"
HTTP_TIMEOUT=30

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_error() {
    echo -e "${RED}Error: $1${NC}" >&2
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}  $1${NC}"
}

print_version() {
    echo "sshifu-trust version $VERSION"
}

print_usage() {
    echo "sshifu-trust version $VERSION"
    echo ""
    echo "Usage: sudo sshifu-trust [options] <sshifu-server>"
    echo ""
    echo "Commands:"
    echo "  help, -h, --help     Show this help message"
    echo "  version, -v, --version  Show version information"
    echo ""
    echo "Description:"
    echo "  Configure SSH server to trust sshifu Certificate Authority."
    echo ""
    echo "Arguments:"
    echo "  <sshifu-server>  URL or hostname of the sshifu server"
    echo ""
    echo "Environment Variables:"
    echo "  SSHIFU_SERVER    sshifu server URL (used if no argument provided)"
    echo ""
    echo "Examples:"
    echo "  sudo sshifu-trust auth.example.com"
    echo "  SSHIFU_SERVER=auth.example.com sudo sshifu-trust"
    echo "  sudo sshifu-trust  # (will prompt for server)"
}

# Check if running as root, if not re-exec with sudo
check_root() {
    if [[ $EUID -ne 0 ]]; then
        print_error "This script must be run as root (will re-exec with sudo)"
        
        # Check if sudo is available
        if ! command -v sudo &> /dev/null; then
            print_error "sudo is required but not installed"
            exit 1
        fi
        
        # Re-exec this script with sudo
        # If script is from stdin (piped), download it first
        if [[ -t 0 ]]; then
            # Script file is available
            exec sudo bash "$0" "$@"
        else
            # Script was piped in, need to download it again
            # Try to get the script URL from common sources
            local script_url="${SSHIFU_TRUST_URL:-}"
            
            if [[ -z "$script_url" ]]; then
                # Try to detect if we're running from a known URL
                if [[ -n "${SSHIFU_SERVER:-}" ]]; then
                    # Extract domain from SSHIFU_SERVER to guess script location
                    local domain="${SSHIFU_SERVER#https://}"
                    domain="${domain#http://}"
                    domain="${domain%%/*}"
                    script_url="https://raw.githubusercontent.com/azophy/sshifu/main/cmd/sshifu-trust/sshifu-trust.sh"
                else
                    script_url="https://raw.githubusercontent.com/azophy/sshifu/main/cmd/sshifu-trust/sshifu-trust.sh"
                fi
            fi
            
            print_info "Re-executing from: $script_url"
            exec curl -fsSL "$script_url" | sudo bash -s -- "$@"
        fi
    fi
}

# Normalize server URL
normalize_server_url() {
    local server="$1"
    if [[ ! "$server" =~ ^https?:// ]]; then
        server="https://$server"
    fi
    # Remove trailing slash
    server="${server%/}"
    echo "$server"
}

# Download CA public key
download_ca_public_key() {
    local base_url="$1"
    local response
    local public_key

    response=$(curl -sf --max-time "$HTTP_TIMEOUT" "${base_url}/api/v1/ca/pub")
    if [[ $? -ne 0 ]]; then
        print_error "Failed to download CA public key from $base_url"
        return 1
    fi

    public_key=$(echo "$response" | grep -o '"public_key"[[:space:]]*:[[:space:]]*"[^"]*"' | cut -d'"' -f4)
    if [[ -z "$public_key" ]]; then
        print_error "Failed to parse CA public key from response"
        return 1
    fi

    echo "$public_key"
}

# Install CA key
install_ca_key() {
    local ca_pub_key="$1"
    
    mkdir -p "$(dirname "$CA_INSTALL_PATH")"
    echo "$ca_pub_key" > "$CA_INSTALL_PATH"
    chmod 644 "$CA_INSTALL_PATH"
}

# Read host public key
read_host_public_key() {
    if [[ ! -f "$HOST_KEY_PATH" ]]; then
        print_error "Host public key not found at $HOST_KEY_PATH"
        return 1
    fi
    cat "$HOST_KEY_PATH"
}

# Get host principals
get_host_principals() {
    local principals=()
    local hostname
    
    hostname=$(hostname)
    principals+=("$hostname")
    
    # Add localhost variants
    principals+=("localhost" "localhost.localdomain")
    
    # Try to get additional hostnames from /etc/hosts
    if [[ -f /etc/hosts ]]; then
        while IFS= read -r line; do
            # Skip comments and empty lines
            [[ "$line" =~ ^[[:space:]]*# ]] && continue
            [[ -z "$line" ]] && continue
            
            # Parse hosts from line (skip IP address)
            read -ra parts <<< "$line"
            for ((i=1; i<${#parts[@]}; i++)); do
                local host="${parts[$i]}"
                # Skip if already in principals or is localhost
                if [[ ! " ${principals[*]} " =~ " ${host} " ]]; then
                    principals+=("$host")
                fi
            done
        done < /etc/hosts
    fi
    
    echo "${principals[@]}"
}

# Request host certificate
request_host_certificate() {
    local base_url="$1"
    local host_pub_key="$2"
    local principals="$3"
    
    local principals_json
    principals_json=$(echo "$principals" | tr ' ' '\n' | sed 's/.*/"&"/' | paste -sd ',' -)
    
    local response
    response=$(curl -sf --max-time "$HTTP_TIMEOUT" \
        -X POST "${base_url}/api/v1/sign/host" \
        -H "Content-Type: application/json" \
        -d "{\"public_key\":\"$host_pub_key\",\"principals\":[$principals_json],\"ttl\":\"720h\"}")
    
    if [[ $? -ne 0 ]]; then
        print_error "Failed to request host certificate from $base_url"
        return 1
    fi
    
    local certificate
    certificate=$(echo "$response" | grep -o '"certificate"[[:space:]]*:[[:space:]]*"[^"]*"' | cut -d'"' -f4)
    if [[ -z "$certificate" ]]; then
        print_error "Failed to parse host certificate from response"
        return 1
    fi
    
    echo "$certificate"
}

# Install host certificate
install_host_certificate() {
    local cert="$1"
    echo "$cert" > "$HOST_CERT_PATH"
    chmod 644 "$HOST_CERT_PATH"
}

# Update sshd_config
update_sshd_config() {
    local has_trusted_ca=false
    local has_host_cert=false
    local temp_file
    temp_file=$(mktemp)
    
    while IFS= read -r line || [[ -n "$line" ]]; do
        local trimmed="${line#"${line%%[![:space:]]*}"}"
        
        if [[ "$trimmed" =~ ^TrustedUserCAKeys ]]; then
            echo "TrustedUserCAKeys $CA_INSTALL_PATH" >> "$temp_file"
            has_trusted_ca=true
        elif [[ "$trimmed" =~ ^HostCertificate ]]; then
            echo "HostCertificate $HOST_CERT_PATH" >> "$temp_file"
            has_host_cert=true
        else
            echo "$line" >> "$temp_file"
        fi
    done < "$SSHD_CONFIG_PATH"
    
    # Add missing directives
    if [[ "$has_trusted_ca" == false ]]; then
        echo "TrustedUserCAKeys $CA_INSTALL_PATH" >> "$temp_file"
    fi
    if [[ "$has_host_cert" == false ]]; then
        echo "HostCertificate $HOST_CERT_PATH" >> "$temp_file"
    fi
    
    mv "$temp_file" "$SSHD_CONFIG_PATH"
    chmod 600 "$SSHD_CONFIG_PATH"
}

# Restart SSH daemon
restart_sshd() {
    local ssh_services=("sshd" "ssh" "openssh-daemon" "openssh")
    
    # Try systemctl first
    if command -v systemctl &> /dev/null; then
        for service in "${ssh_services[@]}"; do
            print_info "Trying systemctl restart $service... "
            if systemctl restart "$service" 2>/dev/null; then
                print_success "success"
                return 0
            fi
            echo "failed"
        done
    fi
    
    # Fallback to service command
    for service in "${ssh_services[@]}"; do
        print_info "Trying service $service restart... "
        if service "$service" restart 2>/dev/null; then
            print_success "success"
            return 0
        fi
        echo "failed"
    done
    
    print_error "Failed to restart SSH service (tried: ${ssh_services[*]})"
    return 1
}

# Main function
main() {
    # Handle special commands
    case "${1:-}" in
        -h|-help|--help|help)
            print_usage
            exit 0
            ;;
        -v|-version|--version|version)
            print_version
            exit 0
            ;;
    esac
    
    # Get server from argument, environment, or prompt
    local server="${1:-}"
    
    # Check environment variable if no argument provided
    if [[ -z "$server" ]]; then
        server="${SSHIFU_SERVER:-}"
    fi
    
    # Prompt user if still no server (only if stdin is a terminal)
    if [[ -z "$server" ]]; then
        if [[ -t 0 ]]; then
            echo "SSH Server Trust Configuration"
            echo "=============================="
            echo ""
            read -rp "Enter sshifu-server URL (e.g., auth.example.com): " server
            if [[ -z "$server" ]]; then
                print_error "No server specified. Exiting."
                exit 1
            fi
        else
            print_error "No server specified. Use:"
            echo "  - Command argument: $0 auth.example.com"
            echo "  - Environment variable: SSHIFU_SERVER=auth.example.com $0"
            echo "  - Curl pattern: curl ... | sudo bash -s -- auth.example.com"
            exit 1
        fi
    fi
    
    # Check root
    check_root
    
    # Normalize server URL
    local base_url
    base_url=$(normalize_server_url "$server")
    
    echo "SSH Server Trust Configuration"
    echo "=============================="
    echo "sshifu-server: $base_url"
    echo ""
    
    # Step 1: Download CA public key
    echo "1. Downloading CA public key..."
    local ca_pub_key
    ca_pub_key=$(download_ca_public_key "$base_url")
    if [[ $? -ne 0 ]]; then
        exit 1
    fi
    print_success "CA public key downloaded (${#ca_pub_key} bytes)"
    
    # Step 2: Install CA key
    echo "2. Installing CA public key..."
    install_ca_key "$ca_pub_key"
    print_success "CA key installed to $CA_INSTALL_PATH"
    
    # Step 3: Read host public key
    echo "3. Reading host public key..."
    local host_pub_key
    host_pub_key=$(read_host_public_key)
    if [[ $? -ne 0 ]]; then
        exit 1
    fi
    print_success "Host public key loaded (${#host_pub_key} bytes)"
    
    # Step 4: Get host principals
    local principals
    principals=$(get_host_principals)
    print_info "Host principals: $principals"
    
    # Step 5: Request host certificate
    echo "4. Requesting host certificate..."
    local host_cert
    host_cert=$(request_host_certificate "$base_url" "$host_pub_key" "$principals")
    if [[ $? -ne 0 ]]; then
        exit 1
    fi
    print_success "Host certificate received (${#host_cert} bytes)"
    
    # Step 6: Install host certificate
    echo "5. Installing host certificate..."
    install_host_certificate "$host_cert"
    print_success "Host certificate installed to $HOST_CERT_PATH"
    
    # Step 7: Update sshd_config
    echo "6. Updating sshd_config..."
    update_sshd_config
    print_success "sshd_config updated"
    
    # Step 8: Restart sshd
    echo "7. Restarting SSH daemon..."
    restart_sshd
    if [[ $? -ne 0 ]]; then
        exit 1
    fi
    print_success "SSH daemon restarted"
    
    echo ""
    print_success "SSH server configured successfully!"
    print_info "SSH daemon has been restarted."
}

main "$@"
