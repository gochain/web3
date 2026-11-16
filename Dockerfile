# Build stage (pure Go, zero CGO)
FROM golang:1.26-alpine AS build-env
ARG VERSION=dev
WORKDIR /web3
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.Version=${VERSION}" -o /tmp/web3 ./cmd/web3

# Final stage
FROM alpine
RUN apk add --no-cache ca-certificates git
WORKDIR /app
COPY --from=build-env /tmp/web3 /app/web3
ENTRYPOINT ["/app/web3"]
