build:
	CGO_ENABLED=0 go build -trimpath ./cmd/web3

install: build
	sudo cp web3 /usr/local/bin/web3

docker:
	docker build -t gochain/web3:latest .

push: docker
	docker push gochain/web3:latest

test:
	CGO_ENABLED=0 go vet ./...
	CGO_ENABLED=0 go test -v ./...

.PHONY: install test build docker push
