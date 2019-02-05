build: 
	go build ./cmd/web3

install:
	go install ./cmd/web3

docker: 
	docker build -t gochain/web3:latest .

push: docker
	# todo: version these, or auto push this using CI
	docker push gochain/web3:latest

test:
	go test ./...

release:
	GOOS=linux go build -o web3_linux ./cmd/web3 
	# docker run --rm -v ${PWD}:/dev/web3 -w /dev/web3 treeder/go-dev go build -o web3_alpine ./cmd/web3 

.PHONY: install test build docker release
