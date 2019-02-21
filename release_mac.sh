#!/bin/bash
set -ex

version=$(grep -m1 -Eo "[0-9]+\.[0-9]+\.[0-9]+" cmd/web3/version.go)
echo "Version: $version"

make release

# Upload to github
url='https://api.github.com/repos/gochain-io/web3/releases'
output=$(curl -s "$url/tags/v$version")
upload_url=$(echo "$output" | python -c 'import json,sys;obj=json.load(sys.stdin);print obj["upload_url"]' | sed -E "s/\{.*//")
html_url=$(echo "$output" | python -c 'import json,sys;obj=json.load(sys.stdin);print obj["html_url"]')
curl --data-binary "@web3_mac"  -H "Content-Type: application/octet-stream" -u $GH_DEPLOY_USER:$GH_DEPLOY_KEY $upload_url\?name\=web3_mac >/dev/null