#!/bin/sh
set -e

RESET="\\033[0m"
RED="\\033[31;1m"
GREEN="\\033[32;1m"
YELLOW="\\033[33;1m"
BLUE="\\033[34;1m"
WHITE="\\033[37;1m"

print_unsupported_platform()
{
    >&2 say_red "error: We're sorry, but it looks like pulumictl is not supported on your platform"
    >&2 say_red "       We support 64-bit versions of Linux and macOS and are interested in supporting"
    >&2 say_red "       more platforms.  Please open an issue at https://github.com/pulumi/pulumictl/issues/new and"
    >&2 say_red "       let us know what platform you're using!"
}

say_green()
{
    [ -z "${SILENT}" ] && printf "%b%s%b\\n" "${GREEN}" "$1" "${RESET}"
    return 0
}

say_red()
{
    printf "%b%s%b\\n" "${RED}" "$1" "${RESET}"
}

say_yellow()
{
    [ -z "${SILENT}" ] && printf "%b%s%b\\n" "${YELLOW}" "$1" "${RESET}"
    return 0
}

say_blue()
{
    [ -z "${SILENT}" ] && printf "%b%s%b\\n" "${BLUE}" "$1" "${RESET}"
    return 0
}

say_white()
{
    [ -z "${SILENT}" ] && printf "%b%s%b\\n" "${WHITE}" "$1" "${RESET}"
    return 0
}

at_exit()
{
    # shellcheck disable=SC2181
    # https://github.com/koalaman/shellcheck/wiki/SC2181
    # Disable because we don't actually know the command we're running
    if [ "$?" -ne 0 ]; then
        >&2 say_red
        >&2 say_red "We're sorry, but it looks like something might have gone wrong during installation."
        >&2 say_red "If you need help, please open an issue at https://github.com/pulumi/pulumictl/issues/new"
    fi
}

trap at_exit EXIT

VERSION=""
INSTALL_ROOT=""
NO_EDIT_PATH=""
SILENT=""
while [ $# -gt 0 ]; do
    case "$1" in
        --version)
            if [ "$2" != "latest" ]; then
                VERSION=$2
            fi
            ;;
        --silent)
            SILENT="--silent"
            ;;
        --install-root)
            INSTALL_ROOT=$2
            ;;
        --no-edit-path)
            NO_EDIT_PATH="true"
            ;;
     esac
     shift
done

if [ -z "${VERSION}" ]; then
    # Query GitHub API for the most recent release
    if [ -n "$GITHUB_TOKEN" ]; then
        RESPONSE=$(curl --retry 3 --fail --silent -L \
            --header "Authorization: Bearer $GITHUB_TOKEN" \
            "https://api.github.com/repos/pulumi/pulumictl/releases/latest")
    else
        RESPONSE=$(curl --retry 3 --fail --silent -L \
            "https://api.github.com/repos/pulumi/pulumictl/releases/latest")
    fi
    
    if [ -z "$RESPONSE" ]; then
        >&2 say_red "error: could not determine latest version of pulumictl, try passing --version X.Y.Z to"
        >&2 say_red "       install an explicit version"
        exit 1
    fi
    
    VERSION=$(echo "$RESPONSE" | grep '"tag_name":' | sed -E 's/.*"tag_name": "v?([^"]+)".*/\1/')
    
    if [ -z "$VERSION" ]; then
        >&2 say_red "error: could not parse version from GitHub API response, try passing --version X.Y.Z to"
        >&2 say_red "       install an explicit version"
        exit 1
    fi
fi

OS=""
case $(uname) in
    "Linux") OS="linux";;
    "Darwin") OS="darwin";;
    *)
        print_unsupported_platform
        exit 1
        ;;
esac

ARCH=""
case $(uname -m) in
    "x86_64") ARCH="amd64";;
    "arm64") ARCH="arm64";;
    "aarch64") ARCH="arm64";;
    *)
        print_unsupported_platform
        exit 1
        ;;
esac

TARBALL_URL="https://github.com/pulumi/pulumictl/releases/download/v${VERSION}/"
TARBALL_PATH=pulumictl-v${VERSION}-${OS}-${ARCH}.tar.gz
PULUMICTL_INSTALL_ROOT=${INSTALL_ROOT}

if [ "$PULUMICTL_INSTALL_ROOT" = "" ]; then
    # Default to ~/.pulumi
    PULUMICTL_INSTALL_ROOT="${HOME}/.pulumi"
fi
PULUMICTL_CLI="${PULUMICTL_INSTALL_ROOT}/bin/pulumictl"

if [ -d "${PULUMICTL_CLI}" ]; then
    say_red "error: ${PULUMICTL_CLI} already exists and is a directory, refusing to proceed."
    exit 1
elif [ ! -f "${PULUMICTL_CLI}" ]; then
    say_blue "=== Installing pulumictl v${VERSION} ==="
else
    say_blue "=== Upgrading pulumictl $(${PULUMICTL_CLI} version) to v${VERSION} ==="
fi

TARBALL_DEST=$(mktemp -t pulumictl.tar.gz.XXXXXXXXXX)

