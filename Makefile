UNAME_S := $(shell uname -s)
build: 
	go build ./cmd/web3

install: build
	sudo cp web3 /usr/local/bin/web3

docker: 
	docker build -t gochain/web3:latest .

push: docker
	# todo: version these, or auto push this using CI
	docker push gochain/web3:latest

test:
	go test ./...

release:
ifeq ($(UNAME_S),Linux)
	GOOS=linux go build -o web3_linux ./cmd/web3 
	docker create -v /data --name web3_sources alpine /bin/true
	docker cp -a . web3_sources:/data/
	docker run --rm --volumes-from web3_sources -w /data treeder/go-dev go build -o web3_alpine ./cmd/web3 
	docker cp web3_sources:/data/web3_alpine web3_alpine
	docker rm -f web3_sources
endif
ifeq ($(UNAME_S),Darwin)
	go build -o web3_mac ./cmd/web3
endif
.PHONY: install test build docker release
