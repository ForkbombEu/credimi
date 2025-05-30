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
	DockerInspectContainerFailed: {"CRE216", "to inspect container"},
	DockerFetchLogsFailed:        {"CRE217", "failed to fetch container logs"},
	CommandExecutionFailed:       {"CRE301", "Command execution failed"},
	StepCIRunFailed:              {"CRE302", "StepCI run failed"},
	UnexpectedStepCIOutput:       {"CRE303", "Unexpected output from StepCI run"},
	UnexpectedHTTPResponse:       {"CRE304", "Unexpected HTTP response body"},
	EWCCheckFailed:               {"CRE305", "EWC check failed"},
	EudiwCheckFailed:             {"CRE306", "Eudiw check failed"},
	UnexpectedDockerOutput:       {"CRE307", "Unexpected output from docker container"},
	ZenroomExecutionFailed:       {"CRE308", "execution of Zenroom failed"},
	ReadFromReaderFailed:         {"CRE901", "Failed to read from reader"},
	CopyFromReaderFailed:         {"CRE902", "Failed to copy from reader"},
	MkdirFailed:                  {"CRE903", "Failed to create a new folder"},
	WriteFileFailed:              {"CRE904", "Failed to write to a file"},
}

const (
	MissingOrInvalidConfig       = "CRE201"
	MissingOrInvalidPayload      = "CRE202"
	JSONMarshalFailed            = "CRE203"
	JSONUnmarshalFailed          = "CRE204"
	TemplateRenderFailed         = "CRE205"
	ParseURLFailed               = "CRE206"
	EmailSendFailed              = "CRE207"
	CreateHTTPRequestFailed      = "CRE208"
	ExecuteHTTPRequestFailed     = "CRE209"
	UnregisteredStructType       = "CRE210"
	DockerClientCreationFailed   = "CRE211"
	DockerPullImageFailed        = "CRE212"
	DockerCreateContainerFailed  = "CRE213"
	DockerStartContainerFailed   = "CRE214"
	DockerWaitContainerFailed    = "CRE215"
	DockerInspectContainerFailed = "CRE216"
	DockerFetchLogsFailed        = "CRE217"
	CommandExecutionFailed       = "CRE301"
	StepCIRunFailed              = "CRE302"
	UnexpectedStepCIOutput       = "CRE303"
	UnexpectedHTTPResponse       = "CRE304"
	EWCCheckFailed               = "CRE305"
	EudiwCheckFailed             = "CRE306"
	UnexpectedDockerOutput       = "CRE307"
	ZenroomExecutionFailed       = "CRE308"
	ReadFromReaderFailed         = "CRE901"
	CopyFromReaderFailed         = "CRE902"
	MkdirFailed                  = "CRE903"
	WriteFileFailed              = "CRE904"
)
