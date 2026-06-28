#!/bin/sh
set -eu

# Potaco installer - downloads and installs the potaco CLI
# Interactive by default. POTACO_NON_INTERACTIVE=1 is for agents and automated
# terminal execution, not a polished scripting API.

# ============================================================================
# Constants
# ============================================================================
REPO="ncxton/potaco"
GITHUB_API="https://api.github.com/repos/${REPO}/releases/latest"
GITHUB_BASE="https://github.com/${REPO}"

# Color codes (stripped if NO_COLOR or TERM=dumb)
if [ "${NO_COLOR:-}" ] || [ "${TERM:-}" = "dumb" ] || [ ! -t 1 ]; then
    CYAN=""
    GREEN=""
    YELLOW=""
    RED=""
    RESET=""
else
    CYAN="\033[36m"
    GREEN="\033[32m"
    YELLOW="\033[33m"
    RED="\033[31m"
    RESET="\033[0m"
fi

NON_INTERACTIVE="${POTACO_NON_INTERACTIVE:-0}"

# ============================================================================
# Output helpers
# ============================================================================

info() {
    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf '%s\n' "$1"
    else
        printf "${CYAN}%s${RESET}\n" "$1"
    fi
}

warn() {
    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf '%s\n' "$1" >&2
    else
        printf "${YELLOW}%s${RESET}\n" "$1" >&2
    fi
}

error() {
    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf '%s\n' "$1" >&2
    else
        printf "${RED}%s${RESET}\n" "$1" >&2
    fi
}

success() {
    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf '%s\n' "$1"
    else
        printf "${GREEN}%s${RESET}\n" "$1"
    fi
}

confirm() {
    prompt="$1"
    default="${2:-y}"

    if [ "$NON_INTERACTIVE" = "1" ]; then
        return 0
    fi

    if [ "$default" = "y" ]; then
        suffix="[Y/n]"
    else
        suffix="[y/N]"
    fi

    if [ -r /dev/tty ]; then
        printf "%s %s " "$prompt" "$suffix" > /dev/tty
        answer=""
        read answer < /dev/tty || answer=""
    else
        warn "No TTY available for confirmation."
        return 1
    fi

    case "$answer" in
        [Yy]*)
            return 0
            ;;
        [Nn]*)
            return 1
            ;;
        "")
            [ "$default" = "y" ]
            ;;
        *)
            return 1
            ;;
    esac
}

# Spinner
# ============================================================================

SPINNER_PID=""
SPINNER_RUNNING=0
SPINNER_CHARS="|/-\\"

spinner_start() {
    if [ "$NON_INTERACTIVE" = "1" ] || [ "${TERM:-}" = "dumb" ] || [ ! -t 2 ]; then
        return
    fi
    msg="$1"
    (
        i=0
        while true; do
            char=$(printf '%s' "$SPINNER_CHARS" | cut -c$((i % 4 + 1)))
            printf "\r${CYAN}[%s]${RESET} %s   " "$char" "$msg" >&2
            # sleep 0.1 is non-POSIX (fractional), but both supported platforms
            # (macOS/Linux) support it, and the spinner is guarded by ! -t 2
            # so it only runs in interactive terminal mode, never in CI/pipes.
            sleep 0.1
            i=$((i + 1))
        done
    ) 2>/dev/null &
    SPINNER_PID=$!
    SPINNER_RUNNING=1
}

spinner_stop() {
    if [ "$SPINNER_RUNNING" = "1" ] && [ -n "$SPINNER_PID" ]; then
        kill "$SPINNER_PID" 2>/dev/null || true
        wait "$SPINNER_PID" 2>/dev/null || true
        printf "\r\033[K" >&2
        SPINNER_RUNNING=0
    fi
}

# ============================================================================
# Platform detection
# ============================================================================

detect_platform() {
    os=$(uname -s)
    arch=$(uname -m)

    case "$os" in
        Darwin) os="apple-darwin" ;;
        Linux)  os="linux" ;;
        *)
            error "Unsupported operating system: $os"
            error "Potaco supports macOS and Linux only."
            exit 1
            ;;
    esac

    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *)
            error "Unsupported architecture: $arch"
            error "Potaco supports amd64 (x86_64) and arm64 (aarch64) only."
            exit 1
            ;;
    esac

    printf '%s/%s' "$os" "$arch"
}

# ============================================================================
# Version detection
# ============================================================================

