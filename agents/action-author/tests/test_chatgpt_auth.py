# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Security and lifecycle tests for the local ChatGPT OAuth credential store."""

import json
import stat
from datetime import UTC, datetime, timedelta
from types import SimpleNamespace
from urllib.parse import parse_qs, urlparse

import jwt
import pytest
from cryptography.hazmat.primitives.asymmetric import rsa

from credimi_action_author import chatgpt_auth as auth

PRIVATE = rsa.generate_private_key(public_exponent=65537, key_size=2048)
OTHER_PRIVATE = rsa.generate_private_key(public_exponent=65537, key_size=2048)


def identity(key=PRIVATE, **overrides):
    claims = {
        "iss": auth.ISSUER,
        "aud": "oaiapp_test",
        "sub": "account-1",
        "email": "person@example.test",
        "nonce": "expected-nonce",
        "iat": datetime.now(UTC),
        "exp": datetime.now(UTC) + timedelta(minutes=10),
    }
    claims.update(overrides)
    return jwt.encode(claims, key, algorithm="RS256", headers={"kid": "test"})


@pytest.fixture
def verify_locally(monkeypatch):
    class Keys:
        def get_signing_key_from_jwt(self, token):
            return type("Key", (), {"key": PRIVATE.public_key()})()

    monkeypatch.setattr(auth.jwt, "PyJWKClient", lambda url: Keys())


def test_id_token_requires_valid_signature_audience_and_nonce(monkeypatch, verify_locally):
    assert auth.verify_chatgpt_id_token(identity(), "oaiapp_test", "expected-nonce")["sub"] == (
        "account-1"
    )
    for token, client_id, nonce in (
        (identity(nonce="wrong"), "oaiapp_test", "expected-nonce"),
        (identity(aud="another-client"), "oaiapp_test", "expected-nonce"),
        (identity(key=OTHER_PRIVATE), "oaiapp_test", "expected-nonce"),
    ):
        with pytest.raises(RuntimeError, match="identity validation failed"):
            auth.verify_chatgpt_id_token(token, client_id, nonce)


def test_callback_validates_state_before_using_an_authorization_error():
    with pytest.raises(RuntimeError, match="state did not match"):
        auth._validate_callback("error=access_denied&state=wrong", "right", "dynamic_agent_client")
    with pytest.raises(RuntimeError, match="not authorized"):
        auth._validate_callback("error=access_denied&state=right", "right", "dynamic_agent_client")


def test_registration_and_returning_client_ids_cannot_be_substituted():
    code, client = auth._validate_callback(
        "code=one-time&state=ok&client_id=oaiapp_issued", "ok", "dynamic_agent_client"
    )
    assert (code, client) == ("one-time", "oaiapp_issued")
    with pytest.raises(RuntimeError, match="different client"):
        auth._validate_callback("code=x&state=ok&client_id=oaiapp_other", "ok", "oaiapp_issued")
    with pytest.raises(RuntimeError, match="code and issued client"):
        auth._validate_callback("state=ok", "ok", "dynamic_agent_client")


def test_plan_scope_and_returning_identity_are_required(monkeypatch):
    monkeypatch.setattr(
        auth, "verify_chatgpt_id_token", lambda *args: {"sub": "new", "iss": auth.ISSUER}
    )
    tokens = {"id_token": "id", "scope": "openid email", "access_token": "a", "refresh_token": "r"}
    with pytest.raises(RuntimeError, match="plan usage was not authorized"):
        auth._credentials(tokens, "oaiapp_test", "urn:uuid:host", "nonce", None)
    tokens["scope"] = auth.SCOPES
    with pytest.raises(RuntimeError, match="differs"):
        auth._credentials(tokens, "oaiapp_test", "urn:uuid:host", "nonce", {"subject": "old"})


def test_credential_file_is_owner_only_from_creation(tmp_path):
    path = tmp_path / "private" / "chatgpt.json"
    record = {"access_token": "secret", "client_id": "oaiapp_test"}
    auth._save(record, path)
    assert stat.S_IMODE(path.stat().st_mode) == 0o600
    assert stat.S_IMODE(path.parent.stat().st_mode) == 0o700
    assert json.loads(path.read_text()) == record


