#!/bin/sh
set -eu

command_exists() {
    command -v "$@" > /dev/null 2>&1
}

if [ "${1:-}" != "" ]; then
    version="$1"
else
    if command_exists curl; then
        version=$(curl -fsSL https://api.github.com/repos/gochain/web3/releases/latest | grep '"tag_name"' | head -n 1 | cut -d '"' -f 4)
    elif command_exists wget; then
        version=$(wget -qO- https://api.github.com/repos/gochain/web3/releases/latest | grep '"tag_name"' | head -n 1 | cut -d '"' -f 4)
    else
        echo >&2 'Error: curl or wget is required to install web3.'
        exit 1
    fi
fi

if [ -z "${version:-}" ]; then
    echo >&2 'Error: unable to determine latest web3 version.'
    exit 1
fi

os_name="$(uname -s)"
arch_name="$(uname -m)"

case "$arch_name" in
    x86_64|amd64)
        arch="amd64"
        ;;
    arm64|aarch64)
        arch="arm64"
        ;;
    *)
        echo >&2 "Error: unsupported architecture '$arch_name' (64-bit x86_64 or arm64 required)."
        exit 1
        ;;
esac

case "$os_name" in
    Linux)
        if [ "$arch" = "arm64" ]; then
            asset="web3_linux_arm64"
        else
            asset="web3_linux"
        fi
        ;;
    Darwin)
        if [ "$arch" = "arm64" ]; then
            asset="web3_mac_arm64"
        else
            asset="web3_mac"
        fi
        ;;
    MINGW*|MSYS*|CYGWIN*|Windows_NT)
        asset="web3.exe"
        ;;
    *)
        echo >&2 "Error: unsupported OS '$os_name'."
        exit 1
        ;;
esac

url="https://github.com/gochain/web3/releases/download/${version}/${asset}"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
tmpbin="${tmpdir}/${asset}"

echo "Downloading web3 ${version} (${os_name}/${arch})..."
if command_exists curl; then
    curl -fsSL -o "$tmpbin" "$url"
elif command_exists wget; then
    wget -qO "$tmpbin" "$url"
else
    echo >&2 'Error: curl or wget is required.'
    exit 1
fi

chmod +x "$tmpbin"

if [ "$asset" = "web3.exe" ]; then
    install_dir="${HOME}/bin"
    mkdir -p "$install_dir"
    mv "$tmpbin" "${install_dir}/web3.exe"
    echo "Installed to ${install_dir}/web3.exe"
    "${install_dir}/web3.exe" --version
    exit 0
fi

user="$(id -un 2>/dev/null || true)"
if [ "$user" = "root" ]; then
    mv "$tmpbin" /usr/local/bin/web3
elif command_exists sudo; then
    sudo mv "$tmpbin" /usr/local/bin/web3
elif command_exists su; then
    su -c "mv '$tmpbin' /usr/local/bin/web3"
else
    echo >&2 'Error: root/sudo privileges are required to install to /usr/local/bin/web3.'
    exit 1
fi

web3 --version
