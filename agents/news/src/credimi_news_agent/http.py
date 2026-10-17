# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""HTTP client for the I/O activities, with optional recording and offline replay.

* ``NEWS_AGENT_FIXTURES_DIR=<dir>``: answer every request from ``<dir>`` (no network). A request
  without a recorded file gets ``404`` with a hint, so a missing fixture fails visibly.
* ``NEWS_AGENT_RECORD_DIR=<dir>``: make real requests and save each response into ``<dir>``.
* ``GITHUB_TOKEN``: optional; raises the GitHub API limit from 60 to 5000 requests per hour.
  It is sent only to ``api.github.com`` and never recorded.
"""

from __future__ import annotations

import hashlib
import json
import os
import re
from pathlib import Path

import httpx

USER_AGENT = "credimi-news-agent/0.1 (+https://credimi.io)"
TIMEOUT = httpx.Timeout(20.0)


def fixture_name(url: httpx.URL) -> str:
    """Stable, readable file name for a request URL (query parameters sorted)."""
    query = "&".join(sorted(str(url.query, "utf-8").split("&"))) if url.query else ""
    key = f"{url.host}{url.path}?{query}"
    slug = re.sub(r"[^A-Za-z0-9]+", "_", f"{url.host}{url.path}").strip("_")[:120]
    return f"{slug}-{hashlib.sha256(key.encode()).hexdigest()[:10]}.json"


class FixtureTransport(httpx.AsyncBaseTransport):
    def __init__(self, directory: Path) -> None:
        self._dir = directory

    async def handle_async_request(self, request: httpx.Request) -> httpx.Response:
        path = self._dir / fixture_name(request.url)
        if not path.is_file():
            return httpx.Response(
                404,
                json={"message": f"no fixture {path.name} for {request.url}"},
                request=request,
            )
        data = json.loads(path.read_text())
        return httpx.Response(
            data["status"],
            headers={"content-type": data["content_type"]},
            content=data["body"].encode(),
            request=request,
        )


class RecordingTransport(httpx.AsyncBaseTransport):
    def __init__(self, directory: Path) -> None:
        self._dir = directory
        self._inner = httpx.AsyncHTTPTransport()

    async def handle_async_request(self, request: httpx.Request) -> httpx.Response:
        response = await self._inner.handle_async_request(request)
        body = await response.aread()
        self._dir.mkdir(parents=True, exist_ok=True)
        (self._dir / fixture_name(request.url)).write_text(
            json.dumps(
                {
                    "url": str(request.url),
                    "status": response.status_code,
                    "content_type": response.headers.get("content-type", ""),
                    "body": body.decode("utf-8", errors="replace"),
                },
                indent=1,
            )
        )
        # `aread` already decoded the body, so drop the encoding headers of the original.
        headers = {
            k: v
            for k, v in response.headers.items()
            if k.lower() not in ("content-encoding", "content-length", "transfer-encoding")
        }
        return httpx.Response(response.status_code, headers=headers, content=body, request=request)

    async def aclose(self) -> None:
        await self._inner.aclose()


def _transport() -> httpx.AsyncBaseTransport | None:
    if fixtures := os.environ.get("NEWS_AGENT_FIXTURES_DIR"):
        return FixtureTransport(Path(fixtures))
    if record := os.environ.get("NEWS_AGENT_RECORD_DIR"):
        return RecordingTransport(Path(record))
    return None


def client() -> httpx.AsyncClient:
    return httpx.AsyncClient(
        transport=_transport(),
        timeout=TIMEOUT,
        follow_redirects=False,
        headers={"User-Agent": USER_AGENT},
    )


def github_headers() -> dict[str, str]:
    headers = {
        "Accept": "application/vnd.github+json",
        "X-GitHub-Api-Version": "2022-11-28",
    }
    if (token := os.environ.get("GITHUB_TOKEN")) and not os.environ.get("NEWS_AGENT_FIXTURES_DIR"):
        headers["Authorization"] = f"Bearer {token}"
    return headers
