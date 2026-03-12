#!/bin/bash
set -e

# ============================================================================
# Configuration
# ============================================================================
readonly REPO="azophy/sshifu"
readonly SCRIPT_NAME="sshifu-install"
readonly DEFAULT_APPS="sshifu,sshifu-server,sshifu-trust"
readonly VALID_APPS="sshifu sshifu-server sshifu-trust"

# Defaults
APPS_TO_INSTALL="${INSTALL_APP:-}"
VERSION="${INSTALL_VERSION:-latest}"
INSTALL_PREFIX="${INSTALL_PREFIX:-$HOME/.sshifu}"
NO_PATH="${INSTALL_NO_PATH:-0}"
VERBOSE="${INSTALL_VERBOSE:-0}"
USE_SUDO=0

# ============================================================================
# Utility Functions
# ============================================================================
log_info() { echo "[INFO] $*"; }
log_error() { echo "[ERROR] $*" >&2; }
log_verbose() { [ "$VERBOSE" = "1" ] && echo "[DEBUG] $*" || true; }

die() {
    log_error "$1"
    exit 1
}

# ============================================================================
# Argument Parsing
# ============================================================================
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --app)
                APPS_TO_INSTALL="$2"
                shift 2
                ;;
            --version)
                VERSION="$2"
                shift 2
                ;;
            --prefix)
                INSTALL_PREFIX="$2"
                shift 2
                ;;
            --system)
                INSTALL_PREFIX="/usr/local"
                USE_SUDO=1
                shift
                ;;
            --no-path)
                NO_PATH=1
                shift
                ;;
            --verbose)
                VERBOSE=1
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                die "Unknown option: $1. Use --help for usage."
                ;;
        esac
    done

    # Default to all apps if none specified
    if [ -z "$APPS_TO_INSTALL" ]; then
        APPS_TO_INSTALL="$DEFAULT_APPS"
    fi
}

# ============================================================================
# System Detection
# ============================================================================
detect_os() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    case $os in
        linux|darwin)
            echo "$os"
            ;;
        mingw*|msys*|cygwin*)
            echo "windows"
            ;;
        *)
            die "Unsupported OS: $os"
            ;;
    esac
}

detect_arch() {
    local arch=$(uname -m)
    case $arch in
        x86_64|x64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        armv7l|armhf)
            echo "arm"
            ;;
        *)
            die "Unsupported architecture: $arch"
            ;;
    esac
}

# ============================================================================
# Release Detection
# ============================================================================
get_latest_version() {
    local response
    response=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null)

    if [ $? -ne 0 ]; then
        die "Failed to fetch latest release. Check your internet connection."
    fi

    local version
    version=$(echo "$response" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' | sed 's/^v//')

    if [ -z "$version" ]; then
        die "Failed to parse version from GitHub API response"
    fi

    echo "$version"
}

# ============================================================================
# Download & Install
# ============================================================================
download_and_install_app() {
    local app="$1"
    local os="$2"
    local arch="$3"
    local version="$4"

    # Determine archive format based on OS
    local archive_ext
    local extract_cmd
    if [ "$os" = "windows" ]; then
        archive_ext="zip"
        extract_cmd="unzip -o"
    else
        archive_ext="tar.gz"
        extract_cmd="tar -xzf"
    fi

    local asset_name="${app}-${os}-${arch}.${archive_ext}"
    local download_url="https://github.com/${REPO}/releases/download/v${version}/${asset_name}"

    log_info "Downloading ${app}..."
    log_verbose "URL: ${download_url}"

    # Download to temp file
    local temp_file=$(mktemp)
    if ! curl -fsSL -o "$temp_file" "$download_url"; then
        log_error "Failed to download ${app}"
        rm -f "$temp_file"
        return 1
    fi

    # Extract
    local bin_dir="${INSTALL_PREFIX}/bin"
    mkdir -p "$bin_dir"

    log_verbose "Extracting to ${bin_dir}"
    if [ "$os" = "windows" ]; then
        # For zip, extract to temp dir first, then move binary
        local temp_dir=$(mktemp -d)
        if ! unzip -o -q "$temp_file" -d "$temp_dir"; then
            log_error "Failed to extract ${app}"
            rm -f "$temp_file"
            rm -rf "$temp_dir"
            return 1
        fi
        # Move the binary from temp dir to bin dir and rename
        mv "${temp_dir}/${app}"* "${bin_dir}/${app}"
        rm -rf "$temp_dir"
    else
        # Extract to temp dir first, then move and rename
        local temp_dir=$(mktemp -d)
        if ! $extract_cmd "$temp_file" -C "$temp_dir"; then
            log_error "Failed to extract ${app}"
            rm -f "$temp_file"
            rm -rf "$temp_dir"
            return 1
        fi
        # Move the binary from temp dir to bin dir and rename
        mv "${temp_dir}/${app}"* "${bin_dir}/${app}"
        rm -rf "$temp_dir"
    fi

    rm -f "$temp_file"

    # Make executable
    chmod +x "${bin_dir}/${app}"

    log_info "✓ ${app} installed"
    return 0
}

