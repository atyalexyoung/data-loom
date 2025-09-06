FROM golang:1.25

# Create non-root vscode user
RUN useradd -m -s /bin/bash -u 1000 vscode \
    && apt-get update \
    && apt-get install -y sudo \
    && echo "vscode ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers \
    && rm -rf /var/lib/apt/lists/*

# Set workspace and give ownership to vscode
WORKDIR /workspace
RUN chown -R vscode:vscode /workspace

# Switch to vscode user
USER vscode

# Use a GOPATH inside the home directory (writable)
ENV GOPATH=/home/vscode/go
ENV PATH="$GOPATH/bin:$PATH"

# Install Go tools (gopls + delve)
RUN mkdir -p $GOPATH \
    && go install golang.org/x/tools/gopls@latest \
    && go install github.com/go-delve/delve/cmd/dlv@latest

# Optionally copy Go modules for caching
COPY go.mod go.sum ./
RUN go mod download

