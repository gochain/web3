#!/bin/sh
set -e

# Install script to install web3
if [ "$1" != "" ]; then
    version="$1"
else
    version=`curl --silent https://api.github.com/repos/gochain/web3/releases/latest  | grep tag_name | cut -f 2 -d : | cut -f 2 -d '"'`
fi


command_exists() {
    command -v "$@" > /dev/null 2>&1
}

case "$(uname -m)" in
    *64)
    ;;
    *)
        echo >&2 'Error: you are not using a 64bit platform.'
        echo >&2 'Web3 CLI currently only supports 64bit platforms.'
        exit 1
    ;;
esac

user="$(id -un 2>/dev/null || true)"

sh_c='sh -c'
if [ "$user" != 'root' ]; then
    if command_exists sudo; then
        sh_c='sudo -E sh -c'
        elif command_exists su; then
        sh_c='su -c'
    else
        echo >&2 'Error: this installer needs the ability to run commands as root.'
        echo >&2 'We are unable to find either "sudo" or "su" available to make this happen.'
        exit 1
    fi
fi

curl=''
if command_exists curl; then
    curl='curl -sSL -o'
    elif command_exists wget; then
    curl='wget -qO'
    elif command_exists busybox && busybox --list-modules | grep -q wget; then
    curl='busybox wget -qO'
else
    echo >&2 'Error: this installer needs the ability to run wget or curl.'
    echo >&2 'We are unable to find either "wget" or "curl" available to make this happen.'
    exit 1
fi

url='https://github.com/gochain/web3/releases/download'


# perform some very rudimentary platform/architecture detection
case "$(uname)" in
    Linux)
        case "`uname -m`" in
            x86_64|amd64)
                echo "Installing web3 for linux x86_64"
                $sh_c "$curl /tmp/web3_linux $url/$version/web3_linux"
            ;;
            arm*|aarch*)
                $sh_c "$curl /tmp/web3_linux $url/$version/web3_linux_arm64"
                echo "Installing web3 for linux_arm64"
            ;;
            *)
            echo "Unknown architecture: $(uname -i)";;
        esac
        $sh_c "mv /tmp/web3_linux /usr/local/bin/web3"
        $sh_c "chmod +x /usr/local/bin/web3"
        web3
    ;;
    Darwin)
        case "`uname -m`" in
            x86_64|amd64)
                $sh_c "$curl /tmp/web3_mac $url/$version/web3_mac"
                echo "Installing web3 for mac x86_64"
            ;;
            arm*|aarch*)
                $sh_c "$curl /tmp/web3_linux $url/$version/web3_mac_arm64"
                echo "Installing web3 for mac arm64"
            ;;
            *)
            echo "Unknown architecture: $(uname -i)";;
        esac
        $sh_c "mv /tmp/web3_mac /usr/local/bin/web3"
        $sh_c "chmod +x /usr/local/bin/web3"
        web3
    ;;
    WindowsNT)
        $sh_c "$curl $url/$version/web3.exe"
        # TODO how to make executable? chmod? how to do tmp file and move?
        web3.exe
    ;;
    *)
    cat >&2 <<'EOF'

  Either your platform is not easily detectable or is not supported by this
  installer script (yet - PRs welcome!.
EOF
        exit 1
esac

exit 0
