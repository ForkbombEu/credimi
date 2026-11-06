// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package actionauthorui serves the action author chat UI (agents/action-author-ui)
// under PathPrefix/<session id>/ through a reverse proxy scoped to one session.
//
// The UI service has no authentication of its own and is reachable only through
// this proxy, so the proxy is the security boundary:
//   - the caller is a Credimi user authenticated by the `pb_auth` cookie
//     (temporalui.Caller), and the session must be one Credimi recorded
//     in the caller's organization namespace;
//   - only the user who started the session sends state-changing requests;
//     other members of the organization can read the chat;
//   - the service learns the namespace and the session's chat workflow only from
//     NamespaceHeader and ChatHeader, which the proxy always overwrites;
//   - only the routes the embedded chat needs reach the service, and every route
//     naming a workflow must name the session's chat;
//   - state-changing requests must come from Credimi's own origin.
package actionauthorui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/temporalui"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

// PathPrefix is the public path of the embedded chat UI.
const PathPrefix = "/action-author-ui"

const (
	// NamespaceHeader carries the caller's organization namespace to the service.
	NamespaceHeader = "X-Credimi-Namespace"
	// ChatHeader carries the session's chat workflow ID to the service.
	ChatHeader = "X-Credimi-Action-Author-Chat"
)

// maxBody bounds the JSON bodies the proxy reads to check their session.
const maxBody = 1 << 20

const authCookieName = "pb_auth"

const sessionMissing = "This authoring session does not exist."

type route int

const (
	routeForbidden route = iota
	routeProxy
	// routeBody is proxied once its JSON body names the chat as session_id.
	routeBody
)

var staticFile = regexp.MustCompile(`^[A-Za-z0-9._-]+\.(svg|css)$`)

// chatRoutes are the GET API routes whose last segment is a workflow ID.
var chatRoutes = map[string]bool{
	"workflow-status": true,
	"status":          true,
	"agent-interface": true,
}

// bodyRoutes are the POST API routes whose JSON body names the session.
var bodyRoutes = map[string]bool{
	"messages": true,
	"chat":     true,
	"approve":  true,
}

// classify decides how a request for the path segments below the session is
// served. Anything not listed is refused, among them the harness's own session
// start (POST /api/sessions), callback results (reviews go through Credimi) and
// file downloads.
func classify(method string, segments []string, query url.Values, chat string) route {
	read := method == http.MethodGet || method == http.MethodHead
	switch {
	case read && len(segments) == 1 && segments[0] == "":
		return routeProxy
	case read && len(segments) == 2 && segments[0] == "assets" && segments[1] != "":
		return routeProxy
	case read && len(segments) == 1 && staticFile.MatchString(segments[0]):
		return routeProxy
	}
	if len(segments) < 2 || segments[0] != "api" {
		return routeForbidden
	}
	api := segments[1:]
	switch method {
	case http.MethodGet:
		return classifyAPIRead(api, query, chat)
	case http.MethodPost:
		switch {
		case len(api) == 3 && api[0] == "sessions" && api[1] == chat && api[2] == "close":
			return routeProxy
		case len(api) == 1 && bodyRoutes[api[0]]:
			return routeBody
		}
	}
	return routeForbidden
}

func classifyAPIRead(api []string, query url.Values, chat string) route {
	switch {
	case len(api) == 1 && (api[0] == "agents" || api[0] == "sessions"):
		return routeProxy
	case len(api) == 2 && chatRoutes[api[0]] && api[1] == chat:
		return routeProxy
	case len(api) == 1 && api[0] == "attach":
		if ids := query["session_id"]; len(ids) == 1 && ids[0] == chat {
			return routeProxy
		}
	}
	return routeForbidden
}

// splitPath splits escapedPath into the session ID, the unescaped segments
// below it and their escaped form. A path ending at the ID returns errRedirect.
func splitPath(
	escapedPath string,
) (sessionID string, segments []string, rawRest string, err error) {
	trimmed, ok := strings.CutPrefix(escapedPath, PathPrefix+"/")
	if !ok {
		return "", nil, "", errForbidden
	}
	sessionID, rawRest, hasRest := strings.Cut(trimmed, "/")
	if !actionauthor.IsSessionID(sessionID) {
		return "", nil, "", errNotFound
	}
	if !hasRest {
		return sessionID, nil, "", errRedirect
	}
	for _, raw := range strings.Split(rawRest, "/") {
		segment, unescapeErr := url.PathUnescape(raw)
		// An escaped slash would change the route the service decodes.
		if unescapeErr != nil || segment == "." || segment == ".." ||
			strings.Contains(segment, "/") {
			return "", nil, "", errForbidden
		}
		segments = append(segments, segment)
	}
	return sessionID, segments, "/" + rawRest, nil
}

