// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package temporalui serves the upstream Temporal UI under PathPrefix through a
// read-only reverse proxy scoped to the caller's organization namespace.
//
// The embedded UI runs with auth disabled and is reachable only through this proxy,
// so the proxy is the security boundary:
//   - the caller is a Credimi user authenticated by the `pb_auth` cookie the webapp
//     already keeps in sync with the PocketBase auth store;
//   - only GET and HEAD reach the UI, so no workflow can be started, signaled,
//     cancelled, terminated or reset from it;
//   - API calls must target the caller's organization namespace; the namespace
//     list is replaced by the caller's namespace alone.
//
// HTML pages get a stylesheet hiding the UI's own navigation shell, since Credimi
// pages embed only the workflow content.
package temporalui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

// PathPrefix is the public path of the embedded UI; it must match the UI's
// TEMPORAL_UI_PUBLIC_PATH.
const PathPrefix = "/temporal-ui"

// authCookieName is the cookie written by the webapp (pb.authStore.exportToCookie).
const authCookieName = "pb_auth"

type routeKind int

const (
	routeForbidden routeKind = iota
	// routeProxy forwards unchanged: API calls and build assets.
	routeProxy
	// routePage forwards a client-side page; HTML documents get embedStyle.
	routePage
	routeNamespaceList
	routeHome
)

// unscopedAPIEndpoints hold server metadata only, no tenant data.
var unscopedAPIEndpoints = map[string]bool{
	"settings":     true,
	"cluster-info": true,
	"system-info":  true,
}

// classify decides how a request for escapedPath (below PathPrefix) is served for
// a caller whose organization namespace is namespace.
func classify(escapedPath, namespace string) routeKind {
	rest := strings.TrimPrefix(escapedPath, PathPrefix)
	if rest == "" || rest == "/" {
		return routeHome
	}
	if !strings.HasPrefix(rest, "/") {
		return routeForbidden
	}

	rawSegments := strings.Split(rest[1:], "/")
	segments := make([]string, len(rawSegments))
	for i, raw := range rawSegments {
		segment, err := url.PathUnescape(raw)
		if err != nil || segment == "." || segment == ".." {
			return routeForbidden
		}
		segments[i] = segment
	}

	switch segments[0] {
	case "api":
		return classifyAPI(segments, namespace)
	case "auth":
		return routeForbidden
	case "namespaces":
		if len(segments) > 1 && segments[1] != namespace {
			return routeHome
		}
		return routePage
	case "_app":
		return routeProxy
	default:
		// Other client-side pages and static files; their data comes from the API.
		return routePage
	}
}

func classifyAPI(segments []string, namespace string) routeKind {
	if len(segments) < 3 || segments[1] != "v1" {
		return routeForbidden
	}
	if segments[2] != "namespaces" {
		if len(segments) == 3 && unscopedAPIEndpoints[segments[2]] {
			return routeProxy
		}
		return routeForbidden
	}
	if len(segments) == 3 {
		return routeNamespaceList
	}
	if segments[3] == namespace {
		return routeProxy
	}
	return routeForbidden
}

func homePath(namespace string) string {
	return PathPrefix + "/namespaces/" + url.PathEscape(namespace) + "/workflows"
}

// cookieAuthToken extracts the PocketBase token from the `pb_auth` cookie, whose
// value is the URI-encoded JSON `{"token": "...", "record": {...}}`.
func cookieAuthToken(r *http.Request) string {
	cookie, err := r.Cookie(authCookieName)
	if err != nil {
		return ""
	}
	raw, err := url.PathUnescape(cookie.Value)
	if err != nil {
		return ""
	}
	var payload struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return ""
	}
	return payload.Token
}

// callerNamespace returns the organization namespace of the authenticated user.
func callerNamespace(e *core.RequestEvent) (string, error) {
	user := e.Auth
	if user == nil {
		token := cookieAuthToken(e.Request)
		if token == "" {
			return "", e.UnauthorizedError("Sign in to Credimi to see workflow details.", nil)
		}
		record, err := e.App.FindAuthRecordByToken(token, core.TokenTypeAuth)
		if err != nil {
			return "", e.UnauthorizedError("Your session expired, sign in again.", err)
		}
		user = record
	}
	if user.Collection().Name != "users" {
		return "", e.ForbiddenError("Only organization members can see workflows.", nil)
	}
	namespace, err := pbutils.GetUserOrganizationCanonifiedName(e.App, user.Id)
	if err != nil || namespace == "" {
		return "", e.ForbiddenError("You are not a member of any organization.", err)
	}
	return namespace, nil
}

