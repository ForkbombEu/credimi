// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
)

const validPipelineYAML = `
name: test-pipeline
steps:
  - id: step-1
    use: http-request
    with:
      payload:
        url: "https://example.com"
        method: GET
`

const invalidStepYAML = `
name: test-pipeline
steps:
  - id: step-1
    use: mobile-automation
    with:
      payload:
        url: "https://example.com"
`

const multiStepYAML = `
name: test-pipeline
steps:
  - id: step-1
    use: http-request
    with:
      payload:
        url: "https://example.com/first"
  - id: step-2
    use: http-request
    with:
      payload:
        url: "https://example.com/second"
`

const mixedStepsYAML = `
name: test-pipeline
steps:
  - id: step-1
    use: http-request
    with:
      payload:
        url: "https://example.com"
  - id: step-2
    use: mobile-automation
    with:
      payload:
        runner_id: runner-1
`

func setupPipelineExecuteApp(t testing.TB) *tests.TestApp {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	PipelineTemporalInternalRoutes.Add(app)
	return app
}

func mockTemporalClient(
	t *testing.T,
	result workflowengine.WorkflowResult,
	getErr error,
) {
	t.Helper()

	orig := pipelineTemporalClient
	t.Cleanup(func() { pipelineTemporalClient = orig })

	mockClient := temporalmocks.NewClient(t)
	workflowRun := temporalmocks.NewWorkflowRun(t)

	workflowRun.On("GetID").Return("wf-test-123").Maybe()
	workflowRun.On("GetRunID").Return("run-test-456").Maybe()
	workflowRun.On("Get", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			if out, ok := args.Get(1).(*workflowengine.WorkflowResult); ok {
				*out = result
			}
		}).
		Return(getErr)

	mockClient.On(
		"ExecuteWorkflow",
		mock.Anything,
		mock.Anything,
		"Dynamic Pipeline Workflow",
		mock.Anything,
	).Return(workflowRun, nil)

	pipelineTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}
}

func TestHandlePipelineExecute_EmptyBody(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:           "empty body returns 400",
		Method:         http.MethodPost,
		URL:            "/api/pipeline/execute",
		Body:           nil,
		ExpectedStatus: http.StatusBadRequest,
		ExpectedContent: []string{
			"body",
			"request body cannot be empty",
		},
		TestAppFactory: setupPipelineExecuteApp,
	}
	scenario.Test(t)
}

func TestHandlePipelineExecute_InvalidYAML(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:   "invalid YAML returns 400",
		Method: http.MethodPost,
		URL:    "/api/pipeline/execute",
		Body:   rawBody("this: is: not: valid: yaml:::"),
		ExpectedContent: []string{
			"yaml",
			"failed to parse pipeline YAML",
		},
		ExpectedStatus: http.StatusBadRequest,
		TestAppFactory: setupPipelineExecuteApp,
	}
	scenario.Test(t)
}

func TestHandlePipelineExecute_InvalidStepType(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:   "non http-request step returns 400",
		Method: http.MethodPost,
		URL:    "/api/pipeline/execute",
		Body:   rawBody(invalidStepYAML),
		ExpectedContent: []string{
			"yaml",
			"mobile-automation",
			"Only 'http-request' steps are allowed",
		},
		ExpectedStatus: http.StatusBadRequest,
		TestAppFactory: setupPipelineExecuteApp,
	}
	scenario.Test(t)
}

func TestHandlePipelineExecute_MixedStepTypes(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:   "mixed step types returns 400",
		Method: http.MethodPost,
		URL:    "/api/pipeline/execute",
		Body:   rawBody(mixedStepsYAML),
		ExpectedContent: []string{
			"yaml",
			"mobile-automation",
		},
		ExpectedStatus: http.StatusBadRequest,
		TestAppFactory: setupPipelineExecuteApp,
	}
	scenario.Test(t)
}

func TestHandlePipelineExecute_RedirectWithoutDeeplink(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:   "redirect=true without deeplink=true returns 400",
		Method: http.MethodPost,
		URL:    "/api/pipeline/execute?redirect=true",
		Body:   rawBody(validPipelineYAML),
		ExpectedContent: []string{
			"redirect",
			"redirect=true requires deeplink=true",
		},
		ExpectedStatus: http.StatusBadRequest,
		TestAppFactory: setupPipelineExecuteApp,
	}
	scenario.Test(t)
}

