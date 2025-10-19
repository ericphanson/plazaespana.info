# Use official Go image with latest stable version
FROM golang:1.25-bookworm

ARG TZ
ENV TZ="$TZ"

ARG CLAUDE_CODE_VERSION=latest
ARG GH_VERSION=2.63.2
ARG GIT_DELTA_VERSION=0.18.2
ARG ZSH_IN_DOCKER_VERSION=1.2.0
ARG NODE_MAJOR=20

# Configure locale to fix bash locale warnings
RUN apt-get update && apt-get install -y locales && \
    sed -i '/en_US.UTF-8/s/^# //g' /etc/locale.gen && \
    locale-gen en_US.UTF-8 && \
    apt-get clean && rm -rf /var/lib/apt/lists/*

ENV LANG=en_US.UTF-8 \
    LANGUAGE=en_US:en \
    LC_ALL=en_US.UTF-8

# Install Node.js repository
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    gnupg \
    wget \
    && mkdir -p /etc/apt/keyrings \
    && curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg \
    && echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_$NODE_MAJOR.x nodistro main" > /etc/apt/sources.list.d/nodesource.list \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Install additional tools for development and firewall
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
        git \
        ca-certificates \
        curl \
        vim \
        nano \
        less \
        man-db \
        bash-completion \
        iptables \
        ipset \
        iproute2 \
        dnsutils \
        aggregate \
        jq \
        fzf \
        zsh \
        unzip \
        gnupg2 \
        nodejs \
        build-essential \
        procps \
        direnv \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Pin specific gh CLI version to prevent drift
RUN ARCH=$(dpkg --print-architecture) && \
    wget "https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_${ARCH}.deb" && \
    dpkg -i "gh_${GH_VERSION}_linux_${ARCH}.deb" && \
    rm "gh_${GH_VERSION}_linux_${ARCH}.deb"

# Install Git Delta
RUN ARCH=$(dpkg --print-architecture) && \
    wget "https://github.com/dandavison/delta/releases/download/${GIT_DELTA_VERSION}/git-delta_${GIT_DELTA_VERSION}_${ARCH}.deb" && \
    dpkg -i "git-delta_${GIT_DELTA_VERSION}_${ARCH}.deb" && \
    rm "git-delta_${GIT_DELTA_VERSION}_${ARCH}.deb"

# Create a non-root user
ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=$USER_UID

RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m -s /bin/zsh $USERNAME \
    && apt-get update \
    && apt-get install -y sudo \
    && echo $USERNAME ALL=\(root\) NOPASSWD:ALL > /etc/sudoers.d/$USERNAME \
    && chmod 0440 /etc/sudoers.d/$USERNAME \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Ensure user has access to /usr/local/share
RUN mkdir -p /usr/local/share/npm-global && \
    chown -R $USERNAME:$USERNAME /usr/local/share

# Persist bash history
RUN SNIPPET="export PROMPT_COMMAND='history -a' && export HISTFILE=/commandhistory/.bash_history" \
    && mkdir /commandhistory \
    && touch /commandhistory/.bash_history \
    && chown -R $USERNAME /commandhistory

# Set `DEVCONTAINER` environment variable to help with orientation
ENV DEVCONTAINER=true

# Create workspace and config directories and set permissions
RUN mkdir -p /workspace /home/$USERNAME/.claude && \
    chown -R $USERNAME:$USERNAME /workspace /home/$USERNAME/.claude

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

# Switch to non-root user for zsh setup
USER $USERNAME

# Install global npm packages
ENV NPM_CONFIG_PREFIX=/usr/local/share/npm-global
ENV PATH=$PATH:/usr/local/share/npm-global/bin

# Set the default shell to zsh
ENV SHELL=/bin/zsh

# Set the default editor and visual
ENV EDITOR=nano
ENV VISUAL=nano

# Set up zsh with plugins
RUN sh -c "$(wget -O- https://github.com/deluan/zsh-in-docker/releases/download/v${ZSH_IN_DOCKER_VERSION}/zsh-in-docker.sh)" -- \
    -p git \
    -p fzf \
    -a "source /usr/share/doc/fzf/examples/key-bindings.zsh" \
    -a "source /usr/share/doc/fzf/examples/completion.zsh" \
    -a "export PROMPT_COMMAND='history -a' && export HISTFILE=/commandhistory/.bash_history" \
    -a "alias check-disk='df -h /workspace && du -sh /workspace/* 2>/dev/null | sort -h'" \
    -a "eval \"\$(direnv hook zsh)\"" \
    -x

# Install Claude Code
RUN npm install -g @anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}

# Copy and set up scripts
USER root
COPY .devcontainer/init-firewall.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/init-firewall.sh && \
    echo "$USERNAME ALL=(root) NOPASSWD: /usr/local/bin/init-firewall.sh" > /etc/sudoers.d/$USERNAME-firewall && \
    chmod 0440 /etc/sudoers.d/$USERNAME-firewall

# Switch back to non-root user
USER $USERNAME

# Set the working directory
WORKDIR /workspace

# Set Go environment for the user
ENV GOPATH=/go
ENV PATH=$PATH:/go/bin

# Create go directories with proper permissions
RUN sudo mkdir -p /go/pkg /go/bin && \
    sudo chown -R $USERNAME:$USERNAME /go
