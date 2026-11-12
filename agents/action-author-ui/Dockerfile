# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

# Pinned base images and harness. These defaults are the only place the versions
# live: Compose passes an override only when the variable of the same name is set
# (Coolify, a local upgrade test), so a build that receives no build args (Coolify
# with Docker build secrets) still gets valid image names.
# The harness must equal the runners' agent harness (credimi-runner/agent/uv.lock).
ARG ACTION_AUTHOR_UI_PYTHON_VERSION=3.14-slim@sha256:af8a5aa14780025dbfbcfbc81bd8a91ebd87711dd5b0f48ed71090949dd7cb1d
ARG ACTION_AUTHOR_UI_UV_VERSION=0.13.6@sha256:af22a52f5eafef3d78e551599d405c02df61919d18028f0cea8eda93bc9986d0
ARG AGENT_HARNESS_VERSION=0.8.0
FROM ghcr.io/astral-sh/uv:${ACTION_AUTHOR_UI_UV_VERSION} AS uv
FROM python:${ACTION_AUTHOR_UI_PYTHON_VERSION}
ARG ACTION_AUTHOR_UI_PYTHON_VERSION
ARG ACTION_AUTHOR_UI_UV_VERSION
ARG AGENT_HARNESS_VERSION
COPY --from=uv /uv /usr/local/bin/uv
ENV UV_COMPILE_BYTECODE=1 UV_LINK_MODE=copy UV_PYTHON_DOWNLOADS=never \
    PATH=/app/.venv/bin:$PATH
WORKDIR /app
COPY pyproject.toml uv.lock README.md ./
COPY src ./src
# An override must still pin a digest; scripts/worktree-compose.sh checks it before
# Compose pulls, and this in-image check covers builds without that script (Coolify).
# Only the locked, hash-checked harness is installed: another version needs a uv.lock change.
RUN case "$ACTION_AUTHOR_UI_PYTHON_VERSION" in *@sha256:*) ;; *) echo "ACTION_AUTHOR_UI_PYTHON_VERSION must pin a digest (tag@sha256:...)" >&2; exit 1;; esac \
 && case "$ACTION_AUTHOR_UI_UV_VERSION" in *@sha256:*) ;; *) echo "ACTION_AUTHOR_UI_UV_VERSION must pin a digest (tag@sha256:...)" >&2; exit 1;; esac \
 && test -n "$AGENT_HARNESS_VERSION" \
 && uv sync --frozen --no-dev \
 && locked="$(/app/.venv/bin/python -c 'import importlib.metadata as m; print(m.version("temporal-agent-harness"))')" \
 && if [ "$locked" != "$AGENT_HARNESS_VERSION" ]; then \
      echo "AGENT_HARNESS_VERSION=${AGENT_HARNESS_VERSION} but uv.lock pins ${locked}; update uv.lock" >&2; \
      exit 1; \
    fi \
 && useradd --create-home --uid 10001 app
USER app
EXPOSE 8080
CMD ["/app/.venv/bin/python", "-m", "credimi_action_author_ui.app"]
