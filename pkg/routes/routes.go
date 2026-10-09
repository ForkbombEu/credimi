// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package routes provides the routing and HTTP handling for the application.
// It includes functions to bind application hooks, register routes, and configure
// additional modules such as JavaScript VM and database migration commands.
// It also includes a reverse proxy for routing requests to different services.
package routes

import (
	"fmt"
	"log"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/forkbombeu/credimi/pkg/conformancecatalog"
	"github.com/forkbombeu/credimi/pkg/internal/apis"
	"github.com/forkbombeu/credimi/pkg/internal/apis/handlers"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/logo"
	"github.com/forkbombeu/credimi/pkg/internal/pb"
	pipelineresults "github.com/forkbombeu/credimi/pkg/internal/pipeline_results"
	"github.com/forkbombeu/credimi/pkg/internal/recordsecrets"
	"github.com/forkbombeu/credimi/pkg/internal/temporalui"
	walletversions "github.com/forkbombeu/credimi/pkg/internal/wallet_versions"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine/hooks"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/jsvm"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
)

func bindAppHooks(app core.App) {
	uiProxy := createReverseProxy(
		utils.GetEnvironmentVariable("ADDRESS_UI", "http://localhost:5100"),
	)
	temporalUITarget := utils.GetEnvironmentVariable(
		"ADDRESS_TEMPORAL_UI",
		"http://localhost:8281",
	)
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.Any("/{path...}", func(e *core.RequestEvent) error {
			// Unknown API paths answer 404 here: the Vite dev server proxies /api/
			// back to PocketBase, so forwarding them to the UI would loop.
			if p := e.Request.URL.Path; p == "/api" || strings.HasPrefix(p, "/api/") {
				return e.NotFoundError("", nil)
			}
			return uiProxy(e)
		})
		target, err := url.Parse(temporalUITarget)
		if err != nil {
			return fmt.Errorf("parse ADDRESS_TEMPORAL_UI: %w", err)
		}
		temporalUI := temporalui.Handler(target)
		se.Router.Any(temporalui.PathPrefix, temporalUI)
		se.Router.Any(temporalui.PathPrefix+"/{path...}", temporalUI)
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
	conformancecatalog.Register(app)
	hooks.TemporalClientSetupHook(app)
	pb.HookOrganizations(app)
	pb.RegisterMobileRunnerWorkerManagerHooks(app)
	pb.RegisterMobileRunnerHooks(app)
	pb.RegisterMobileDeviceHooks(app)
	pb.RegisterPipelineHooks(app)
	pb.RegisterWalletActionHooks(app)
	pb.RegisterSchedulesHooks(app)
	apis.RegisterMyRoutes(app)
	handlers.RegisterRealtimeLogsAuthorizationHook(app)
	hooks.WorkersHook(app)
	canonify.RegisterCanonifyHooks(app)
	apis.HookAtUserCreation(app)
	apis.HookAtUserLogin(app)
	apis.HookTurnstileVerification(app)
	logo.LogoHooks(app)
	walletversions.WalletVersionHooks(app)
	pipelineresults.RegisterPipelineResultsHooks(app)
	recordsecrets.RegisterHooks(app)
	// apis.IssuersRoutes.Add(app)

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

		proxy := &httputil.ReverseProxy{}
		proxy.Rewrite = func(req *httputil.ProxyRequest) {
			req.Out.URL.Scheme = targetURL.Scheme
			req.Out.URL.Host = targetURL.Host
			req.Out.Host = targetURL.Host
			req.Out.Header.Set("X-Forwarded-For", req.In.RemoteAddr)
			if origin := req.In.Header.Get("Origin"); origin != "" {
				req.Out.Header.Set("Origin", origin)
			}
			if referer := req.In.Header.Get("Referer"); referer != "" {
				req.Out.Header.Set("Referer", referer)
			}
		}
		proxy.ServeHTTP(r.Response, r.Request)
		return nil
	}
}
