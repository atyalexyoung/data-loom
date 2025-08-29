FROM golang:1.25

# Make sure Go binaries are in PATH
ENV PATH="$PATH:$(go env GOPATH)/bin"

WORKDIR /workspace

# Optionally copy your Go modules for caching
COPY go.mod go.sum ./
RUN go mod download