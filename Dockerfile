# SPDX-FileCopyrightText: 2025 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

FROM golang:1.24-alpine AS builder
WORKDIR /src

COPY go.mod go.sum .
RUN go mod download
COPY . ./
ENV GOCACHE=/go-cache
ENV GOMODCACHE=/gomod-cache
RUN --mount=type=cache,target=/gomod-cache --mount=type=cache,target=/go-cache \
    go build -o credimi main.go

FROM debian:12-slim
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
    --mount=target=/var/cache/apt,type=cache,sharing=locked \
    rm -f /etc/apt/apt.conf.d/docker-clean \
    && apt-get update \
    && apt-get -y --no-install-recommends install \
    build-essential make bash curl git tmux wget ca-certificates unzip 
WORKDIR /app

# install temporal
ARG TARFILE=temporal_cli_latest_linux_amd64.tar.gz
RUN wget 'https://temporal.download/cli/archive/latest?platform=linux&arch=amd64' -O $TARFILE
RUN tar xf $TARFILE
RUN rm $TARFILE
RUN mv temporal /usr/local/bin

# install credimi
COPY --from=builder /src/credimi /usr/local/bin/credimi
RUN chmod +x /usr/local/bin/credimi

# install mise and mise tools
SHELL ["/bin/bash", "-o", "pipefail", "-c"]
ENV MISE_DATA_DIR="/mise"
ENV MISE_CONFIG_DIR="/mise"
ENV MISE_CACHE_DIR="/mise/cache"
ENV MISE_INSTALL_PATH="/usr/local/bin/mise"
ENV PATH="/mise/shims:$PATH"
RUN curl https://mise.run | sh
COPY .mise.toml ./
RUN mise trust
RUN --mount=type=cache,target=/mise/cache mise i

# install bun deps and cache
COPY ./webapp/package.json webapp/
COPY ./webapp/bun.lock webapp/
WORKDIR /app/webapp
RUN bun i --frozen-lockfile

# install overmind
RUN curl -sLO https://github.com/DarthSim/overmind/releases/download/v2.5.1/overmind-v2.5.1-linux-amd64.gz
RUN gunzip overmind-v2.5.1-linux-amd64.gz
RUN mv overmind-v2.5.1-linux-amd64 /usr/local/bin/overmind
RUN chmod +x /usr/local/bin/overmind

WORKDIR /app

# install the stepci-captured-runner
RUN mkdir .bin
RUN wget https://github.com/ForkbombEu/stepci-captured-runner/releases/latest/download/stepci-captured-runner-Linux-x86_64 -O .bin/stepci-captured-runner && chmod +x .bin/stepci-captured-runner

#install et-tu-cesr
RUN wget https://github.com/ForkbombEu/et-tu-cesr/releases/latest/download/et-tu-cesr-linux-amd64 -O .bin/et-tu-cesr && chmod +x .bin/et-tu-cesr




# copy everything
COPY . ./
RUN credimi migrate up


WORKDIR /app/webapp
ARG PUBLIC_POCKETBASE_URL
ENV PUBLIC_POCKETBASE_URL ${PUBLIC_POCKETBASE_URL}
ENV DATA_DB_PATH /app/pb_data/data.db
RUN bun run build
WORKDIR /app

HEALTHCHECK --interval=30s --timeout=10s --start-period=120s --retries=3 CMD curl --fail http://localhost:8090 || exit 1
CMD ["overmind", "s", "-f", "/app/Procfile" ]
