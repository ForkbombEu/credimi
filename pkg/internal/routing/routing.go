// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package routing

import (
	"fmt"
	"log"
	"net/http"
	"reflect"

	"github.com/forkbombeu/credimi/pkg/internal/apierror" // Adjust import path
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/pocketbase/pocketbase/tools/router"
)

type HandlerFunc func(e *core.RequestEvent) error

type HandlerFactory func() func(*core.RequestEvent) error

type RouteGroup struct {
	BaseURL                string
	Routes                 []RouteDefinition
	Middlewares            []*hook.Handler[*core.RequestEvent]
	AuthenticationRequired bool
}

type QuerySearchAttribute struct {
	Name        string `json:"name"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

type RouteDefinition struct {
	Method                string
	Path                  string
	Handler               HandlerFactory
	RequestSchema         any
	ResponseSchema        any
	Description           string
	Summary               string
	Examples              []string
	Middlewares           []*hook.Handler[*core.RequestEvent]
	ExcludedMiddlewares   []string
	QuerySearchAttributes []QuerySearchAttribute
}

func GetValidatedInput[T any](e *core.RequestEvent) (T, error) {
	validatedInput := e.Request.Context().Value(middlewares.ValidatedInputKey)
	var zero T

	if validatedInput == nil {
		return zero, nil
	}
	typedInput, ok := validatedInput.(T)
	if !ok {
		expectedType := fmt.Sprintf("%T", zero)
		actualType := fmt.Sprintf("%T", validatedInput)
		errMsg := fmt.Sprintf(
			"critical type mismatch for validated input: expected %s, got %s",
			expectedType,
			actualType,
		)
		return zero, apierror.New(
			http.StatusInternalServerError,
			"routing",
			"Input Type Mismatch",
			errMsg,
		).JSON(e)
	}
	return typedInput, nil
}

func (r *RouteGroup) Add(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		basePath := r.BaseURL
		if basePath == "" {
			basePath = "/api"
		}
		rg := se.Router.Group(basePath)
		rg.Bind(r.Middlewares...)
		RegisterRoutesWithValidation(app, rg, r.Routes, r.AuthenticationRequired)
		return se.Next()
	})
	app.OnServe()
}

func RegisterRoutesWithValidation(
	app core.App,
	group *router.RouterGroup[*core.RequestEvent],
	routes []RouteDefinition,
	needsAuth bool,
) {
	log.Println("Registering routes with validation")

	for _, route := range routes {
		inputType := reflect.TypeOf(route.RequestSchema)

		validatorMiddleware := middlewares.DynamicValidateInputByType(inputType)

		needsValidationBinding := inputType != nil

		if needsAuth {
			route.Middlewares = append(route.Middlewares, apis.RequireAuth())
		}

		switch route.Method {
		case http.MethodPost:
			registrar := group.POST(route.Path, route.Handler()).
				Bind(route.Middlewares...).
				Unbind(route.ExcludedMiddlewares...)
			if needsValidationBinding {
				registrar.Bind(&hook.Handler[*core.RequestEvent]{Func: validatorMiddleware})
			}
		case http.MethodGet:
			group.GET(route.Path, route.Handler()).
				Bind(route.Middlewares...).
				Unbind(route.ExcludedMiddlewares...)
		case http.MethodPut:
			registrar := group.PUT(route.Path, route.Handler()).
				Bind(route.Middlewares...).
				Unbind(route.ExcludedMiddlewares...)
			if needsValidationBinding {
				registrar.Bind(&hook.Handler[*core.RequestEvent]{Func: validatorMiddleware})
			}
		case http.MethodPatch:
			registrar := group.PATCH(route.Path, route.Handler()).
				Bind(route.Middlewares...).
				Unbind(route.ExcludedMiddlewares...)
			if needsValidationBinding {
				registrar.Bind(&hook.Handler[*core.RequestEvent]{Func: validatorMiddleware})
			}
		case http.MethodDelete:
			registrar := group.DELETE(route.Path, route.Handler()).
				Bind(route.Middlewares...).
				Unbind(route.ExcludedMiddlewares...)
			if needsValidationBinding {
				app.Logger().
					Warn("Binding validation middleware to DELETE route", "path", route.Path)
				registrar.Bind(&hook.Handler[*core.RequestEvent]{Func: validatorMiddleware})
			}
		case http.MethodHead:
			group.HEAD(route.Path, route.Handler()).
				Bind(route.Middlewares...).
				Unbind(route.ExcludedMiddlewares...)
		case http.MethodOptions:
			group.OPTIONS(route.Path, route.Handler()).
				Bind(route.Middlewares...).
				Unbind(route.ExcludedMiddlewares...)
		default:
			app.Logger().Warn("Unsupported HTTP method in route definition during registration",
				"method", route.Method,
				"path", route.Path,
			)
		}
	}
}

func RegisterRoutesWithoutValidation(
	app core.App,
	group *router.RouterGroup[*core.RequestEvent],
	routes []RouteDefinition,
) {
	for _, route := range routes {
		switch route.Method {
		case http.MethodPost:
			group.POST(route.Path, route.Handler()).
				Bind(route.Middlewares...).
				Unbind(route.ExcludedMiddlewares...)
		case http.MethodGet:
			group.GET(route.Path, route.Handler()).
				Bind(route.Middlewares...).
				Unbind(route.ExcludedMiddlewares...)
		case http.MethodPut:
			group.PUT(route.Path, route.Handler()).
				Bind(route.Middlewares...).
				Unbind(route.ExcludedMiddlewares...)
		case http.MethodPatch:
			group.PATCH(route.Path, route.Handler()).
				Bind(route.Middlewares...).
				Unbind(route.ExcludedMiddlewares...)
		case http.MethodDelete:
			group.DELETE(route.Path, route.Handler()).
				Bind(route.Middlewares...).
				Unbind(route.ExcludedMiddlewares...)
		default:
			app.Logger().Warn("Unsupported HTTP method in route definition during registration",
				"method", route.Method,
				"path", route.Path,
			)
		}
	}
}
