#!/bin/bash
set -ex

# ensure working dir is clean
# git status
# if [[ -z $(git status -s) ]]
# then
#   echo "tree is clean"
# else
#   echo "tree is dirty, please commit changes before running this"
#   exit 1
# fi

# git pull

version_file="cmd/web3/version.go"
lastcommitmsg=$(git log -1 --pretty=%B)
version=$(docker run --rm -i -w /bump -v $PWD:/bump treeder/bump --filename $version_file $lastcommitmsg)
echo "Version: $version"

make release

tag="v$version"
git add -u
git commit -m "web3 CLI: $version release [skip ci]"
git tag -f -a $tag -m "version $version"
git push --follow-tags --set-upstream origin master

# For GitHub
url='https://api.github.com/repos/gochain/web3/releases'
output=$(curl -s -u $GH_DEPLOY_USER:$GH_DEPLOY_KEY -d "{\"tag_name\": \"v$version\", \"name\": \"v$version\"}" $url)
upload_url=$(echo "$output" | python -c 'import json,sys;obj=json.load(sys.stdin);print obj["upload_url"]' | sed -E "s/\{.*//")
html_url=$(echo "$output" | python -c 'import json,sys;obj=json.load(sys.stdin);print obj["html_url"]')
curl --data-binary "@web3_linux"  -H "Content-Type: application/octet-stream" -u $GH_DEPLOY_USER:$GH_DEPLOY_KEY $upload_url\?name\=web3_linux >/dev/null
curl --data-binary "@web3_alpine" -H "Content-Type: application/octet-stream" -u $GH_DEPLOY_USER:$GH_DEPLOY_KEY $upload_url\?name\=web3_alpine >/dev/null

docker build -t gochain/web3:latest .
docker tag gochain/web3:latest gochain/web3:$version
docker push gochain/web3:$version
docker push gochain/web3:latest
