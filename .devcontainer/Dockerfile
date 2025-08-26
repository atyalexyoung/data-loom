FROM golang:1.25

# Install protobuf compiler
RUN apt-get update && apt-get install -y protobuf-compiler \
    && rm -rf /var/lib/apt/lists/*

# Install Go plugins for protobuf/grpc
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest \
    && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Make sure Go binaries are in PATH
ENV PATH="$PATH:$(go env GOPATH)/bin"

WORKDIR /workspace

# Optionally copy your Go modules for caching
COPY go.mod go.sum ./
RUN go mod download