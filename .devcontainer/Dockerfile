FROM golang:1.25

# Create a non-root user (vscode) with UID 1000 (matches common host UID)
RUN useradd -m -s /bin/bash -u 1000 vscode \
    && apt-get update \
    && apt-get install -y sudo \
    && echo "vscode ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers \
    && rm -rf /var/lib/apt/lists/*

# Make sure Go binaries are in PATH
ENV PATH="$PATH:$(go env GOPATH)/bin"

# Set workspace and give ownership to vscode
WORKDIR /workspace
RUN chown -R vscode:vscode /workspace
# Optionally copy your Go modules for caching
COPY go.mod go.sum ./
RUN go mod download