detect_version() {
    # Try to get the latest release tag from GitHub API
    if command -v curl >/dev/null 2>&1; then
        response=$(curl -fsSL "$GITHUB_API" 2>/dev/null || true)
    elif command -v wget >/dev/null 2>&1; then
        response=$(wget -qO- "$GITHUB_API" 2>/dev/null || true)
    else
        response=""
    fi

    if [ -n "$response" ]; then
        # Parse tag_name from JSON without jq
        tag=$(printf '%s' "$response" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//')
        if [ -n "$tag" ]; then
            printf '%s' "$tag"
            return
        fi
    fi

    # Fallback: empty string means use the "latest" redirect URLs
    printf ''
}

# ============================================================================
# Download helper
# ============================================================================

# Download a URL to a file with retry
# Usage: download_file "url" "output_path"
download_file() {
    url="$1"
    output="$2"
    attempts=0
    max_attempts=3

    while [ "$attempts" -lt "$max_attempts" ]; do
        attempts=$((attempts + 1))
        if command -v curl >/dev/null 2>&1; then
            if curl -fsSL -o "$output" "$url" 2>/dev/null; then
                return 0
            fi
        elif command -v wget >/dev/null 2>&1; then
            if wget -q -O "$output" "$url" 2>/dev/null; then
                return 0
            fi
        else
            error "Neither curl nor wget is available."
            error "Install one of them to use the potaco installer."
            exit 1
        fi

        if [ "$attempts" -lt "$max_attempts" ]; then
            warn "Download failed (attempt $attempts/$max_attempts), retrying..."
            sleep 2
        fi
    done

    error "Failed to download after $max_attempts attempts."
    error "URL: $url"
    return 1
}

# ============================================================================
# Checksum verification
# ============================================================================

# Verify the downloaded tarball against the checksums file
# Usage: verify_checksum "tarball_path" "checksums_path" "tarball_filename"
verify_checksum() {
    tarball="$1"
    checksums="$2"
    tarball_name="$3"

    if [ ! -f "$checksums" ]; then
        error "Checksums file not found; refusing to install without verification."
        return 1
    fi

    if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum "$tarball" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
        actual=$(shasum -a 256 "$tarball" | awk '{print $1}')
    else
        error "Neither sha256sum nor shasum is available; refusing to install without verification."
        return 1
    fi

    # Find the matching line in the checksums file
    expected=$(grep -F "$tarball_name" "$checksums" | awk '{print $1}' | head -1)

    if [ -z "$expected" ]; then
        error "Checksum for $tarball_name not found; refusing to install."
        return 1
    fi

    if [ "$actual" != "$expected" ]; then
        error "Checksum verification failed!"
        error "Expected: $expected"
        error "Actual:   $actual"
        return 1
    fi

    return 0
}

# ============================================================================
# Shell config helper
# ============================================================================

# add_to_shell_config appends a PATH export to the user's shell config file,
# auto-detected from $SHELL. Supports bash, zsh, and fish.
# Usage: add_to_shell_config "/path/to/bin"
add_to_shell_config() {
    bin_dir="$1"
    shell_path="${SHELL:-}"
    config_file=""
    export_line=""

    case "$shell_path" in
        */bash)
            config_file="${HOME}/.bashrc"
            export_line="export PATH=\"${bin_dir}:\$PATH\""
            ;;
        */zsh)
            config_file="${HOME}/.zshrc"
            export_line="export PATH=\"${bin_dir}:\$PATH\""
            ;;
        */fish)
            config_file="${HOME}/.config/fish/config.fish"
            export_line="fish_add_path ${bin_dir}"
            ;;
        *)
            warn "Could not detect shell from \$SHELL ($shell_path)."
            warn "Add ${bin_dir} to your PATH manually."
            return 0
            ;;
    esac

    # Create the config file if it doesn't exist (e.g. fish config)
    config_dir=$(dirname "$config_file")
    mkdir -p "$config_dir" 2>/dev/null || true

    # Check if the export line is already present
    if grep -qF "$bin_dir" "$config_file" 2>/dev/null; then
        info "$bin_dir already in $config_file."
        return 0
    fi

    printf '\n# Added by potaco installer\n%s\n' "$export_line" >> "$config_file"
    success "Added $bin_dir to $config_file"
    info "Restart your shell or run: source $config_file"
}

# ============================================================================
# Main installation flow
# ============================================================================

