// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"net/http"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

var PipelineDetails routing.RouteGroup = routing.RouteGroup{
	BaseURL:                "/api",
	AuthenticationRequired: true,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:  http.MethodGet,
			Path:    "/pipeline/details",
			Handler: HandleGetPipelineDetails,
		},
	},
}

func HandleGetPipelineDetails() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		if authRecord == nil {
			return apierror.New(
				http.StatusUnauthorized,
				"auth",
				"authentication required",
				"user not authenticated",
			).JSON(e)
		}

		organization, err := GetUserOrganizationID(e.App, authRecord.Id)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"organization",
				"failed to get user organization",
				err.Error(),
			).JSON(e)
		}

		pipelineRecords, err := e.App.FindRecordsByFilter(
			"pipelines",
			"owner={:owner} || published={:published}",
			"",
			-1,
			0,
			dbx.Params{
				"owner":     organization,
				"published": true,
			},
		)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"pipelines",
				"failed to fetch pipelines",
				err.Error(),
			).JSON(e)
		}

		if len(pipelineRecords) == 0 {
			return e.JSON(http.StatusOK, map[string]interface{}{
				"pipelines": []interface{}{},
				"count":     0,
			})
		}

		var responsePipelines []map[string]interface{}

		for _, pipelineRecord := range pipelineRecords {
			pipelineID := pipelineRecord.Id

			resultsRecords, err := e.App.FindRecordsByFilter(
				"pipeline_results",
				"pipeline={:pipeline}",
				"",
				-1,
				0,
				dbx.Params{"pipeline": pipelineID},
			)

			var results []map[string]interface{}
			if err == nil {
				for _, resultRecord := range resultsRecords {
					results = append(results, map[string]interface{}{
						"workflow_id": resultRecord.GetString("workflow_id"),
						"run_id":      resultRecord.GetString("run_id"),
					})
				}
			}

			pipeline := map[string]interface{}{
				"id":      pipelineID,
				"name":    pipelineRecord.GetString("name"),
				"owner":   pipelineRecord.GetString("owner"),
				"results": results,
			}

			responsePipelines = append(responsePipelines, pipeline)
		}

		return e.JSON(http.StatusOK, map[string]interface{}{
			"pipelines": responsePipelines,
		})
	}
}
