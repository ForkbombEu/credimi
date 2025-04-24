// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package routes provides the routing and HTTP handling for the application.
// It includes functions to bind application hooks, register routes, and configure
// additional modules such as JavaScript VM and database migration commands.
// It also includes a reverse proxy for routing requests to different services.
package routes

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/jsvm"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	"github.com/forkbombeu/didimo/pkg/internal/pb"
	"github.com/forkbombeu/didimo/pkg/utils"
	"github.com/forkbombeu/didimo/pkg/workflowengine/hooks"
)

func bindAppHooks(app core.App) {
	routes := map[string]string{
		"/{path...}": utils.GetEnvironmentVariable("ADDRESS_UI", "http://localhost:5100"),
	}
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		for path, target := range routes {
			se.Router.Any(path, createReverseProxy(target))
		}
		return se.Next()
	})
}

// Setup initializes the application by binding hooks, registering routes,
// and configuring additional modules. It sets up various functionalities
// such as application hooks, route handlers, worker hooks, JavaScript VM
// integration, and database migration commands.
//
// Parameters:
//   - app: A pointer to the PocketBase application instance.
//
// The function performs the following tasks:
//   - Binds application-specific hooks for handling events and workflows.
//   - Registers HTTP routes for handling specific API endpoints.
//   - Configures worker hooks for background task processing.
//   - Integrates a JavaScript VM for dynamic scripting capabilities.
//   - Registers and configures database migration commands with support
//     for JavaScript-based templates and automatic migration.
func Setup(app *pocketbase.PocketBase) {
	bindAppHooks(app)
	pb.RouteGetConfigsTemplates(app)
	pb.RoutePostPlaceholdersByFilenames(app)
	pb.HookNamespaceOrgs(app)
	pb.HookCredentialWorkflow(app)
	pb.AddOpenID4VPTestEndpoints(app)
	pb.HookUpdateCredentialsIssuers(app)
	pb.RouteWorkflow(app)
	pb.HookAtUserCreation(app)
	hooks.WorkersHook(app)

	jsvm.MustRegister(app, jsvm.Config{
		HooksWatch: true,
	})
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		TemplateLang: migratecmd.TemplateLangJS,
		Automigrate:  true,
	})
}

func createReverseProxy(target string) func(r *core.RequestEvent) error {
	return func(r *core.RequestEvent) error {
		targetURL, err := url.Parse(target)
		if err != nil {
			return err
		}
		if v := utils.GetEnvironmentVariable("DEBUG"); len(v) > 0 {
			log.Printf(
				"Proxying request: %s -> %s%s",
				r.Request.URL.Path,
				targetURL.String(),
				r.Request.URL.Path,
			)
		}

		proxy := httputil.NewSingleHostReverseProxy(targetURL)
		proxy.Director = func(req *http.Request) {
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.Header.Set("Host", targetURL.Host)
			req.Header.Set("X-Forwarded-For", req.RemoteAddr)
			if origin := req.Header.Get("Origin"); origin != "" {
				req.Header.Set("Origin", origin)
			}
			if referer := req.Header.Get("Referer"); referer != "" {
				req.Header.Set("Referer", referer)
			}
		}
		proxy.ServeHTTP(r.Response, r.Request)
		return nil
	}
}
