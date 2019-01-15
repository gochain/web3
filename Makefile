build: 
	go build -o web3

install:
	go build -o ${GOPATH}/bin/web3

docker: 
	docker build -t gochain/web3:latest .

test: build
	./test.sh

release:
	GOOS=linux go build -o web3_linux
	GOOS=darwin go build -o web3_mac
	GOOS=windows go build -o web3.exe
	# Uses fnproject/go:x.x-dev because golang:alpine has this issue: https://github.com/docker-library/golang/issues/155 and this https://github.com/docker-library/golang/issues/153
	docker run --rm -v ${PWD}:/dev/web3 -w /dev/web3 treeder/go-dev go build -o web3_alpine

.PHONY: install test build docker release
