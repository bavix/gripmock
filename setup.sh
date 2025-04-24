#!/bin/sh

set -e

# Check if terminal supports colors using tput
if tput colors >/dev/null 2>&1 && [ "$(tput colors)" -ge 8 ]; then
    # Enable colors if supported
    GREEN=$(tput setaf 2)   # Green for success
    RED=$(tput setaf 1)     # Red for errors
    YELLOW=$(tput setaf 3)  # Yellow for information
    BLUE=$(tput setaf 4)    # Blue for accents
    BOLD=$(tput bold)       # Bold text
    NC=$(tput sgr0)         # Reset color
else
    # Disable colors if not supported
    GREEN=""
    RED=""
    YELLOW=""
    BLUE=""
    BOLD=""
    NC=""
fi

# Symbols
CHECK="✔"
CROSS="✘"
INFO="ℹ"

# Logging functions
log_info() {
    printf "${YELLOW}${INFO} ${NC}%s\n" "$1"
}

log_success() {
    printf "${GREEN}${CHECK} ${NC}%s\n" "$1"
}

log_error() {
    printf "${RED}${CROSS} ${NC}%s\n" "$1" >&2
    exit 1
}

spinner() {
    pid=$1
    message="$2"
    spin='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'
    while kill -0 $pid 2>/dev/null; do
        for i in 0 1 2 3 4 5 6 7 8 9; do
            char=$(echo "$spin" | cut -c$((i + 1)))
            printf "\r%s %s..." "$char" "$message"
            sleep 0.1
        done
    done
    # Clear the spinner line before printing success message
    printf "\r${GREEN}${CHECK} ${NC}%s... Done\n" "$message"
}

check_dependencies() {
    log_info "Checking dependencies..."
    for cmd in curl tar grep awk sha256sum uname stat; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            log_error "Command '${BLUE}$cmd${NC}' is missing. Please install it and try again."
        fi
    done
    log_success "Dependencies are ready."
}

detect_os_and_architecture() {
    log_info "Detecting system information..."
    OS=$(uname -s)
    ARCH=$(uname -m)

    case "$OS" in
        Linux) OS="linux" ;;
        Darwin) OS="darwin" ;;
        *) log_error "Unsupported OS: ${BLUE}$OS${NC}" ;;
    esac

    case "$ARCH" in
        x86_64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *) log_error "Unsupported architecture: ${BLUE}$ARCH${NC}" ;;
    esac

    log_success "Detected OS: ${BLUE}$OS 🌍${NC}"
    log_success "Detected architecture: ${BLUE}$ARCH 💻${NC}"
}

get_latest_version() {
    log_info "Fetching the latest version of GripMock from GitHub..."
    LATEST_RELEASE=$(curl --retry 12 --retry-all-errors -s https://api.github.com/repos/bavix/gripmock/releases/latest)
    LATEST_VERSION=$(echo "$LATEST_RELEASE" | grep '"tag_name":' | awk -F '"' '{print $4}')
    if [ -z "$LATEST_VERSION" ]; then
        log_error "Failed to fetch the latest version of GripMock from GitHub."
    fi
    # Remove the 'v' prefix from the version tag
    LATEST_VERSION=${LATEST_VERSION#v}
    log_success "Latest version: ${BLUE}$LATEST_VERSION 🎉${NC}"
}

download_checksums() {
    CHECKSUM_URL="https://github.com/bavix/gripmock/releases/download/v${LATEST_VERSION}/checksums.txt"

    TMP_DIR=$(mktemp -d)
    CHECKSUM_FILE="$TMP_DIR/checksums.txt"

    log_info "Downloading checksums file..."
    (
        curl --retry 12 --retry-all-errors -sL "$CHECKSUM_URL" -o "$CHECKSUM_FILE" &
        spinner $! "Downloading checksums"
    ) || log_error "Failed to download checksums file."

    log_success "Checksums file downloaded."
}

download_gripmock() {
    DOWNLOAD_URL="https://github.com/bavix/gripmock/releases/download/v${LATEST_VERSION}/gripmock_${LATEST_VERSION}_${OS}_${ARCH}.tar.gz"

    DOWNLOAD_FILE="$TMP_DIR/gripmock.tar.gz"

    log_info "Downloading GripMock for ${BLUE}${OS}/${ARCH}${NC}..."
    (
        curl --retry 12 --retry-all-errors -sL "$DOWNLOAD_URL" -o "$DOWNLOAD_FILE" &
        spinner $! "Downloading GripMock"
    ) || log_error "Download failed. Try again later."

    # Get file size with two decimal places using stat
    FILE_SIZE_BYTES=$(stat --version >/dev/null 2>&1 && stat -c%s "$DOWNLOAD_FILE" || stat -f%z "$DOWNLOAD_FILE")
    FILE_SIZE_MB=$(awk "BEGIN {printf \"%.2f\", ${FILE_SIZE_BYTES} / (1024 * 1024)}")
    log_success "Downloaded GripMock (${BLUE}${FILE_SIZE_MB} MB${NC})"
}

verify_checksum() {
    EXPECTED_CHECKSUM=$(grep "gripmock_${LATEST_VERSION}_${OS}_${ARCH}.tar.gz" "$CHECKSUM_FILE" | awk '{print $1}')
    if [ -z "$EXPECTED_CHECKSUM" ]; then
        log_error "Checksum not found for GripMock_${BLUE}${LATEST_VERSION}_${OS}_${ARCH}${NC}.tar.gz."
    fi

    ACTUAL_CHECKSUM=$(sha256sum "$DOWNLOAD_FILE" | awk '{print $1}')
    if [ "$ACTUAL_CHECKSUM" != "$EXPECTED_CHECKSUM" ]; then
        log_error "Checksum mismatch for GripMock! Expected: ${BLUE}$EXPECTED_CHECKSUM${NC}, Got: ${BLUE}$ACTUAL_CHECKSUM${NC}. File corrupted?"
    fi

    log_success "Checksum verified successfully."
}

install_gripmock() {
    log_info "Extracting GripMock..."
    tar -xzf "$TMP_DIR/gripmock.tar.gz" -C "$TMP_DIR" || log_error "Failed to extract GripMock."

    log_info "Installing GripMock..."
    if [ -w "/usr/local/bin" ] && [ -x "/usr/local/bin" ]; then
        cp "$TMP_DIR/gripmock" /usr/local/bin/gripmock || log_error "Failed to copy GripMock to /usr/local/bin."
    else
        sudo cp "$TMP_DIR/gripmock" /usr/local/bin/gripmock || log_error "Failed to copy GripMock. Try running the script with sudo!"
    fi

    log_success "GripMock has been successfully installed."
    log_info "You can now run '${BOLD}${BLUE}gripmock --help${NC}' to get started."
}

cleanup() {
    if [ -d "$TMP_DIR" ]; then
        rm -rf "$TMP_DIR"
    fi
}

if [ -f "/usr/local/bin/gripmock" ]; then
    log_info "GripMock is already installed. Starting update... 🚀"
else
    log_info "Starting GripMock installation... 🚀"
fi

check_dependencies
detect_os_and_architecture
get_latest_version
download_checksums
download_gripmock
verify_checksum
install_gripmock
cleanup
log_success "Installation complete! You're all set to use ${BLUE}GripMock 🎉${NC}"
