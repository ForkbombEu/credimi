// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package errorcodes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var allCodes = []string{
	MissingOrInvalidConfig,
	MissingOrInvalidPayload,
	JSONMarshalFailed,
	JSONUnmarshalFailed,
	TemplateRenderFailed,
	ParseURLFailed,
	EmailSendFailed,
	CreateHTTPRequestFailed,
	ExecuteHTTPRequestFailed,
	UnregisteredStructType,
	DockerClientCreationFailed,
	DockerPullImageFailed,
	DockerCreateContainerFailed,
	DockerStartContainerFailed,
	DockerWaitContainerFailed,
	DockerInspectContainerFailed,
	DockerFetchLogsFailed,
	InvalidSchema,
	SchemaValidationFailed,
	UnexpectedActivityOutput,
	UnexpectedActivityError,
	UnexpectedActivityErrorDetails,
	IsNotCredentialIssuer,
	InvalidJWTFormat,
	DecodeFailed,
	CESRParsingError,
	PipelineParsingError,
	PipelineInputError,
	PipelineExecutionError,
	ChildWorkflowExecutionError,
	WorkflowCancellationError,
	CommandExecutionFailed,
	StepCIRunFailed,
	UnexpectedStepCIOutput,
	UnexpectedHTTPResponse,
	EWCCheckFailed,
	EudiwCheckFailed,
	UnexpectedDockerOutput,
	ZenroomExecutionFailed,
	OpenIDnetCheckFailed,
	UnexpectedHTTPStatusCode,
	DockerCommandExecutionFailed,
	MobileRunnerBusy,
	ReadFromReaderFailed,
	CopyFromReaderFailed,
	MkdirFailed,
	WriteFileFailed,
	TempFileCreationFailed,
	ReadFileFailed,
}

func TestCodesRegistered(t *testing.T) {
	require.Len(t, Codes, len(allCodes))

	for _, code := range allCodes {
		entry, ok := Codes[code]
		require.True(t, ok, "missing code entry for %s", code)
		require.Equal(t, code, entry.Code)
		require.NotEmpty(t, entry.Description)
	}
}

func TestCodesHaveUniqueValues(t *testing.T) {
	seen := make(map[string]struct{}, len(Codes))

	for key, entry := range Codes {
		require.NotEmpty(t, entry.Code, "empty code for key %s", key)
		require.NotEmpty(t, entry.Description, "empty description for code %s", key)
		require.Equal(t, key, entry.Code, "mismatched code value for key %s", key)

		_, exists := seen[entry.Code]
		require.False(t, exists, "duplicate code value %s", entry.Code)
		seen[entry.Code] = struct{}{}
	}
}
