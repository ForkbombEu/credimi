# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Download the single installer authorized for a session and check its bytes.

Android installers are APKs, iOS installers are zipped simulator ``.app`` bundles; both are
streamed from Credimi with the build-bound grant and must hash to the resolved digest.
"""

import hashlib
import os
from pathlib import Path
from urllib.parse import urlsplit

import httpx

from .contracts import PrepareBuildRequest
from .device import _run

MAX_INSTALLER_BYTES = 500 * 1024 * 1024


def credimi_url() -> str:
    base = os.environ.get("ACTION_AUTHOR_CREDIMI_URL", "http://localhost:8090").rstrip("/")
    parsed = urlsplit(base)
    if (
        parsed.scheme not in {"http", "https"}
        or not parsed.hostname
        or parsed.username
        or parsed.password
        or parsed.query
        or parsed.fragment
        or parsed.path
    ):
        raise ValueError("ACTION_AUTHOR_CREDIMI_URL must be a trusted HTTP(S) origin")
    return base


async def download_installer(req: PrepareBuildRequest, destination: Path) -> None:
    digest = hashlib.sha256()
    size = 0
    async with httpx.AsyncClient(
        timeout=httpx.Timeout(120, connect=10), follow_redirects=False, trust_env=False
    ) as client:
        async with client.stream(
            "GET",
            credimi_url() + "/api/action-author/installer",
            headers={"Credimi-Artifact-Grant": req.build.download_grant},
        ) as response:
            response.raise_for_status()
            with destination.open("wb") as stream:
                async for chunk in response.aiter_bytes(64 * 1024):
                    size += len(chunk)
                    if size > MAX_INSTALLER_BYTES:
                        raise ValueError("selected installer exceeds the 500 MiB limit")
                    digest.update(chunk)
                    stream.write(chunk)
    if digest.hexdigest() != req.build.installer_sha256:
        raise ValueError("selected installer SHA256 does not match the resolved Credimi build")


async def command(arguments: list[str], timeout: float = 60) -> str:
    """Run a host command; its stripped output, or RuntimeError with the output on failure."""
    code, output = await _run(arguments, timeout)
    if code:
        raise RuntimeError(f"{arguments[0]} failed: {output.strip()}")
    return output.strip()
