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
#docker create -v /data --name file alpine /bin/true
#docker cp $version_file file:/data/version.go
# Bump version, patch by default - also checks if previous commit message contains `[bump X]`, and if so, bumps the appropriate semver number - https://github.com/treeder/dockers/tree/master/bump
lastcommitmsg=$(git log -1 --pretty=%B)
version=$(docker run --rm -i -w /bump -v $PWD:/bump treeder/bump --filename $version_file $lastcommitmsg)
# docker cp file:/data/version.go $version_file
#version=$(grep -m1 -Eo "[0-9]+\.[0-9]+\.[0-9]+" $version_file)
echo "Version: $version"

make release

tag="v$version"
git add -u
git commit -m "web3 CLI: $version release [skip ci]"

git tag -f -a $tag -m "version $version"
git push --follow-tags

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
