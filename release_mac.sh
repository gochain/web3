#!/bin/bash
set -ex

version_file="cmd/web3/version.go"
go get github.com/treeder/dockers/bump
$HOME/go/bin/bump --filename $version_file "$(git log -1 --pretty=%B)"
version=$(grep -m1 -Eo "[0-9]+\.[0-9]+\.[0-9]+" $version_file)
echo "Version: $version"

make release

# Upload to github
url='https://api.github.com/repos/gochain/web3/releases'
output=$(curl -s "$url/tags/v$version")
upload_url=$(echo "$output" | python -c 'import json,sys;obj=json.load(sys.stdin);print obj["upload_url"]' | sed -E "s/\{.*//")
html_url=$(echo "$output" | python -c 'import json,sys;obj=json.load(sys.stdin);print obj["html_url"]')
curl --data-binary "@web3_mac"  -H "Content-Type: application/octet-stream" -u $GH_DEPLOY_USER:$GH_DEPLOY_KEY $upload_url\?name\=web3_mac >/dev/null