func TestHandlePipelineExecute_Success(t *testing.T) {
	mockTemporalClient(t, workflowengine.WorkflowResult{
		WorkflowID:    "wf-test-123",
		WorkflowRunID: "run-test-456",
		Output: map[string]any{
			"step-1": map[string]any{
				"outputs": map[string]any{
					"status": 200,
					"body":   "ok",
				},
			},
		},
	}, nil)

	scenario := tests.ApiScenario{
		Name:   "valid pipeline returns 200 with result",
		Method: http.MethodPost,
		URL:    "/api/pipeline/execute",
		Body:   rawBody(validPipelineYAML),
		ExpectedContent: []string{
			"\"workflow_id\"",
			"\"run_id\"",
			"\"result\"",
		},
		ExpectedStatus: http.StatusOK,
		TestAppFactory: setupPipelineExecuteApp,
	}
	scenario.Test(t)
}

func TestHandlePipelineExecute_SuccessMultiStep(t *testing.T) {
	mockTemporalClient(t, workflowengine.WorkflowResult{
		WorkflowID:    "wf-test-123",
		WorkflowRunID: "run-test-456",
		Output: map[string]any{
			"step-1": map[string]any{
				"outputs": map[string]any{"body": "first"},
			},
			"step-2": map[string]any{
				"outputs": map[string]any{"body": "second"},
			},
		},
	}, nil)

	scenario := tests.ApiScenario{
		Name:   "multi-step pipeline returns all step outputs",
		Method: http.MethodPost,
		URL:    "/api/pipeline/execute",
		Body:   rawBody(multiStepYAML),
		ExpectedContent: []string{
			"\"result\"",
			"step-1",
			"step-2",
		},
		ExpectedStatus: http.StatusOK,
		TestAppFactory: setupPipelineExecuteApp,
	}
	scenario.Test(t)
}

func TestHandlePipelineExecute_WorkflowStartFails(t *testing.T) {
	orig := pipelineTemporalClient
	t.Cleanup(func() { pipelineTemporalClient = orig })

	mockClient := temporalmocks.NewClient(t)
	mockClient.On(
		"ExecuteWorkflow",
		mock.Anything,
		mock.Anything,
		"Dynamic Pipeline Workflow",
		mock.Anything,
	).Return(nil, fmt.Errorf("grpc: the client connection is closing"))

	pipelineTemporalClient = func(_ string) (client.Client, error) {
		return mockClient, nil
	}

	scenario := tests.ApiScenario{
		Name:   "workflow start failure returns 500",
		Method: http.MethodPost,
		URL:    "/api/pipeline/execute",
		Body:   rawBody(validPipelineYAML),
		ExpectedContent: []string{
			"workflow",
			"failed to start workflow",
		},
		ExpectedStatus: http.StatusInternalServerError,
		TestAppFactory: setupPipelineExecuteApp,
	}
	scenario.Test(t)
}

func TestHandlePipelineExecute_WorkflowGetFails(t *testing.T) {
	mockTemporalClient(t, workflowengine.WorkflowResult{}, errWorkflowFailed())

	scenario := tests.ApiScenario{
		Name:   "workflow get failure returns 500",
		Method: http.MethodPost,
		URL:    "/api/pipeline/execute",
		Body:   rawBody(validPipelineYAML),
		ExpectedContent: []string{
			"workflow",
			"workflow execution failed",
		},
		ExpectedStatus: http.StatusInternalServerError,
		TestAppFactory: setupPipelineExecuteApp,
	}
	scenario.Test(t)
}

// --- Deeplink tests ---