def saved_record(**overrides):
    record = {
        "email": "person@example.test",
        "issuer": auth.ISSUER,
        "subject": "account-1",
        "client_id": "oaiapp_test",
        "ext_agent_host_id": "urn:uuid:stable-host",
        "id_token": "identity",
        "access_token": "old-access",
        "refresh_token": "old-refresh",
        "token_type": "Bearer",
        "expires_in": 3600,
        "scopes": auth.SCOPES.split(),
        "saved_at": datetime.now(UTC).isoformat(),
    }
    record.update(overrides)
    return record


async def test_refresh_rotates_both_tokens_and_preserves_registration(tmp_path, monkeypatch):
    path = tmp_path / "chatgpt.json"
    auth._save(saved_record(expires_in=1), path)
    monkeypatch.setenv("ACTION_AUTHOR_CHATGPT_CREDENTIALS", str(path))
    monkeypatch.setattr(
        auth,
        "_refreshed",
        lambda old: saved_record(
            access_token="new-access",
            refresh_token="new-refresh",
            ext_agent_host_id=old["ext_agent_host_id"],
        ),
    )
    assert await auth.chatgpt_access_token() == "new-access"
    stored = auth._load(path)
    assert stored["refresh_token"] == "new-refresh"
    assert stored["ext_agent_host_id"] == "urn:uuid:stable-host"


async def test_valid_token_does_not_refresh(tmp_path, monkeypatch):
    path = tmp_path / "chatgpt.json"
    auth._save(saved_record(), path)
    monkeypatch.setenv("ACTION_AUTHOR_CHATGPT_CREDENTIALS", str(path))
    monkeypatch.setattr(auth, "_refreshed", lambda old: pytest.fail("unexpected refresh"))
    assert await auth.chatgpt_access_token() == "old-access"


async def test_terminal_refresh_requires_login_without_erasing_credentials(tmp_path, monkeypatch):
    path = tmp_path / "chatgpt.json"
    auth._save(saved_record(expires_in=1), path)
    monkeypatch.setenv("ACTION_AUTHOR_CHATGPT_CREDENTIALS", str(path))

    def expired(old):
        raise auth.OAuthError("invalid_grant", 400)

    monkeypatch.setattr(auth, "_refreshed", expired)
    with pytest.raises(RuntimeError, match="action-author.login"):
        await auth.chatgpt_access_token()
    assert auth._load(path)["refresh_token"] == "old-refresh"


def test_refresh_error_preserves_oauth_code_without_response_body():
    class Response:
        status_code = 400
        is_error = True

        def json(self):
            return {"error": "invalid_grant", "error_description": "sensitive details"}

    with pytest.raises(auth.OAuthError) as caught:
        auth._token_response(Response())
    assert caught.value.code == "invalid_grant"
    assert "sensitive details" not in str(caught.value)


def test_existing_credential_parent_permissions_are_preserved(tmp_path):
    directory = tmp_path / "shared-config"
    directory.mkdir(mode=0o755)
    path = directory / "chatgpt.json"
    auth._save({"access_token": "synthetic"}, path)
    assert stat.S_IMODE(directory.stat().st_mode) == 0o755
    assert stat.S_IMODE(path.stat().st_mode) == 0o600


def test_relogin_does_not_print_or_open_an_id_token_url(monkeypatch, capsys):
    record = saved_record(id_token="synthetic-sensitive-id-token")
    monkeypatch.setattr(auth, "_load", lambda *args: record)
    closed = []
    monkeypatch.setattr(
        auth,
        "ThreadingHTTPServer",
        lambda *args: SimpleNamespace(
            server_address=("127.0.0.1", 1455), server_close=lambda: closed.append(True)
        ),
    )
    opened = []

    def open_browser(url):
        opened.append(url)
        raise RuntimeError("stop before real authorization")

    monkeypatch.setattr(auth.webbrowser, "open", open_browser)
    with pytest.raises(RuntimeError, match="stop before real authorization"):
        auth.chatgpt_login()
    assert record["id_token"] not in capsys.readouterr().out
    assert record["id_token"] not in opened[0]
    query = parse_qs(urlparse(opened[0]).query)
    assert query["client_id"] == [record["client_id"]]
    assert query["login_hint"] == [record["email"]]
    assert closed == [True]
