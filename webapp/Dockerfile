# SPDX-FileCopyrightText: 2025 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

FROM node:lts-alpine AS builder

RUN apk add --no-cache libc6-compat git
RUN wget "https://github.com/pnpm/pnpm/releases/latest/download/pnpm-linuxstatic-x64" -O /bin/pnpm && \
    chmod +x /bin/pnpm

WORKDIR /build

COPY ./webapp/ .
RUN git init
RUN git submodule add https://github.com/ForkbombEu/client_zencode client_zencode


EXPOSE $NODE_PORT
RUN pnpm i
CMD pnpm serve