func TestHandlePipelineExecute_DeeplinkPresent(t *testing.T) {
	deeplink := "openid-credential-offer://?credential_offer=abc123"

	mockTemporalClient(t, workflowengine.WorkflowResult{
		Output: map[string]any{
			"step-1": map[string]any{
				"outputs": map[string]any{
					"deeplink": deeplink,
				},
			},
		},
	}, nil)

	scenario := tests.ApiScenario{
		Name:   "deeplink=true returns deeplink string",
		Method: http.MethodPost,
		URL:    "/api/pipeline/execute?deeplink=true",
		Body:   rawBody(validPipelineYAML),
		ExpectedContent: []string{
			deeplink,
		},
		NotExpectedContent: []string{
			"\"workflow_id\"",
			"\"result\"",
		},
		ExpectedStatus: http.StatusOK,
		TestAppFactory: setupPipelineExecuteApp,
	}
	scenario.Test(t)
}

func TestHandlePipelineExecute_DeeplinkMissing(t *testing.T) {
	mockTemporalClient(t, workflowengine.WorkflowResult{
		Output: map[string]any{
			"step-1": map[string]any{
				"outputs": map[string]any{
					"body": "no deeplink here",
				},
			},
		},
	}, nil)

	scenario := tests.ApiScenario{
		Name:   "deeplink=true but no deeplink in output returns 404",
		Method: http.MethodPost,
		URL:    "/api/pipeline/execute?deeplink=true",
		Body:   rawBody(validPipelineYAML),
		ExpectedContent: []string{
			"deeplink",
			"deeplink not found",
		},
		ExpectedStatus: http.StatusNotFound,
		TestAppFactory: setupPipelineExecuteApp,
	}
	scenario.Test(t)
}

func TestHandlePipelineExecute_DeeplinkOutputNotMap(t *testing.T) {
	mockTemporalClient(t, workflowengine.WorkflowResult{
		Output: "not a map",
	}, nil)

	scenario := tests.ApiScenario{
		Name:   "deeplink=true but output is not a map returns 404",
		Method: http.MethodPost,
		URL:    "/api/pipeline/execute?deeplink=true",
		Body:   rawBody(validPipelineYAML),
		ExpectedContent: []string{
			"deeplink",
		},
		ExpectedStatus: http.StatusNotFound,
		TestAppFactory: setupPipelineExecuteApp,
	}
	scenario.Test(t)
}

func TestHandlePipelineExecute_RedirectWithDeeplink(t *testing.T) {
	deeplink := "openid-credential-offer://?credential_offer=abc123"

	mockTemporalClient(t, workflowengine.WorkflowResult{
		Output: map[string]any{
			"step-1": map[string]any{
				"outputs": map[string]any{
					"deeplink": deeplink,
				},
			},
		},
	}, nil)

	scenario := tests.ApiScenario{
		Name:           "redirect=true with deeplink=true returns 302",
		Method:         http.MethodPost,
		URL:            "/api/pipeline/execute?deeplink=true&redirect=true",
		Body:           rawBody(validPipelineYAML),
		ExpectedStatus: http.StatusFound,
		TestAppFactory: setupPipelineExecuteApp,
	}
	scenario.Test(t)
}

func TestHandlePipelineExecute_WithAuthUsesOrgNamespace(t *testing.T) {
	userRecord, err := getUserRecordFromName("userA")
	require.NoError(t, err)
	token, err := userRecord.NewAuthToken()
	require.NoError(t, err)

	var capturedNamespace string
	orig := pipelineTemporalClient
	t.Cleanup(func() { pipelineTemporalClient = orig })

	mockClient := temporalmocks.NewClient(t)
	workflowRun := temporalmocks.NewWorkflowRun(t)
	workflowRun.On("GetID").Return("wf-test-123").Maybe()
	workflowRun.On("GetRunID").Return("run-test-456").Maybe()
	workflowRun.On("Get", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			if out, ok := args.Get(1).(*workflowengine.WorkflowResult); ok {
				*out = workflowengine.WorkflowResult{
					WorkflowID:    "wf-test-123",
					WorkflowRunID: "run-test-456",
					Output:        map[string]any{},
				}
			}
		}).
		Return(nil)
	mockClient.On(
		"ExecuteWorkflow",
		mock.Anything,
		mock.Anything,
		"Dynamic Pipeline Workflow",
		mock.Anything,
	).Return(workflowRun, nil)

	pipelineTemporalClient = func(namespace string) (client.Client, error) {
		capturedNamespace = namespace
		return mockClient, nil
	}

	scenario := tests.ApiScenario{
		Name:   "authenticated user uses org namespace",
		Method: http.MethodPost,
		URL:    "/api/pipeline/execute",
		Headers: map[string]string{
			"Authorization": "Bearer " + token,
		},
		Body:           rawBody(validPipelineYAML),
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			"\"workflow_id\"",
			"\"result\"",
		},

		TestAppFactory: setupPipelineExecuteApp,
	}
	scenario.Test(t)

	require.NotEqual(t, "default", capturedNamespace, "authenticated user should use org namespace, not default")
}