type requestScope struct {
	namespace string
	kind      routeKind
}

type scopeKey struct{}

// Handler proxies PathPrefix requests to the Temporal UI at target.
func Handler(target *url.URL) func(*core.RequestEvent) error {
	proxy := newProxy(target)
	return func(e *core.RequestEvent) error {
		method := e.Request.Method
		if method != http.MethodGet && method != http.MethodHead {
			return router.NewApiError(
				http.StatusMethodNotAllowed,
				"The embedded Temporal UI is read-only.",
				nil,
			)
		}
		namespace, err := callerNamespace(e)
		if err != nil {
			return err
		}
		kind := classify(e.Request.URL.EscapedPath(), namespace)
		switch kind {
		case routeForbidden:
			return e.ForbiddenError("This Temporal UI resource is not available.", nil)
		case routeHome:
			return e.Redirect(http.StatusFound, homePath(namespace))
		case routeProxy, routePage, routeNamespaceList:
		}
		ctx := context.WithValue(
			e.Request.Context(),
			scopeKey{},
			requestScope{namespace: namespace, kind: kind},
		)
		proxy.ServeHTTP(e.Response, e.Request.WithContext(ctx))
		return nil
	}
}

func newProxy(target *url.URL) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.SetXForwarded()
			pr.Out.Header.Del("Authorization")
			stripCookie(pr.Out, authCookieName)

			scope, _ := pr.In.Context().Value(scopeKey{}).(requestScope)
			switch scope.kind {
			case routeNamespaceList:
				// Describe the caller's namespace instead of listing all of them.
				pr.Out.URL.Path = PathPrefix + "/api/v1/namespaces/" + scope.namespace
				pr.Out.URL.RawPath = PathPrefix + "/api/v1/namespaces/" +
					url.PathEscape(scope.namespace)
				pr.Out.URL.RawQuery = ""
				// Let the transport negotiate and decode compression so the body
				// can be rewritten.
				pr.Out.Header.Del("Accept-Encoding")
			case routePage:
				pr.Out.Header.Del("Accept-Encoding")
			case routeForbidden, routeProxy, routeHome:
			}
		},
		ModifyResponse: func(resp *http.Response) error {
			scope, _ := resp.Request.Context().Value(scopeKey{}).(requestScope)
			if resp.StatusCode != http.StatusOK {
				return nil
			}
			switch scope.kind {
			case routeNamespaceList:
				return wrapNamespaceList(resp)
			case routePage:
				return injectEmbedStyle(resp)
			case routeForbidden, routeProxy, routeHome:
			}
			return nil
		},
	}
}

// embedStyle hides the Temporal UI shell (side and top navigation): Credimi pages
// embed only the workflow content and own the navigation.
const embedStyle = `<style id="credimi-embed">` +
	`div:has(> nav[data-testid="navigation-header"]),nav[data-testid="top-nav"]` +
	`{display:none !important}</style>`

func injectEmbedStyle(resp *http.Response) error {
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return fmt.Errorf("read Temporal UI page: %w", err)
	}
	body = bytes.Replace(body, []byte("</head>"), []byte(embedStyle+"</head>"), 1)
	setBody(resp, body)
	return nil
}

// wrapNamespaceList turns a DescribeNamespace response into a one-entry
// ListNamespaces response; list entries have the DescribeNamespace shape.
func wrapNamespaceList(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return fmt.Errorf("read namespace description: %w", err)
	}
	wrapped, err := json.Marshal(struct {
		Namespaces    []json.RawMessage `json:"namespaces"`
		NextPageToken string            `json:"nextPageToken"`
	}{Namespaces: []json.RawMessage{body}})
	if err != nil {
		return fmt.Errorf("wrap namespace description: %w", err)
	}
	setBody(resp, wrapped)
	return nil
}

func setBody(resp *http.Response, body []byte) {
	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
	resp.Header.Del("Content-Encoding")
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
