# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

ARG PYTHON_VERSION
ARG UV_VERSION
FROM ghcr.io/astral-sh/uv:${UV_VERSION} AS uv
FROM python:${PYTHON_VERSION}
ARG PYTHON_VERSION
ARG UV_VERSION
ARG AGENT_HARNESS_VERSION
COPY --from=uv /uv /usr/local/bin/uv
ENV UV_COMPILE_BYTECODE=1 UV_LINK_MODE=copy UV_PYTHON_DOWNLOADS=never \
    PATH=/app/.venv/bin:$PATH
WORKDIR /app
COPY pyproject.toml uv.lock README.md ./
COPY src ./src
# scripts/worktree-compose.sh checks the base image digest pins before Compose pulls
# them; this in-image check is the last line of defence for deployments that build
# without that script (Coolify).
# The harness must equal the runners' agent harness (credimi-runner/agent/uv.lock).
# An override of the locked version installs from PyPI without hash checks.
RUN case "$PYTHON_VERSION" in *@sha256:*) ;; *) echo "PYTHON_VERSION must pin a digest (tag@sha256:...)" >&2; exit 1;; esac \
 && case "$UV_VERSION" in *@sha256:*) ;; *) echo "UV_VERSION must pin a digest (tag@sha256:...)" >&2; exit 1;; esac \
 && test -n "$AGENT_HARNESS_VERSION" \
 && uv sync --frozen --no-dev \
 && locked="$(/app/.venv/bin/python -c 'import importlib.metadata as m; print(m.version("temporal-agent-harness"))')" \
 && if [ "$locked" != "$AGENT_HARNESS_VERSION" ]; then \
      uv pip install --python /app/.venv/bin/python "temporal-agent-harness[ui]==${AGENT_HARNESS_VERSION}"; \
    fi \
 && useradd --create-home --uid 10001 app
USER app
EXPOSE 8080
CMD ["/app/.venv/bin/python", "-m", "credimi_action_author_ui.app"]