func TestHandlePipelineExecute_WithoutAuthUsesDefaultNamespace(t *testing.T) {
	var capturedNamespace string
	orig := pipelineTemporalClient
	t.Cleanup(func() { pipelineTemporalClient = orig })

	mockClient := temporalmocks.NewClient(t)
	workflowRun := temporalmocks.NewWorkflowRun(t)
	workflowRun.On("GetID").Return("wf-test-123").Maybe()
	workflowRun.On("GetRunID").Return("run-test-456").Maybe()
	workflowRun.On("Get", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			if out, ok := args.Get(1).(*workflowengine.WorkflowResult); ok {
				*out = workflowengine.WorkflowResult{
					WorkflowID:    "wf-test-123",
					WorkflowRunID: "run-test-456",
					Output:        map[string]any{},
				}
			}
		}).
		Return(nil)
	mockClient.On(
		"ExecuteWorkflow",
		mock.Anything,
		mock.Anything,
		"Dynamic Pipeline Workflow",
		mock.Anything,
	).Return(workflowRun, nil)

	pipelineTemporalClient = func(namespace string) (client.Client, error) {
		capturedNamespace = namespace
		return mockClient, nil
	}

	scenario := tests.ApiScenario{
		Name:           "unauthenticated request uses default namespace",
		Method:         http.MethodPost,
		URL:            "/api/pipeline/execute",
		Body:           rawBody(validPipelineYAML),
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			"\"workflow_id\"",
			"\"result\"",
		},
		TestAppFactory: setupPipelineExecuteApp,
	}
	scenario.Test(t)

	require.Equal(t, "default", capturedNamespace)
}

func TestExtractDeeplink(t *testing.T) {
	t.Run("returns deeplink from last step", func(t *testing.T) {
		output := map[string]any{
			"step-1": map[string]any{
				"outputs": map[string]any{
					"deeplink": "openid://abc",
				},
			},
		}
		deeplink, err := extractDeeplink(output)
		require.NoError(t, err)
		require.Equal(t, "openid://abc", deeplink)
	})

	t.Run("returns error when output is not a map", func(t *testing.T) {
		_, err := extractDeeplink("not a map")
		require.Error(t, err)
		require.Contains(t, err.Error(), "output is not a map")
	})

	t.Run("returns error when deeplink key missing", func(t *testing.T) {
		output := map[string]any{
			"step-1": map[string]any{
				"outputs": map[string]any{
					"body": "no deeplink",
				},
			},
		}
		_, err := extractDeeplink(output)
		require.Error(t, err)
		require.Contains(t, err.Error(), "deeplink not found")
	})

	t.Run("returns error when outputs key missing", func(t *testing.T) {
		output := map[string]any{
			"step-1": map[string]any{
				"body": "no outputs key",
			},
		}
		_, err := extractDeeplink(output)
		require.Error(t, err)
	})

	t.Run("returns error when deeplink is empty string", func(t *testing.T) {
		output := map[string]any{
			"step-1": map[string]any{
				"outputs": map[string]any{
					"deeplink": "",
				},
			},
		}
		_, err := extractDeeplink(output)
		require.Error(t, err)
	})

	t.Run("returns error on empty output map", func(t *testing.T) {
		_, err := extractDeeplink(map[string]any{})
		require.Error(t, err)
	})
}

func rawBody(yaml string) *strings.Reader {
	return strings.NewReader(yaml)
}

func errWorkflowFailed() error {
	return fmt.Errorf("workflow execution failed")
}
