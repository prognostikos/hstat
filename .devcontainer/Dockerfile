ARG GO_VERSION="1.23.4"
ARG BUN_VERSION="1.1.38"

FROM golang:${GO_VERSION} AS gobuilder

FROM oven/bun:${BUN_VERSION} AS bunbuilder

FROM debian:bookworm-slim

# Copy Go from builder
COPY --from=gobuilder /usr/local/go /usr/local/go

# Copy Bun from builder
COPY --from=bunbuilder /usr/local/bin/bun /usr/local/bin/bun

ARG USERNAME=dev
ARG USER_UID=1000
ARG USER_GID=1000

ENV DEBIAN_FRONTEND=noninteractive \
    GOPATH=/home/${USERNAME}/go \
    PATH="/usr/local/go/bin:/home/${USERNAME}/go/bin:/home/${USERNAME}/.local/bin:/home/${USERNAME}/.bun/bin:${PATH}"

# Install base packages
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    git \
    fzf \
    gnupg2 \
    jq \
    less \
    locales \
    man-db \
    nodejs \
    openssh-client \
    ripgrep \
    sudo \
    tree \
    unzip \
    vim \
    whois \
    zsh \
    && rm -rf /var/lib/apt/lists/*

# Setup locale
RUN localedef -i en_US -c -f UTF-8 -A /usr/share/locale/locale.alias en_US.UTF-8
ENV LANG=en_US.UTF-8

# Create bunx hardlink
RUN ln /usr/local/bin/bun /usr/local/bin/bunx

# Create developer user
RUN groupadd --gid ${USER_GID} ${USERNAME} \
    && useradd --uid ${USER_UID} --gid ${USER_GID} -m ${USERNAME} \
    && echo "${USERNAME} ALL=(root) NOPASSWD:ALL" > /etc/sudoers.d/${USERNAME} \
    && chmod 0440 /etc/sudoers.d/${USERNAME} \
    && chsh -s /usr/bin/zsh ${USERNAME}

RUN mkdir -p /workspaces && chown ${USERNAME}:${USERNAME} /workspaces

USER ${USERNAME}

# Create directories
RUN mkdir -p ~/.local/bin ~/go

# Install global tools with bun
RUN bun install --global @anthropic-ai/claude-code

# Install Go tools
RUN go install golang.org/x/tools/gopls@latest \
    && go install github.com/go-delve/delve/cmd/dlv@latest

WORKDIR /workspaces

ENV RUNNING_IN_DEVCONTAINER=1
