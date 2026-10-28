# SPDX-FileCopyrightText: 2026 Forkbomb BV
#
# SPDX-License-Identifier: AGPL-3.0-or-later

"""Local Sign in with ChatGPT credentials for ChatGPT-plan Responses API calls.

This is the documented public-client flow for local/open-source apps, not Codex
CLI authentication. Credentials never enter Temporal or the workflow history.
"""

import asyncio
import base64
import hashlib
import json
import os
import secrets
import threading
import time
import webbrowser
from datetime import UTC, datetime
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from typing import Any
from urllib.parse import parse_qs, urlencode, urlparse
from uuid import uuid4

import httpx
import jwt

ISSUER = "https://auth.openai.com"
AUTHORIZE_URL = f"{ISSUER}/api/accounts/authorize"
TOKEN_URL = f"{ISSUER}/api/accounts/oauth/token"
JWKS_URL = f"{ISSUER}/.well-known/jwks.json"
RESOURCE = "https://api.openai.com/v1"
PLAN_SCOPE = "chatgpt.tokens.use.direct"
SCOPES = f"openid profile email offline_access resource.invoke {PLAN_SCOPE}"
TERMINAL_REFRESH_ERRORS = {
    "invalid_grant",
    "invalid_refresh_token",
    "token_expired",
    "refresh_token_expired",
    "refresh_token_invalidated",
    "refresh_token_reused",
}
_refresh_lock = asyncio.Lock()


def credential_path() -> Path:
    configured = os.environ.get("ACTION_AUTHOR_CHATGPT_CREDENTIALS", "").strip()
    return (
        Path(configured).expanduser()
        if configured
        else Path.home() / ".config" / "credimi-action-author" / "chatgpt.json"
    )


def _load(path: Path | None = None) -> dict[str, Any]:
    path = path or credential_path()
    try:
        record = json.loads(path.read_text())
    except FileNotFoundError as exc:
        raise RuntimeError(
            "ChatGPT is not connected. Run `make action-author.login` in a local terminal."
        ) from exc
    except (OSError, json.JSONDecodeError) as exc:
        raise RuntimeError(f"Cannot read ChatGPT credentials in {path}: {exc}") from exc
    required = ("client_id", "ext_agent_host_id", "access_token", "refresh_token", "saved_at")
    if not isinstance(record, dict) or any(not record.get(key) for key in required):
        raise RuntimeError(f"ChatGPT credentials in {path} are incomplete; sign in again.")
    return record


def _save(record: dict[str, Any], path: Path | None = None) -> None:
    path = path or credential_path()
    path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    temporary = path.with_name(f".{path.name}.{os.getpid()}.{secrets.token_hex(4)}.tmp")
    try:
        descriptor = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(descriptor, "w") as stream:
            stream.write(json.dumps(record, indent=2) + "\n")
        temporary.replace(path)
    finally:
        temporary.unlink(missing_ok=True)


def _expiry(record: dict[str, Any]) -> float:
    saved = datetime.fromisoformat(record["saved_at"].replace("Z", "+00:00")).timestamp()
    return saved + float(record.get("expires_in", 3600))


def _token_response(response: httpx.Response) -> dict[str, Any]:
    try:
        payload = response.json()
    except (json.JSONDecodeError, ValueError) as exc:
        raise RuntimeError(f"OpenAI OAuth returned HTTP {response.status_code}.") from exc
    if response.is_error:
        code = payload.get("error", "oauth_error") if isinstance(payload, dict) else "oauth_error"
        error = RuntimeError(f"OpenAI OAuth failed: {code} (HTTP {response.status_code}).")
        error.oauth_error = code
        raise error
    if not isinstance(payload, dict):
        raise RuntimeError("OpenAI OAuth returned an invalid response.")
    return payload


