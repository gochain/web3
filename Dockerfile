# build stage
FROM golang:1.12-alpine AS build-env
RUN apk --no-cache add build-base git bzr mercurial gcc
ENV D=/web3
WORKDIR $D
# cache dependencies
ADD go.mod $D
ADD go.sum $D
RUN go mod download
# now build
ADD . $D
RUN cd $D && go build -o web3-alpine ./cmd/web3 && cp web3-alpine /tmp/

# final stage
FROM alpine
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build-env /tmp/web3-alpine /app/web3
ENTRYPOINT ["/app/web3"]
