ARG BASE_IMAGE=oraclelinux:10
ARG GO_VERSION=1.26.0

# ---------------------------------------------------------
# Builder Stage
# ---------------------------------------------------------
FROM ${BASE_IMAGE} AS builder

# OCI Image Annotations
LABEL org.opencontainers.image.title="scrapedoctl" \
      org.opencontainers.image.description="Go 1.26 based MCP CLI server for Scrape.do" \
      org.opencontainers.image.source="https://github.com/mr-pmillz/scrapedoctl" \
      org.opencontainers.image.authors="Your Name <your.email@example.com>" \
      org.opencontainers.image.vendor="mr-pmillz" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.base.name="docker.io/library/oraclelinux:10"

# Install dependencies required for development and building Go 1.26
RUN dnf update -y && \
    dnf install -y \
    curl \
    tar \
    gzip \
    git \
    gcc \
    ca-certificates \
    && dnf clean all

# Download and install Go 1.26
ARG GO_VERSION
RUN curl -fsSL "https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz" -o go.tar.gz && \
    tar -C /usr/local -xzf go.tar.gz && \
    rm go.tar.gz

ENV PATH="/usr/local/go/bin:/root/go/bin:${PATH}"

# Setup non-root user for execution
RUN groupadd -r scrape && useradd -r -g scrape -s /sbin/nologin scrape

# Install golangci-lint v2 (required for Go 1.26)
RUN go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.3

# Install sqlc
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Install formatting tools
RUN go install mvdan.cc/gofumpt@latest && \
    go install golang.org/x/tools/cmd/goimports@latest && \
    go install github.com/golangci/golines@latest

# Install PowerShell
RUN dnf install -y https://github.com/PowerShell/PowerShell/releases/download/v7.6.0/powershell-7.6.0-1.rh.x86_64.rpm && \
    dnf clean all

# Install PSScriptAnalyzer
RUN pwsh -Command "Install-Module -Name PSScriptAnalyzer -Force -Scope CurrentUser"

WORKDIR /src

# We won't copy go.mod immediately in dev mode if we want to mount it, 
# but for the final build it's needed.
# For local dev via podman run, we can just map the current directory.
COPY . .

# Avoid failing if go.mod doesn't exist yet (we will initialize it inside container)
RUN if [ -f go.mod ]; then go mod download && go mod verify; fi

# Build the CLI application
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
RUN if [ -f cmd/scrapedoctl/main.go ]; then \
    CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w \
      -X github.com/mr-pmillz/scrapedoctl/internal/version.Version=${VERSION} \
      -X github.com/mr-pmillz/scrapedoctl/internal/version.GitCommit=${GIT_COMMIT} \
      -X github.com/mr-pmillz/scrapedoctl/internal/version.BuildDate=${BUILD_DATE}" \
    -o /bin/scrapedoctl ./cmd/scrapedoctl; fi

# ---------------------------------------------------------
# Production Stage
# ---------------------------------------------------------
FROM scratch AS production

# OCI Image Annotations for Production
LABEL org.opencontainers.image.title="scrapedoctl" \
      org.opencontainers.image.description="Standalone binary of the scrapedoctl MCP server"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /bin/scrapedoctl /bin/

USER scrape:scrape

ENTRYPOINT ["/bin/scrapedoctl"]