def verify_chatgpt_id_token(token: str, client_id: str, nonce: str) -> dict[str, Any]:
    """Verify OpenAI's signed identity token; an unsigned claim is never trusted."""
    try:
        key = jwt.PyJWKClient(JWKS_URL).get_signing_key_from_jwt(token).key
        claims = jwt.decode(
            token,
            key,
            algorithms=["RS256"],
            audience=client_id,
            issuer=ISSUER,
            options={"require": ["sub", "exp", "iat"]},
        )
    except jwt.PyJWTError as exc:
        raise RuntimeError(f"ChatGPT identity validation failed: {exc}") from exc
    if not secrets.compare_digest(str(claims.get("nonce", "")), nonce):
        raise RuntimeError("ChatGPT identity validation failed: nonce mismatch.")
    return claims


def _validate_callback(query: str, state: str, client_id: str) -> tuple[str, str]:
    parameters = parse_qs(query, keep_blank_values=True)
    returned_state = parameters.get("state", [""])[0]
    if not returned_state or not secrets.compare_digest(returned_state, state):
        raise RuntimeError("ChatGPT sign-in state did not match; the code was not used.")
    if error := parameters.get("error", [""])[0]:
        raise RuntimeError(f"ChatGPT sign-in was not authorized: {error}.")
    code = parameters.get("code", [""])[0]
    returned_client = parameters.get("client_id", [client_id])[0]
    if not code or not returned_client:
        raise RuntimeError("ChatGPT sign-in did not return a code and issued client ID.")
    if client_id != "dynamic_agent_client" and returned_client != client_id:
        raise RuntimeError("ChatGPT sign-in returned a different client registration.")
    return code, returned_client


def _credentials(
    tokens: dict[str, Any],
    client_id: str,
    host_id: str,
    nonce: str,
    previous: dict[str, Any] | None,
) -> dict[str, Any]:
    if PLAN_SCOPE not in tokens.get("scope", "").split():
        raise RuntimeError(
            "ChatGPT plan usage was not authorized. Enable it during sign-in; "
            "no inference was attempted."
        )
    claims = verify_chatgpt_id_token(tokens.get("id_token", ""), client_id, nonce)
    if previous and claims["sub"] != previous.get("subject"):
        raise RuntimeError("The signed-in ChatGPT account differs from the saved registration.")
    for key in ("access_token", "refresh_token"):
        if not isinstance(tokens.get(key), str) or not tokens[key]:
            raise RuntimeError(f"OpenAI OAuth did not return {key}.")
    return {
        "email": claims.get("email", ""),
        "issuer": claims["iss"],
        "subject": claims["sub"],
        "client_id": client_id,
        "ext_agent_host_id": host_id,
        "id_token": tokens["id_token"],
        "access_token": tokens["access_token"],
        "refresh_token": tokens["refresh_token"],
        "token_type": tokens.get("token_type", "Bearer"),
        "expires_in": tokens.get("expires_in", 3600),
        "scopes": tokens["scope"].split(),
        "saved_at": datetime.now(UTC).isoformat(),
    }


def _refreshed(record: dict[str, Any]) -> dict[str, Any]:
    with httpx.Client(timeout=30) as client:
        response = client.post(
            TOKEN_URL,
            data={
                "grant_type": "refresh_token",
                "client_id": record["client_id"],
                "refresh_token": record["refresh_token"],
                "resource": RESOURCE,
            },
        )
    tokens = _token_response(response)
    granted_scopes = tokens.get("scope") or " ".join(record.get("scopes", []))
    if PLAN_SCOPE not in granted_scopes.split():
        raise RuntimeError("ChatGPT plan usage is no longer authorized; sign in again.")
    if not tokens.get("access_token") or not tokens.get("refresh_token"):
        raise RuntimeError("OpenAI OAuth refresh did not return replacement tokens.")
    return {
        **record,
        "access_token": tokens["access_token"],
        "refresh_token": tokens["refresh_token"],
        "id_token": tokens.get("id_token", record.get("id_token", "")),
        "token_type": tokens.get("token_type", "Bearer"),
        "expires_in": tokens.get("expires_in", 3600),
        "scopes": granted_scopes.split(),
        "saved_at": datetime.now(UTC).isoformat(),
    }


