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

# .NET SDK
RUN wget https://packages.microsoft.com/config/debian/12/packages-microsoft-prod.deb -O packages-microsoft-prod.deb \
    && sudo dpkg -i packages-microsoft-prod.deb \
    && rm packages-microsoft-prod.deb \
    && sudo apt-get update \
    && sudo apt-get install -y dotnet-sdk-8.0 \
    && sudo rm -rf /var/lib/apt/lists/*

USER vscode

# Go stuff
ENV GOPATH=/home/vscode/go
ENV PATH="$GOPATH/bin:$PATH"

RUN mkdir -p $GOPATH \
    && go install golang.org/x/tools/gopls@latest \
    && go install github.com/go-delve/delve/cmd/dlv@latest

COPY go.mod go.sum ./
RUN go mod download

