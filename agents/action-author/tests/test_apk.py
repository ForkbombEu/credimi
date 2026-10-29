# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

import hashlib
from types import SimpleNamespace

import httpx
import pytest

from credimi_action_author import apk, installer
from credimi_action_author.activities import ActionAuthorActivities
from credimi_action_author.contracts import PrepareBuildRequest, ResolvedBuild
from credimi_action_author.fake_device import FakeDevice
from credimi_action_author.models import ReplayRequest


def selected_build(content=b"selected APK", platform="android"):
    return PrepareBuildRequest(
        build=ResolvedBuild(
            wallet_id="wallet",
            version_id="version",
            version_identifier="organization/wallet/version",
            owner_namespace="organization",
            platform=platform,
            installer_name="selected.apk",
            installer_sha256=hashlib.sha256(content).hexdigest(),
            download_grant="test-only-capability",
        )
    )


@pytest.mark.parametrize("body,status", [(b"changed APK", 200), (b"", 302)])
async def test_changed_or_redirected_installer_is_not_accepted(monkeypatch, tmp_path, body, status):
    requests = []

    def serve(request):
        requests.append(request)
        return httpx.Response(
            status, content=body, headers={"Location": "https://other.example/installer"}
        )

    client = httpx.AsyncClient(transport=httpx.MockTransport(serve))
    monkeypatch.setattr(installer.httpx, "AsyncClient", lambda **kwargs: client)
    monkeypatch.setenv("ACTION_AUTHOR_CREDIMI_URL", "http://credimi.example")
    with pytest.raises((ValueError, httpx.HTTPStatusError)):
        await installer.download_installer(selected_build(), tmp_path / "installer.apk")
    assert [request.url.host for request in requests] == ["credimi.example"]


async def test_download_retains_exact_selected_bytes(monkeypatch, tmp_path):
    content = b"selected APK"

    def serve(request):
        if request.headers.get("Credimi-Artifact-Grant") != "test-only-capability":
            return httpx.Response(403)
        return httpx.Response(200, content=content)

    client = httpx.AsyncClient(transport=httpx.MockTransport(serve))
    monkeypatch.setattr(installer.httpx, "AsyncClient", lambda **kwargs: client)
    destination = tmp_path / "installer.apk"
    await installer.download_installer(selected_build(content), destination)
    assert destination.read_bytes() == content


async def test_package_mismatch_stops_before_install(monkeypatch):
    commands = []

    async def download(request, destination):
        destination.write_bytes(b"selected APK")

    async def command(arguments, timeout=60):
        commands.append(arguments)
        return "org.other.wallet"

    monkeypatch.setattr(apk, "download_installer", download)
    monkeypatch.setattr(apk, "command", command)
    request = selected_build()
    request.expected_app_id = "eu.europa.ec.euidi"
    with pytest.raises(ValueError, match="requested package"):
        await apk.prepare_apk(SimpleNamespace(serial="fake"), request)
    assert [arguments[0] for arguments in commands] == ["apkanalyzer"]


async def test_android_phone_refuses_an_ios_build():
    with pytest.raises(ValueError, match="only Android builds"):
        await apk.prepare_apk(SimpleNamespace(serial="fake"), selected_build(platform="ios"))


async def test_changed_installed_build_never_clears_or_replays(tmp_path):
    device = FakeDevice()
    device.installed_sha256 = "b" * 64
    activities = ActionAuthorActivities(device, tmp_path)
    request = ReplayRequest(
        app_id="eu.europa.ec.euidi",
        slot="onboarding",
        yaml="unused",
        installer_sha256="a" * 64,
    )
    with pytest.raises(ValueError, match="installed wallet changed"):
        await activities._run_once(request, tmp_path / "unused.yaml")
    assert device.calls == [("close_session",)]