# ============================================================================
# PATH Configuration
# ============================================================================
configure_path() {
    local bin_dir="${INSTALL_PREFIX}/bin"

    # Check if already in PATH
    if echo ":$PATH:" | grep -q ":${bin_dir}:"; then
        log_info "Install directory already in PATH"
        return 0
    fi

    # Skip if requested
    if [ "$NO_PATH" = "1" ]; then
        log_info "Skipping PATH configuration"
        return 0
    fi

    # Try to add to shell profile
    local shell_profiles=(".bashrc" ".zshrc" ".profile" ".bash_profile")
    local profile_added=false

    for profile in "${shell_profiles[@]}"; do
        local profile_path="$HOME/$profile"
        if [ -f "$profile_path" ]; then
            if ! grep -q "${bin_dir}" "$profile_path"; then
                echo "" >> "$profile_path"
                echo "# Added by sshifu install script" >> "$profile_path"
                echo "export PATH=\"${bin_dir}:\$PATH\"" >> "$profile_path"
                log_info "Added to PATH in $profile"
                profile_added=true
                break
            fi
        fi
    done

    if [ "$profile_added" = "false" ]; then
        log_info "Add to PATH manually:"
        echo "  export PATH=\"${bin_dir}:\$PATH\""
    fi
}

# ============================================================================
# Verification
# ============================================================================
verify_installation() {
    local bin_dir="${INSTALL_PREFIX}/bin"
    local success=true

    for app in $(echo "$APPS_TO_INSTALL" | tr ',' ' '); do
        if [ -x "${bin_dir}/${app}" ]; then
            log_info "✓ ${app} verified"
        else
            log_error "✗ ${app} not found"
            success=false
        fi
    done

    if [ "$success" = "false" ]; then
        die "Installation incomplete"
    fi
}

# ============================================================================
# Help
# ============================================================================
show_help() {
    cat << EOF
${SCRIPT_NAME} - Install sshifu binaries

Usage:
  curl ... | bash [OPTIONS]
  curl ... | INSTALL_APP=sshifu bash

Options:
  --app APPS        Apps to install (comma-separated)
                    Valid: sshifu, sshifu-server, sshifu-trust
                    Default: all three apps
  --version VER     Version to install (default: latest)
  --prefix PATH     Install location (default: ~/.sshifu)
  --system          Install to /usr/local (requires sudo)
  --no-path         Skip PATH configuration
  --verbose         Show detailed output
  --help            Show this help message

Examples:
  # Install all apps
  curl ... | bash

  # Install only sshifu CLI
  curl ... | INSTALL_APP=sshifu bash
  curl ... | bash -s -- --app sshifu

  # Install specific version
  curl ... | INSTALL_VERSION=0.1.0 bash

  # Install to custom location
  curl ... | INSTALL_PREFIX=/opt/sshifu bash

  # Install all to system location
  curl ... | bash -s -- --system
EOF
}

# ============================================================================
# Main
# ============================================================================
main() {
    parse_args "$@"

    log_info "Starting installation..."

    # Detect system
    local os=$(detect_os)
    local arch=$(detect_arch)
    log_info "Detected: ${os}-${arch}"

    # Resolve version
    if [ "$VERSION" = "latest" ]; then
        VERSION=$(get_latest_version)
    fi
    log_info "Version: ${VERSION}"

    # Validate apps
    for app in $(echo "$APPS_TO_INSTALL" | tr ',' ' '); do
        if ! echo "$VALID_APPS" | grep -qw "$app"; then
            die "Invalid app: $app. Valid: ${VALID_APPS// /, }"
        fi
    done
    log_info "Apps: ${APPS_TO_INSTALL}"

    # Install each app
    local failed=0
    for app in $(echo "$APPS_TO_INSTALL" | tr ',' ' '); do
        if ! download_and_install_app "$app" "$os" "$arch" "$VERSION"; then
            failed=$((failed + 1))
        fi
    done

    if [ $failed -gt 0 ]; then
        die "$failed app(s) failed to install"
    fi

    # Configure PATH
    configure_path

    # Verify
    verify_installation

    log_info "Installation complete!"
    log_info "Run '${APPS_TO_INSTALL%%,*} --help' to get started"
}

main "$@"