async def chatgpt_access_token() -> str:
    """Return a local access token, refreshing it near expiry without exposing it to Temporal."""
    path = credential_path()
    record = await asyncio.to_thread(_load, path)
    if time.time() < _expiry(record) - 120:
        return record["access_token"]
    async with _refresh_lock:
        record = await asyncio.to_thread(_load, path)
        if time.time() < _expiry(record) - 120:
            return record["access_token"]
        try:
            refreshed = await asyncio.to_thread(_refreshed, record)
        except RuntimeError as exc:
            if getattr(exc, "oauth_error", "") in TERMINAL_REFRESH_ERRORS:
                raise RuntimeError(
                    "ChatGPT authorization expired. Run `make action-author.login` again."
                ) from exc
            raise
        await asyncio.to_thread(_save, refreshed, path)
        return refreshed["access_token"]


def _authorization_url(
    client_id: str, host_id: str, redirect_uri: str, state: str, nonce: str, challenge: str
) -> str:
    parameters = {
        "client_id": client_id,
        "ext_agent_host_id": host_id,
        "response_type": "code",
        "redirect_uri": redirect_uri,
        "scope": SCOPES,
        "resource": RESOURCE,
        "state": state,
        "nonce": nonce,
        "code_challenge_method": "S256",
        "code_challenge": challenge,
    }
    if client_id == "dynamic_agent_client":
        parameters["agent_name_hint"] = "credimi-action-author"
    else:
        # Prefill the account without putting an identity token in the printed URL.
        previous = _load()
        if previous.get("email"):
            parameters["login_hint"] = previous["email"]
    return f"{AUTHORIZE_URL}?{urlencode(parameters)}"


def chatgpt_login() -> None:
    """Connect this local installation through OpenAI's browser-based public-client flow."""
    path = credential_path()
    try:
        previous = _load(path)
    except RuntimeError:
        previous = None
    client_id = previous["client_id"] if previous else "dynamic_agent_client"
    host_id = previous["ext_agent_host_id"] if previous else f"urn:uuid:{uuid4()}"
    state, nonce = secrets.token_urlsafe(32), secrets.token_urlsafe(32)
    verifier = secrets.token_urlsafe(64)
    challenge = (
        base64.urlsafe_b64encode(hashlib.sha256(verifier.encode()).digest()).rstrip(b"=").decode()
    )
    result: dict[str, str] = {}
    received = threading.Event()

    class Callback(BaseHTTPRequestHandler):
        def do_GET(self) -> None:  # noqa: N802 - stdlib handler API
            if urlparse(self.path).path != "/auth/callback":
                self.send_error(404)
                return
            result["query"] = urlparse(self.path).query
            self.send_response(200)
            self.send_header("Content-Type", "text/plain; charset=utf-8")
            self.end_headers()
            self.wfile.write(b"ChatGPT sign-in received. Return to the action-author terminal.")
            received.set()

        def log_message(self, format: str, *args: Any) -> None:  # noqa: A002
            pass

    server = ThreadingHTTPServer(("127.0.0.1", 0), Callback)
    port = server.server_address[1]
    redirect_uri = f"http://127.0.0.1:{port}/auth/callback"
    url = _authorization_url(client_id, host_id, redirect_uri, state, nonce, challenge)
    print("Opening ChatGPT sign-in. Authorize ChatGPT plan usage for credimi-action-author.")
    print("If no browser opens, use the printed local sign-in URL (it contains no tokens).")
    print(url)
    webbrowser.open(url)
    server.timeout = 1
    deadline = time.monotonic() + 300
    try:
        while not received.is_set():
            if time.monotonic() >= deadline:
                raise RuntimeError(
                    "ChatGPT sign-in timed out; run `make action-author.login` again."
                )
            server.handle_request()
        code, issued_client = _validate_callback(result["query"], state, client_id)
        with httpx.Client(timeout=30) as client:
            response = client.post(
                TOKEN_URL,
                data={
                    "grant_type": "authorization_code",
                    "code": code,
                    "client_id": issued_client,
                    "code_verifier": verifier,
                    "redirect_uri": redirect_uri,
                    "resource": RESOURCE,
                },
            )
        record = _credentials(_token_response(response), issued_client, host_id, nonce, previous)
        _save(record, path)
    finally:
        server.server_close()
    print(f"ChatGPT connected for {record['email'] or record['subject']}; credentials: {path}")


if __name__ == "__main__":
    chatgpt_login()
