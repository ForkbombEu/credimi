// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"net/http"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

type IdentifierValidateRequest struct {
	CanonifiedName string `json:"canonified_name"`
}

// IdentifierResolveInternalRequest asks for a record on behalf of the
// organization that owns a running workflow.
type IdentifierResolveInternalRequest struct {
	CanonifiedName string `json:"canonified_name"`
	Collection     string `json:"collection"`
	OwnerNamespace string `json:"owner_namespace"`
}

// canonifyInternalCollections lists the collections Temporal workers resolve
// through HandleIdentifierResolveInternal. Every one has an `owner` relation
// and a viewRule of `published = true || <member of owner organization>`.
var canonifyInternalCollections = map[string]struct{}{
	"pipelines":      {},
	"custom_checks":  {},
	"wallet_actions": {},
}

var CanonifyRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/canonify",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:        http.MethodPost,
			Path:          "/identifier/validate",
			Handler:       HandleIdentifierValidate,
			RequestSchema: IdentifierValidateRequest{},
			Description:   "Validate an entity identifier",
		},
		{
			Method:      http.MethodGet,
			Path:        "/identifier/get",
			Handler:     HandleGetIdentifier,
			Description: "Get the canonical identifier path of a record by id",
		},
	},
}

// CanonifyTemporalInternalRoutes lets Temporal workers read the records a
// pipeline references. Workers authenticate with the internal admin key but
// get only what the pipeline's owner organization could view, because the
// response lands in Temporal history, which the organization can read.
var CanonifyTemporalInternalRoutes routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api/canonify/internal",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
		middlewares.RequireInternalAdminAPIKey(),
	},
	Routes: []routing.RouteDefinition{
		{
			Method:        http.MethodPost,
			Path:          "/resolve",
			Handler:       HandleIdentifierResolveInternal,
			RequestSchema: IdentifierResolveInternalRequest{},
			Description:   "Resolve a record on behalf of the owner organization of a workflow",
		},
	},
}

func HandleIdentifierValidate() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		req, err := routing.GetValidatedInput[IdentifierValidateRequest](e)
		if err != nil {
			return err
		}
		record, err := canonify.Validate(e.App, req.CanonifiedName)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"identifier",
				"failed to validate identifier",
				err.Error(),
			)
		}
		if apiErr := requireRecordViewAccess(e, record); apiErr != nil {
			return apiErr
		}
		// Enrich hooks hide fields such as credential secrets from non-owners.
		if err := apis.EnrichRecord(e, record); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"identifier",
				"failed to enrich record",
				err.Error(),
			)
		}
		record.WithCustomData(true)
		record.Set("__canonified_path__", canonify.NormalizePath(req.CanonifiedName))
		return e.JSON(http.StatusOK, map[string]any{
			"message": "valid identifier",
			"record":  record,
		})
	}
}

// HandleIdentifierResolveInternal returns a published record of any
// organization, or an unpublished record of the owner organization.
func HandleIdentifierResolveInternal() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		req, err := routing.GetValidatedInput[IdentifierResolveInternalRequest](e)
		if err != nil {
			return err
		}
		collection := strings.TrimSpace(req.Collection)
		if _, ok := canonifyInternalCollections[collection]; !ok {
			return apierror.New(
				http.StatusBadRequest,
				"collection",
				"unsupported collection",
				collection,
			)
		}
		ownerNamespace := strings.TrimSpace(req.OwnerNamespace)
		if ownerNamespace == "" {
			return apierror.New(
				http.StatusBadRequest,
				"owner_namespace",
				"owner_namespace_required",
				"missing owner_namespace",
			)
		}

		notFound := apierror.New(
			http.StatusNotFound,
			"identifier",
			"record not found",
			"the requested record does not exist or the owner organization cannot view it",
		)
		record, err := canonify.Validate(e.App, req.CanonifiedName)
		if err != nil || record.Collection().Name != collection {
			return notFound
		}
		if !record.GetBool("published") {
			owner, err := e.App.FindFirstRecordByFilter(
				"organizations",
				"canonified_name = {:namespace}",
				map[string]any{"namespace": ownerNamespace},
			)
			if err != nil || record.GetString("owner") != owner.Id {
				return notFound
			}
		}

		record.WithCustomData(true)
		record.Set("__canonified_path__", canonify.NormalizePath(req.CanonifiedName))
		return e.JSON(http.StatusOK, map[string]any{
			"message": "valid identifier",
			"record":  record,
		})
	}
}

func HandleGetIdentifier() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		collection := e.Request.URL.Query().Get("collection")
		if collection == "" {
			return apierror.New(
				http.StatusBadRequest,
				"collection",
				"collection is required",
				"missing collection",
			)
		}

		recID := e.Request.URL.Query().Get("id")
		if recID == "" {
			return apierror.New(
				http.StatusBadRequest,
				"id",
				"record id is required",
				"missing id",
			)
		}

		if _, ok := canonify.CanonifyPaths[collection]; !ok && collection != "hub_items" {
			return apierror.New(
				http.StatusBadRequest,
				"collection",
				"no path template for collection",
				collection,
			)
		}

		rec, err := e.App.FindRecordById(collection, recID)
		if err != nil {
			return apierror.New(
				http.StatusNotFound,
				"id",
				"record not found",
				err.Error(),
			)
		}

		if collection == "hub_items" {
			colType := rec.GetString("type")
			collection = strings.Trim(colType, `"`)
			rec, err = e.App.FindRecordById(collection, recID)
			if err != nil {
				return apierror.New(
					http.StatusNotFound,
					"id",
					"record not found",
					err.Error(),
				)
			}
		}

		if apiErr := requireRecordViewAccess(e, rec); apiErr != nil {
			return apiErr
		}

		tpl, ok := canonify.CanonifyPaths[collection]
		if !ok {
			return apierror.New(
				http.StatusBadRequest,
				"collection",
				"no path template for collection",
				collection,
			)
		}

		identifier, err := canonify.BuildPath(e.App, rec, tpl, "")
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"path",
				"failed to build identifier path",
				err.Error(),
			)
		}

		return e.JSON(http.StatusOK, map[string]any{
			"collection": collection,
			"id":         recID,
			"identifier": identifier,
		})
	}
}

// requireRecordViewAccess applies the record collection's viewRule to the
// caller. Canonify resolves records with app-level queries, so without this
// check it would disclose records the native records API hides. Denials answer
// 404, like the native API.
func requireRecordViewAccess(e *core.RequestEvent, record *core.Record) *apierror.APIError {
	info, err := e.RequestInfo()
	if err != nil {
		return apierror.New(
			http.StatusInternalServerError,
			"request",
			"failed to read request info",
			err.Error(),
		)
	}
	canAccess, err := e.App.CanAccessRecord(record, info, record.Collection().ViewRule)
	if err != nil || !canAccess {
		return apierror.New(
			http.StatusNotFound,
			"identifier",
			"record not found",
			"the requested record does not exist or you cannot view it",
		)
	}
	return nil
}
