# Use official Go image with latest stable version
FROM golang:1.25-bookworm

# Install additional tools for development and firewall
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
        git \
        ca-certificates \
        curl \
        vim \
        less \
        man-db \
        bash-completion \
        iptables \
        ipset \
        iproute2 \
        dnsutils \
        aggregate \
        jq \
        gh \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Create a non-root user
ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=$USER_UID

RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME \
    && apt-get update \
    && apt-get install -y sudo \
    && echo $USERNAME ALL=\(root\) NOPASSWD:ALL > /etc/sudoers.d/$USERNAME \
    && chmod 0440 /etc/sudoers.d/$USERNAME \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install Go tools for development
RUN go install golang.org/x/tools/gopls@latest \
    && go install github.com/go-delve/delve/cmd/dlv@latest \
    && go install honnef.co/go/tools/cmd/staticcheck@latest \
    && go install github.com/fatih/gomodifytags@latest \
    && go install github.com/josharian/impl@latest \
    && go install github.com/cweill/gotests/gotests@latest

# Verify Go supports FreeBSD cross-compilation
RUN go env && \
    echo "Testing FreeBSD cross-compilation support..." && \
    GOOS=freebsd GOARCH=amd64 go version

# Copy and set up firewall script
COPY init-firewall.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/init-firewall.sh && \
    echo "$USERNAME ALL=(root) NOPASSWD: /usr/local/bin/init-firewall.sh" > /etc/sudoers.d/$USERNAME-firewall && \
    chmod 0440 /etc/sudoers.d/$USERNAME-firewall

# Set the working directory
WORKDIR /workspace

# Switch to non-root user
USER $USERNAME

# Set Go environment for the user
ENV GOPATH=/go
ENV PATH=$PATH:/go/bin

# Create go directories with proper permissions
RUN sudo mkdir -p /go/pkg /go/bin && \
    sudo chown -R $USERNAME:$USERNAME /go