download_tarball() {
    say_white "+ Downloading ${TARBALL_URL}${TARBALL_PATH}..."
    # This should opportunistically use the GITHUB_TOKEN to avoid rate limiting
    if [ -n "$GITHUB_TOKEN" ]; then
        curl --retry 2 --fail ${SILENT} -L \
            --header "Authorization: Bearer $GITHUB_TOKEN" \
            -o "${TARBALL_DEST}" "${TARBALL_URL}${TARBALL_PATH}"
    else
        curl --retry 2 --fail ${SILENT} -L \
            -o "${TARBALL_DEST}" "${TARBALL_URL}${TARBALL_PATH}"
    fi
}

if download_tarball; then
    say_white "+ Extracting to ${PULUMICTL_INSTALL_ROOT}/bin"

    # If `~/.pulumi/bin/pulumictl` exists, remove it
    if [ -e "${PULUMICTL_INSTALL_ROOT}/bin/pulumictl" ]; then
        rm -f "${PULUMICTL_INSTALL_ROOT}/bin/pulumictl"
    fi

    mkdir -p "${PULUMICTL_INSTALL_ROOT}/bin"

    # Yarn's shell installer does a similar dance of extracting to a temp
    # folder and copying to not depend on additional tar flags
    EXTRACT_DIR=$(mktemp -dt pulumictl.XXXXXXXXXX)
    tar zxf "${TARBALL_DEST}" -C "${EXTRACT_DIR}"

    # Copy the binary to the install location
    cp "${EXTRACT_DIR}/pulumictl" "${PULUMICTL_INSTALL_ROOT}/bin/"

    rm -f "${TARBALL_DEST}"
    rm -rf "${EXTRACT_DIR}"
else
    >&2 say_red "error: failed to download ${TARBALL_URL}${TARBALL_PATH}"
    >&2 say_red "       check your internet and try again; if the problem persists, file an"
    >&2 say_red "       issue at https://github.com/pulumi/pulumictl/issues/new"
    exit 1
fi

# Now that we have installed pulumictl, if it is not already on the path, let's add a line to the
# user's profile to add the folder to the PATH for future sessions.
if [ "${NO_EDIT_PATH}" != "true" ] && ! command -v pulumictl >/dev/null; then
    # If we can, we'll add a line to the user's .profile adding ${PULUMICTL_INSTALL_ROOT}/bin to the PATH
    SHELL_NAME=$(basename "${SHELL}")
    PROFILE_FILE=""

    case "${SHELL_NAME}" in
        "bash")
            # Terminal.app on macOS prefers .bash_profile to .bashrc, so we prefer that
            # file when trying to put our export into a profile. On *NIX, .bashrc is
            # preferred as it is sourced for new interactive shells.
            if [ "$(uname)" != "Darwin" ]; then
                if [ -e "${HOME}/.bashrc" ]; then
                    PROFILE_FILE="${HOME}/.bashrc"
                elif [ -e "${HOME}/.bash_profile" ]; then
                    PROFILE_FILE="${HOME}/.bash_profile"
                fi
            else
                if [ -e "${HOME}/.bash_profile" ]; then
                    PROFILE_FILE="${HOME}/.bash_profile"
                elif [ -e "${HOME}/.bashrc" ]; then
                    PROFILE_FILE="${HOME}/.bashrc"
                fi
            fi
            ;;
        "zsh")
            if [ -e "${ZDOTDIR:-$HOME}/.zshrc" ]; then
                PROFILE_FILE="${ZDOTDIR:-$HOME}/.zshrc"
            fi
            ;;
    esac

    if [ -n "${PROFILE_FILE}" ]; then
        LINE_TO_ADD="export PATH=\$PATH:${PULUMICTL_INSTALL_ROOT}/bin"
        if ! grep -q "# add pulumictl to the PATH" "${PROFILE_FILE}"; then
            say_white "+ Adding ${PULUMICTL_INSTALL_ROOT}/bin to \$PATH in ${PROFILE_FILE}"
            printf "\\n# add pulumictl to the PATH\\n%s\\n" "${LINE_TO_ADD}" >> "${PROFILE_FILE}"
        fi

        EXTRA_INSTALL_STEP="+ Please restart your shell or add ${PULUMICTL_INSTALL_ROOT}/bin to your \$PATH"
    else
        EXTRA_INSTALL_STEP="+ Please add ${PULUMICTL_INSTALL_ROOT}/bin to your \$PATH"
    fi
fi

# Warn if the pulumictl command is available, but it is in a different location from the current install root.
if [ "$(command -v pulumictl)" != "" ] && [ "$(command -v pulumictl)" != "${PULUMICTL_INSTALL_ROOT}/bin/pulumictl" ]; then
    say_yellow
    say_yellow "warning: pulumictl has been installed to ${PULUMICTL_INSTALL_ROOT}/bin, but it looks like there's a different copy"
    say_yellow "         on your \$PATH at $(dirname "$(command -v pulumictl)"). You'll need to explicitly invoke the"
    say_yellow "         version you just installed or modify your \$PATH to prefer this location."
fi

say_blue
say_blue "=== pulumictl is now installed! 🍹 ==="
if [ "$EXTRA_INSTALL_STEP" != "" ]; then
    say_white "${EXTRA_INSTALL_STEP}"
fi
say_green "+ Get started: https://github.com/pulumi/pulumictl"