var (
	errForbidden = errors.New("forbidden path")
	errNotFound  = errors.New("unknown session")
	errRedirect  = errors.New("missing trailing slash")
)

// sameOrigin reports whether the request's Origin is Credimi's own host.
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	parsed, err := url.Parse(origin)
	return err == nil && parsed.Host != "" && parsed.Host == r.Host
}

type forward struct {
	namespace, chat, path, rawPath string
}

type forwardKey struct{}

// Handler proxies PathPrefix requests to the chat UI service at target.
func Handler(target *url.URL) func(*core.RequestEvent) error {
	proxy := newProxy(target)
	return func(e *core.RequestEvent) error {
		sessionID, segments, rawRest, err := splitPath(e.Request.URL.EscapedPath())
		switch {
		case errors.Is(err, errRedirect):
			location := PathPrefix + "/" + sessionID + "/"
			if e.Request.URL.RawQuery != "" {
				location += "?" + e.Request.URL.RawQuery
			}
			return e.Redirect(http.StatusPermanentRedirect, location)
		case errors.Is(err, errNotFound):
			return e.NotFoundError(sessionMissing, nil)
		case err != nil:
			return e.ForbiddenError("This chat resource is not available.", nil)
		}

		user, namespace, err := temporalui.Caller(e)
		if err != nil {
			return err
		}
		recorded, err := actionauthor.FindSession(e.App, namespace, sessionID)
		if err != nil {
			if errors.Is(err, actionauthor.ErrSessionNotRecorded) {
				return e.NotFoundError(sessionMissing, nil)
			}
			return e.InternalServerError("", err)
		}
		chat := sessionID + activities.ActionAuthorWorkflowIDSuffix

		method := e.Request.Method
		if method != http.MethodGet && method != http.MethodHead && !sameOrigin(e.Request) {
			return e.ForbiddenError("Cross-site requests are refused.", nil)
		}
		// Sessions recorded without an actor have no owner and are read-only.
		if method != http.MethodGet && method != http.MethodHead && recorded.ActorID != user.Id {
			return e.ForbiddenError(
				"Only the person who started this session can write in its chat.",
				nil,
			)
		}
		switch classify(method, segments, e.Request.URL.Query(), chat) {
		case routeForbidden:
			return e.ForbiddenError("This chat resource is not available.", nil)
		case routeBody:
			if err := requireBodySession(e, chat); err != nil {
				return err
			}
		case routeProxy:
		}

		scoped := e.Request.WithContext(
			context.WithValue(e.Request.Context(), forwardKey{}, forward{
				namespace: namespace,
				chat:      chat,
				path:      "/" + strings.Join(segments, "/"),
				rawPath:   rawRest,
			}),
		)
		proxy.ServeHTTP(e.Response, scoped)
		return nil
	}
}

// requireBodySession buffers the JSON body and refuses it unless its
// session_id is chat; the buffered body is then forwarded unchanged.
func requireBodySession(e *core.RequestEvent, chat string) error {
	body, err := io.ReadAll(io.LimitReader(e.Request.Body, maxBody+1))
	if err != nil {
		return e.BadRequestError("Unreadable request body.", err)
	}
	if len(body) > maxBody {
		return router.NewApiError(http.StatusRequestEntityTooLarge, "Request body too large.", nil)
	}
	var payload struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.SessionID != chat {
		return e.ForbiddenError("This chat resource is not available.", nil)
	}
	e.Request.Body = io.NopCloser(bytes.NewReader(body))
	e.Request.ContentLength = int64(len(body))
	e.Request.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	return nil
}

func newProxy(target *url.URL) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			scope, _ := pr.In.Context().Value(forwardKey{}).(forward)
			pr.SetURL(target)
			pr.SetXForwarded()
			pr.Out.URL.Path = scope.path
			pr.Out.URL.RawPath = scope.rawPath
			pr.Out.Header.Del("Authorization")
			stripCookie(pr.Out, authCookieName)
			pr.Out.Header.Del(NamespaceHeader)
			pr.Out.Header.Del(ChatHeader)
			pr.Out.Header.Set(NamespaceHeader, scope.namespace)
			pr.Out.Header.Set(ChatHeader, scope.chat)
		},
		// Stream server-sent events (/api/attach, /api/chat) as they arrive.
		FlushInterval: -1,
	}
}

func stripCookie(r *http.Request, name string) {
	cookies := r.Cookies()
	r.Header.Del("Cookie")
	for _, cookie := range cookies {
		if cookie.Name != name {
			r.AddCookie(cookie)
		}
	}
}
