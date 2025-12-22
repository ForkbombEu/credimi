// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package errorcodes

type Code struct {
	Code        string
	Description string
}

var Codes = map[string]Code{
	MissingOrInvalidConfig:       {"CRE201", "Missing or invalid required configuration"},
	MissingOrInvalidPayload:      {"CRE202", "Missing or invalid value in payload"},
	JSONMarshalFailed:            {"CRE203", "Failed to marshal JSON"},
	JSONUnmarshalFailed:          {"CRE204", "Failed to unmarshal JSON"},
	TemplateRenderFailed:         {"CRE205", "Failed to render YAML template"},
	ParseURLFailed:               {"CRE206", "Failed to parse URL"},
	EmailSendFailed:              {"CRE207", "Failed to send email"},
	CreateHTTPRequestFailed:      {"CRE208", "Failed to create HTTP request"},
	ExecuteHTTPRequestFailed:     {"CRE209", "Failed to execute HTTP request"},
	UnregisteredStructType:       {"CRE210", "Unregistered struct type for JSON validation"},
	DockerClientCreationFailed:   {"CRE211", "Failed to create Docker client"},
	DockerPullImageFailed:        {"CRE212", "Failed to pull image"},
	DockerCreateContainerFailed:  {"CRE213", "Failed to create container"},
	DockerStartContainerFailed:   {"CRE214", "Failed to start container"},
	DockerWaitContainerFailed:    {"CRE215", "Failed waiting for container"},
	DockerInspectContainerFailed: {"CRE216", "Failed to inspect container"},
	DockerFetchLogsFailed:        {"CRE217", "Failed to fetch container logs"},
	InvalidSchema:                {"CRE218", "Invalid json schema"},
	SchemaValidationFailed:       {"CRE219", "Schema validation failed"},
	UnexpectedActivityOutput:     {"CRE220", "Unexpected output from activity"},
	UnexpectedActivityError: {
		"CRE221",
		"Activity error should be temporal.ApplicationError",
	},
	UnexpectedActivityErrorDetails: {"CRE222", "Unexpected activity error details type"},
	IsNotCredentialIssuer:          {"CRE223", "The input issuer URL is not a credential issuer"},
	InvalidJWTFormat:               {"CRE224", "Invalid JWT format"},
	DecodeFailed:                   {"CRE225", "Failed to decode string"},
	CESRParsingError:               {"CRE226", "Failed to parse CESR"},
	PipelineParsingError:           {"CRE227", "Failed to parse pipeline yaml"},
	PipelineInputError:             {"CRE228", "Failed to resolve pipeline inputs"},
	PipelineExecutionError:         {"CRE229", "Failed to execute pipeline worflow"},
	ChildWorkflowExecutionError:    {"CRE230", "Failed to execute child workflow"},
	WorkflowCancellationError:      {"CRE231", "Workflow was cancelled"},
	CommandExecutionFailed:         {"CRE301", "Command execution failed"},
	StepCIRunFailed:                {"CRE302", "StepCI run failed"},
	UnexpectedStepCIOutput:         {"CRE303", "Unexpected output from StepCI run"},
	UnexpectedHTTPResponse:         {"CRE304", "Unexpected HTTP response body"},
	EWCCheckFailed:                 {"CRE305", "EWC check failed"},
	EudiwCheckFailed:               {"CRE306", "Eudiw check failed"},
	UnexpectedDockerOutput:         {"CRE307", "Unexpected output from docker container"},
	ZenroomExecutionFailed:         {"CRE308", "Execution of Zenroom failed"},
	OpenIDnetCheckFailed:           {"CRE309", "OpenIDnet check failed"},
	UnexpectedHTTPStatusCode:       {"CRE310", "Unexpected HTTP status code"},
	DockerCommandExecutionFailed:   {"CRE311", "Docker command execution failed"},
	ReadFromReaderFailed:           {"CRE901", "Failed to read from reader"},
	CopyFromReaderFailed:           {"CRE902", "Failed to copy from reader"},
	MkdirFailed:                    {"CRE903", "Failed to create a new folder"},
	WriteFileFailed:                {"CRE904", "Failed to write to a file"},
	TempFileCreationFailed:         {"CRE905", "Failed to create a temporary file"},
	ReadFileFailed:                 {"CRE906", "Failed to read a file"},
}

const (
	MissingOrInvalidConfig         = "CRE201"
	MissingOrInvalidPayload        = "CRE202"
	JSONMarshalFailed              = "CRE203"
	JSONUnmarshalFailed            = "CRE204"
	TemplateRenderFailed           = "CRE205"
	ParseURLFailed                 = "CRE206"
	EmailSendFailed                = "CRE207"
	CreateHTTPRequestFailed        = "CRE208"
	ExecuteHTTPRequestFailed       = "CRE209"
	UnregisteredStructType         = "CRE210"
	DockerClientCreationFailed     = "CRE211"
	DockerPullImageFailed          = "CRE212"
	DockerCreateContainerFailed    = "CRE213"
	DockerStartContainerFailed     = "CRE214"
	DockerWaitContainerFailed      = "CRE215"
	DockerInspectContainerFailed   = "CRE216"
	DockerFetchLogsFailed          = "CRE217"
	InvalidSchema                  = "CRE218"
	SchemaValidationFailed         = "CRE219"
	UnexpectedActivityOutput       = "CRE220"
	UnexpectedActivityError        = "CRE221"
	UnexpectedActivityErrorDetails = "CRE222"
	IsNotCredentialIssuer          = "CRE223"
	InvalidJWTFormat               = "CRE224"
	DecodeFailed                   = "CRE225"
	CESRParsingError               = "CRE226"
	PipelineParsingError           = "CRE227"
	PipelineInputError             = "CRE228"
	PipelineExecutionError         = "CRE229"
	ChildWorkflowExecutionError    = "CRE230"
	WorkflowCancellationError      = "CRE231"
	CommandExecutionFailed         = "CRE301"
	StepCIRunFailed                = "CRE302"
	UnexpectedStepCIOutput         = "CRE303"
	UnexpectedHTTPResponse         = "CRE304"
	EWCCheckFailed                 = "CRE305"
	EudiwCheckFailed               = "CRE306"
	UnexpectedDockerOutput         = "CRE307"
	ZenroomExecutionFailed         = "CRE308"
	OpenIDnetCheckFailed           = "CRE309"
	UnexpectedHTTPStatusCode       = "CRE310"
	DockerCommandExecutionFailed   = "CRE311"
	ReadFromReaderFailed           = "CRE901"
	CopyFromReaderFailed           = "CRE902"
	MkdirFailed                    = "CRE903"
	WriteFileFailed                = "CRE904"
	TempFileCreationFailed         = "CRE905"
	ReadFileFailed                 = "CRE906"
)