main() {
    # Check for required tools
    if ! command -v tar >/dev/null 2>&1; then
        error "tar is required but not found in PATH."
        error "Install tar to use the potaco installer."
        exit 1
    fi

    # Detect platform
    platform=$(detect_platform)
    os=$(printf '%s' "$platform" | cut -d/ -f1)
    arch=$(printf '%s' "$platform" | cut -d/ -f2)

    # Detect version
    version=$(detect_version)
    asset_version=""
    if [ -n "$version" ]; then
        asset_version=$(printf '%s' "$version" | sed 's/^v//')
    fi

    # Determine download URLs
    if [ -z "$version" ]; then
        error "Could not determine the latest potaco release version from GitHub."
        error "Check your network connection or download a release archive manually:"
        error "${GITHUB_BASE}/releases/latest"
        exit 1
    fi
    tarball_name="potaco_${asset_version}_${os}_${arch}.tar.gz"
    tarball_url="${GITHUB_BASE}/releases/download/${version}/${tarball_name}"
    checksums_name="potaco_${asset_version}_checksums.txt"
    checksums_url="${GITHUB_BASE}/releases/download/${version}/${checksums_name}"
    version_display="$version"
    install_dir="${HOME}/.local/bin"
    install_path="${install_dir}/potaco"
    add_path_after_install="0"

    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf 'Installing potaco %s for %s to %s\n' "$version_display" "$platform" "$install_path"
    else
        printf 'potaco %s\n' "$version_display"
        printf 'platform: %s\n' "$platform"
        printf 'install:  %s\n' "$install_path"
        printf 'source:   %s\n' "$GITHUB_BASE"
        printf "\n"
        if ! confirm "Install potaco ${version_display}?" "y"; then
            warn "Installation cancelled."
            exit 0
        fi
        printf "\n"
    fi

    # Create temp directory
    tmpdir=$(mktemp -d 2>/dev/null || mktemp -d -t potaco-install)
    tarball_path="${tmpdir}/${tarball_name}"
    checksums_path="${tmpdir}/${checksums_name}"
    cleanup() {
        rm -rf "$tmpdir" 2>/dev/null || true
    }
    trap cleanup EXIT INT TERM

    # Download tarball
    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf 'Downloading...\n'
    else
        spinner_start "Downloading..."
    fi

    if ! download_file "$tarball_url" "$tarball_path"; then
        spinner_stop
        exit 1
    fi

    spinner_stop

    # Download checksums
    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf 'Downloading checksums...\n'
    else
        spinner_start "Downloading checksums..."
    fi

    if ! download_file "$checksums_url" "$checksums_path"; then
        spinner_stop
        error "Could not download checksums. Aborting installation."
        exit 1
    fi
    spinner_stop

    # Verify checksum
    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf 'Verifying checksum...\n'
    else
        spinner_start "Verifying checksum..."
    fi

    if ! verify_checksum "$tarball_path" "$checksums_path" "$tarball_name"; then
        spinner_stop
        error "Checksum verification failed. Aborting installation."
        exit 1
    fi
    spinner_stop

    # After the download is complete and verified, decide whether to update PATH.
    case ":${PATH}:" in
        *":${install_dir}:"*)
            ;;
        *)
            if [ "$NON_INTERACTIVE" = "1" ]; then
                add_path_after_install="1"
            else
                printf "\n"
                if confirm "$install_dir is not in your PATH. Add it automatically after install?" "y"; then
                    add_path_after_install="1"
                else
                    add_path_after_install="0"
                fi
            fi
            ;;
    esac

    # Extract
    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf 'Extracting...\n'
    else
        spinner_start "Extracting..."
    fi

    tar -xzf "$tarball_path" -C "$tmpdir"
    binary_path="${tmpdir}/potaco"
    spinner_stop

    if [ ! -f "$binary_path" ]; then
        error "Binary not found in archive after extraction."
        error "Expected: $binary_path"
        exit 1
    fi

    # Install to ~/.local/bin (always, no sudo needed)
    mkdir -p "$install_dir" 2>/dev/null || true

    # Install the binary
    if [ "$NON_INTERACTIVE" = "1" ]; then
        printf 'Installing to %s...\n' "$install_path"
    else
        spinner_start "Installing..."
    fi

    if [ -w "$install_dir" ]; then
        mv "$binary_path" "$install_path"
        chmod +x "$install_path"
    else
        spinner_stop
        error "Cannot write to $install_dir."
        error "Ensure ~/.local/bin exists and is writable."
        exit 1
    fi

    spinner_stop

    if [ "$add_path_after_install" = "1" ]; then
        add_to_shell_config "$install_dir"
    elif [ "$add_path_after_install" = "0" ]; then
        case ":${PATH}:" in
            *":${install_dir}:"*) ;;
            *) warn "Add it manually: export PATH=\"${install_dir}:\$PATH\"" ;;
        esac
    fi

    # Print success
    printf "\n"
    success "Done. Potaco installed to $install_path"
    printf 'Next: potaco auth add openai\n'
    printf '      potaco gen --prompt "hello"\n'
    exit 0
}

main "$@"
