// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"go.temporal.io/sdk/client"
)

type MessageState struct {
	State   string `json:"state"`
	Details string `json:"details"`
}

type WorkflowExecutionInfo struct {
	Name                 string                    `json:"name"`
	ID                   string                    `json:"id"`
	RunID                string                    `json:"runId"`
	Status               string                    `json:"status"`
	StateTransitionCount string                    `json:"stateTransitionCount"`
	StartTime            string                    `json:"startTime"`
	CloseTime            string                    `json:"closeTime"`
	ExecutionTime        string                    `json:"executionTime"`
	HistorySizeBytes     string                    `json:"historySizeBytes"`
	HistoryLength        string                    `json:"historyLength"`
	AssignedBuildID      string                    `json:"assignedBuildId"`
	SearchAttributes     *WorkflowSearchAttributes `json:"searchAttributes,omitempty"`
	Memo                 *Memo                     `json:"memo,omitempty"`
	VersioningInfo       *VersioningInfo           `json:"versioningInfo,omitempty"`
}

type WorkflowExecution struct {
	Execution                    *WorkflowIdentifier              `json:"execution"`
	Type                         WorkflowType                     `json:"type"`
	StartTime                    string                           `json:"startTime"`
	CloseTime                    string                           `json:"closeTime"`
	ExecutionTime                string                           `json:"executionTime"`
	Status                       string                           `json:"status"`
	TaskQueue                    *string                          `json:"taskQueue,omitempty"`
	HistoryEvents                string                           `json:"historyEvents"`
	HistorySizeBytes             string                           `json:"historySizeBytes"`
	MostRecentWorkerVersionStamp *MostRecentWorkflowVersionStamp  `json:"mostRecentWorkerVersionStamp,omitempty"`
	AssignedBuildID              *string                          `json:"assignedBuildId,omitempty"`
	SearchAttributes             *DecodedWorkflowSearchAttributes `json:"searchAttributes,omitempty"`
	Memo                         *Memo                            `json:"memo,omitempty"`
	RootExecution                *WorkflowIdentifier              `json:"rootExecution,omitempty"`
	PendingChildren              []PendingChildren                `json:"pendingChildren,omitempty"`
	PendingNexusOperations       []PendingNexusOperation          `json:"pendingNexusOperations,omitempty"`
	PendingActivities            []PendingActivityInfo            `json:"pendingActivities,omitempty"`
	PendingWorkflowTask          *PendingWorkflowTaskInfo         `json:"pendingWorkflowTask,omitempty"`
	StateTransitionCount         string                           `json:"stateTransitionCount"`
	ParentNamespaceID            *string                          `json:"parentNamespaceId,omitempty"`
	ParentExecution              *WorkflowIdentifier              `json:"parentExecution,omitempty"`
	URL                          string                           `json:"url"`
	IsRunning                    bool                             `json:"isRunning"`
	DefaultWorkflowTaskTimeout   *Duration                        `json:"defaultWorkflowTaskTimeout,omitempty"`
	CanBeTerminated              bool                             `json:"canBeTerminated"`
	Callbacks                    *Callbacks                       `json:"callbacks,omitempty"`
	VersioningInfo               *VersioningInfo                  `json:"versioningInfo,omitempty"`
	Summary                      *Payload                         `json:"summary,omitempty"`
	Details                      *Payload                         `json:"details,omitempty"`
}

type WorkflowExecutionSummary struct {
	Execution   *WorkflowIdentifier         `json:"execution"          validate:"required"`
	Type        WorkflowType                `json:"type"               validate:"required"`
	StartTime   string                      `json:"startTime"`
	EndTime     string                      `json:"endTime"`
	Status      string                      `json:"status"             validate:"required"`
	DisplayName string                      `json:"displayName"        validate:"required"`
	Children    []*WorkflowExecutionSummary `json:"children,omitempty"`
	Results     []PipelineResults           `json:"results,omitempty"`
}

type WorkflowExecutionAPIResponse struct {
	WorkflowExecutionInfo  *WorkflowExecutionInfo               `json:"workflowExecutionInfo,omitempty"`
	PendingActivities      []PendingActivityInfo                `json:"pendingActivities,omitempty"`
	PendingChildren        []PendingChildren                    `json:"pendingChildren,omitempty"`
	PendingNexusOperations []PendingNexusOperation              `json:"pendingNexusOperations,omitempty"`
	ExecutionConfig        *WorkflowExecutionConfigWithMetadata `json:"executionConfig,omitempty"`
	Callbacks              *Callbacks                           `json:"callbacks,omitempty"`
	PendingWorkflowTask    *PendingWorkflowTaskInfo             `json:"pendingWorkflowTask,omitempty"`
}

type PipelineResults struct {
	Video      string `json:"video,omitempty"`
	Screenshot string `json:"screenshot,omitempty"`
}

type WorkflowStatus string

const (
	WorkflowStatusRunning        WorkflowStatus = "Running"
	WorkflowStatusTimedOut       WorkflowStatus = "TimedOut"
	WorkflowStatusCompleted      WorkflowStatus = "Completed"
	WorkflowStatusFailed         WorkflowStatus = "Failed"
	WorkflowStatusContinuedAsNew WorkflowStatus = "ContinuedAsNew"
	WorkflowStatusCanceled       WorkflowStatus = "Canceled"
	WorkflowStatusTerminated     WorkflowStatus = "Terminated"
)

type FilterParameters struct {
	WorkflowID      *string         `json:"workflowId,omitempty"`
	WorkflowType    *string         `json:"workflowType,omitempty"`
	ExecutionStatus *WorkflowStatus `json:"executionStatus,omitempty"`
	TimeRange       interface{}     `json:"timeRange,omitempty"` // string or Duration
	Query           *string         `json:"query,omitempty"`
}

type ArchiveFilterParameters struct {
	WorkflowID      *string         `json:"workflowId,omitempty"`
	WorkflowType    *string         `json:"workflowType,omitempty"`
	ExecutionStatus *WorkflowStatus `json:"executionStatus,omitempty"`
	CloseTime       interface{}     `json:"closeTime,omitempty"` // string or Duration
	Query           *string         `json:"query,omitempty"`
}

type SearchAttributeType string

const (
	Bool        SearchAttributeType = "Bool"
	Datetime    SearchAttributeType = "Datetime"
	Double      SearchAttributeType = "Double"
	Int         SearchAttributeType = "Int"
	Keyword     SearchAttributeType = "Keyword"
	Text        SearchAttributeType = "Text"
	KeywordList SearchAttributeType = "KeywordList"
	Unspecified SearchAttributeType = "Unspecified"
)

type SearchAttributes map[string]SearchAttributeType

type SearchAttributesResponse struct {
	CustomAttributes map[string]SearchAttributeType `json:"customAttributes"`
	SystemAttributes map[string]SearchAttributeType `json:"systemAttributes"`
	StorageSchema    interface{}                    `json:"storageSchema"` // adjust if known
}

type MostRecentWorkflowVersionStamp struct {
	WorkflowVersionTimestamp string `json:"workflowVersionTimestamp"`
	UseVersioning            *bool  `json:"useVersioning,omitempty"`
}

type Payload struct {
	Metadata map[string]string `json:"metadata,omitempty"`
	Data     *string           `json:"data,omitempty"`
}

type Payloads []Payload

// Response types for checks handlers

// ReRunCheckResponse represents the response when re-running a check
type ReRunCheckResponse struct {
	WorkflowID string `json:"workflow_id" validate:"required"`
	RunID      string `json:"run_id"      validate:"required"`
}

// ListMyChecksResponse represents the response containing list of workflow executions
type ListMyChecksResponse struct {
	Executions []*WorkflowExecutionSummary `json:"executions" validate:"required"`
}

// GetMyCheckRunResponse represents the response for getting a specific check run
type GetMyCheckRunResponse struct {
	WorkflowExecutionInfo  *WorkflowExecutionInfo               `json:"workflowExecutionInfo,omitempty"`
	PendingActivities      []PendingActivityInfo                `json:"pendingActivities,omitempty"`
	PendingChildren        []PendingChildren                    `json:"pendingChildren,omitempty"`
	PendingNexusOperations []PendingNexusOperation              `json:"pendingNexusOperations,omitempty"`
	ExecutionConfig        *WorkflowExecutionConfigWithMetadata `json:"executionConfig,omitempty"`
	Callbacks              *Callbacks                           `json:"callbacks,omitempty"`
	PendingWorkflowTask    *PendingWorkflowTaskInfo             `json:"pendingWorkflowTask,omitempty"`
}

// ListMyCheckRunsResponse represents the response containing list of runs for a specific check
type ListMyCheckRunsResponse struct {
	Executions []*WorkflowExecutionSummary `json:"executions" validate:"required"`
}

// GetMyCheckRunHistoryResponse represents the response containing workflow execution history
type GetMyCheckRunHistoryResponse struct {
	History   []map[string]interface{} `json:"history"   validate:"required"`
	Count     int                      `json:"count"     validate:"required"`
	Time      string                   `json:"time"      validate:"required"`
	CheckID   string                   `json:"checkId"   validate:"required"`
	RunID     string                   `json:"runId"     validate:"required"`
	Namespace string                   `json:"namespace" validate:"required"`
}

// CancelMyCheckRunResponse represents the response when canceling a workflow execution
type CancelMyCheckRunResponse struct {
	Message   string `json:"message"   validate:"required"`
	CheckID   string `json:"checkId"   validate:"required"`
	RunID     string `json:"runId"     validate:"required"`
	Status    string `json:"status"    validate:"required"`
	Time      string `json:"time"      validate:"required"`
	Namespace string `json:"namespace" validate:"required"`
}

// TerminateMyCheckRunResponse represents the response when terminating a workflow execution
type TerminateMyCheckRunResponse struct {
	Message   string `json:"message"   validate:"required"`
	CheckID   string `json:"checkId"   validate:"required"`
	RunID     string `json:"runId"     validate:"required"`
	Status    string `json:"status"    validate:"required"`
	Time      string `json:"time"      validate:"required"`
	Namespace string `json:"namespace" validate:"required"`
}

// ExportMyCheckRunResponse represents the response when exporting a check run
type ExportMyCheckRunResponse struct {
	Export ExportData `json:"export" validate:"required"`
}

// ExportData represents the exported check run data
type ExportData struct {
	CheckID string                 `json:"checkId" validate:"required"`
	RunID   string                 `json:"runId"   validate:"required"`
	Input   map[string]interface{} `json:"input"   validate:"required"`
	Config  map[string]interface{} `json:"config"  validate:"required"`
}

// ChecksLogsResponse represents the response for workflow logs operations
type ChecksLogsResponse struct {
	Channel    string `json:"channel"     validate:"required"`
	WorkflowID string `json:"workflow_id" validate:"required"`
	RunID      string `json:"run_id"      validate:"required"`
	Message    string `json:"message"     validate:"required"`
	Status     string `json:"status"      validate:"required"`
	Time       string `json:"time"        validate:"required"`
	Namespace  string `json:"namespace"   validate:"required"`
}

// Supporting types for workflow execution details

// HistoryEvent represents a single event in workflow history
type HistoryEvent struct {
	EventID            string                 `json:"eventId"                      validate:"required"`
	EventTime          string                 `json:"eventTime"                    validate:"required"`
	EventType          string                 `json:"eventType"                    validate:"required"`
	Version            string                 `json:"version,omitempty"`
	TaskID             string                 `json:"taskId,omitempty"`
	WorkerVersionStamp *WorkerVersionStamp    `json:"workerVersionStamp,omitempty"`
	Attributes         map[string]interface{} `json:"attributes,omitempty"`
}

// WorkerVersionStamp represents version information for worker
type WorkerVersionStamp struct {
	BuildID       string `json:"buildId,omitempty"`
	BundleID      string `json:"bundleId,omitempty"`
	UseVersioning *bool  `json:"useVersioning,omitempty"`
}

// PendingActivityInfo represents pending activity information
type PendingActivityInfo struct {
	ActivityID         string               `json:"activityId"                   validate:"required"`
	ActivityType       ActivityType         `json:"activityType"                 validate:"required"`
	State              PendingActivityState `json:"state"                        validate:"required"`
	HeartbeatDetails   *Payloads            `json:"heartbeatDetails,omitempty"`
	LastHeartbeatTime  string               `json:"lastHeartbeatTime,omitempty"`
	LastStartedTime    string               `json:"lastStartedTime,omitempty"`
	Attempt            int32                `json:"attempt,omitempty"`
	MaximumAttempts    int32                `json:"maximumAttempts,omitempty"`
	ScheduledTime      string               `json:"scheduledTime,omitempty"`
	ExpirationTime     string               `json:"expirationTime,omitempty"`
	LastFailure        *Failure             `json:"lastFailure,omitempty"`
	LastWorkerIdentity string               `json:"lastWorkerIdentity,omitempty"`
	AssignedBuildID    string               `json:"assignedBuildId,omitempty"`
}

// ActivityType represents activity type information
type ActivityType struct {
	Name string `json:"name" validate:"required"`
}

// PendingActivityState represents the state of a pending activity
type PendingActivityState string

const (
	PendingActivityStateScheduled       PendingActivityState = "Scheduled"
	PendingActivityStateStarted         PendingActivityState = "Started"
	PendingActivityStateCancelRequested PendingActivityState = "CancelRequested"
)

// PendingChildren represents pending child workflow information
type PendingChildren struct {
	WorkflowID        string `json:"workflowId"                  validate:"required"`
	RunID             string `json:"runId,omitempty"`
	WorkflowTypeName  string `json:"workflowTypeName"            validate:"required"`
	InitiatedID       string `json:"initiatedId"                 validate:"required"`
	ParentClosePolicy string `json:"parentClosePolicy,omitempty"`
}

// PendingNexusOperation represents pending Nexus operation information
type PendingNexusOperation struct {
	Endpoint                string                     `json:"endpoint"                          validate:"required"`
	Service                 string                     `json:"service"                           validate:"required"`
	Operation               string                     `json:"operation"                         validate:"required"`
	OperationID             string                     `json:"operationId"                       validate:"required"`
	ScheduledEventID        string                     `json:"scheduledEventId"                  validate:"required"`
	State                   PendingNexusOperationState `json:"state"                             validate:"required"`
	Attempt                 int32                      `json:"attempt,omitempty"`
	NextAttemptScheduleTime string                     `json:"nextAttemptScheduleTime,omitempty"`
	LastAttemptCompleteTime string                     `json:"lastAttemptCompleteTime,omitempty"`
	LastAttemptFailure      *Failure                   `json:"lastAttemptFailure,omitempty"`
}

// PendingNexusOperationState represents the state of a pending Nexus operation
type PendingNexusOperationState string

const (
	PendingNexusOperationStateScheduled  PendingNexusOperationState = "Scheduled"
	PendingNexusOperationStateBackingOff PendingNexusOperationState = "BackingOff"
	PendingNexusOperationStateStarted    PendingNexusOperationState = "Started"
)

// WorkflowExecutionConfigWithMetadata represents workflow execution configuration
type WorkflowExecutionConfigWithMetadata struct {
	TaskQueue                  *TaskQueue    `json:"taskQueue,omitempty"`
	WorkflowExecutionTimeout   *Duration     `json:"workflowExecutionTimeout,omitempty"`
	WorkflowRunTimeout         *Duration     `json:"workflowRunTimeout,omitempty"`
	DefaultWorkflowTaskTimeout *Duration     `json:"defaultWorkflowTaskTimeout,omitempty"`
	UserMetadata               *UserMetadata `json:"userMetadata,omitempty"`
}

// TaskQueue represents task queue information
type TaskQueue struct {
	Name       string        `json:"name"                 validate:"required"`
	Kind       TaskQueueKind `json:"kind,omitempty"`
	NormalName string        `json:"normalName,omitempty"`
}

// TaskQueueKind represents the kind of task queue
type TaskQueueKind string

const (
	TaskQueueKindNormal TaskQueueKind = "Normal"
	TaskQueueKindSticky TaskQueueKind = "Sticky"
)

// Duration represents a time duration
type Duration struct {
	Seconds int64 `json:"seconds,omitempty"`
	Nanos   int32 `json:"nanos,omitempty"`
}

// UserMetadata represents user metadata
type UserMetadata struct {
	Summary *Payload `json:"summary,omitempty"`
	Details *Payload `json:"details,omitempty"`
}

// Callbacks represents workflow callbacks
type Callbacks struct {
	Callbacks []Callback `json:"callbacks,omitempty"`
}

// Callback represents a single callback
type Callback struct {
	Trigger                 *CallbackInfo `json:"trigger,omitempty"`
	PublicInfo              *CallbackInfo `json:"publicInfo,omitempty"`
	State                   CallbackState `json:"state,omitempty"`
	Attempt                 int32         `json:"attempt,omitempty"`
	LastAttemptCompleteTime string        `json:"lastAttemptCompleteTime,omitempty"`
	LastAttemptFailure      *Failure      `json:"lastAttemptFailure,omitempty"`
	NextAttemptScheduleTime string        `json:"nextAttemptScheduleTime,omitempty"`
}

// CallbackInfo represents callback information
type CallbackInfo struct {
	Callback         *Payload `json:"callback,omitempty"`
	RegistrationTime string   `json:"registrationTime,omitempty"`
}

// CallbackState represents callback state
type CallbackState string

const (
	CallbackStateStandby    CallbackState = "Standby"
	CallbackStateScheduled  CallbackState = "Scheduled"
	CallbackStateBackingOff CallbackState = "BackingOff"
	CallbackStateFailed     CallbackState = "Failed"
	CallbackStateSucceeded  CallbackState = "Succeeded"
)

// PendingWorkflowTaskInfo represents pending workflow task information
type PendingWorkflowTaskInfo struct {
	State                 PendingWorkflowTaskState `json:"state,omitempty"`
	ScheduledTime         string                   `json:"scheduledTime,omitempty"`
	OriginalScheduledTime string                   `json:"originalScheduledTime,omitempty"`
	StartedTime           string                   `json:"startedTime,omitempty"`
	Attempt               int32                    `json:"attempt,omitempty"`
	LastFailure           *Failure                 `json:"lastFailure,omitempty"`
}

// PendingWorkflowTaskState represents the state of a pending workflow task
type PendingWorkflowTaskState string

const (
	PendingWorkflowTaskStateScheduled PendingWorkflowTaskState = "Scheduled"
	PendingWorkflowTaskStateStarted   PendingWorkflowTaskState = "Started"
)

// Failure represents failure information
type Failure struct {
	Message     string            `json:"message,omitempty"`
	Source      string            `json:"source,omitempty"`
	StackTrace  string            `json:"stackTrace,omitempty"`
	Cause       *Failure          `json:"cause,omitempty"`
	FailureInfo map[string]string `json:"failureInfo,omitempty"`
}

// Additional types needed for complete Temporal mapping

// WorkflowSearchAttributes represents search attributes for workflows
type WorkflowSearchAttributes map[string]interface{}

// DecodedWorkflowSearchAttributes represents decoded search attributes
type DecodedWorkflowSearchAttributes map[string]interface{}

// Memo represents workflow memo information
type Memo struct {
	Fields map[string]*Payload `json:"fields,omitempty"`
}

// VersioningInfo represents versioning information
type VersioningInfo struct {
	UseVersioning *bool `json:"useVersioning,omitempty"`
}

type WorkflowType struct {
	Name string `json:"name,omitempty"`
}

// WorkflowIdentifier represents a workflow identifier
type WorkflowIdentifier struct {
	WorkflowID string `json:"workflowId"      validate:"required"`
	RunID      string `json:"runId,omitempty"`
}

type ScheduleInfo struct {
	ID string `json:"ID" validate:"required"`

	Spec            *client.ScheduleSpec `json:"Spec,omitempty"`
	WorkflowType    *WorkflowType        `json:"WorkflowType,omitempty"`
	NextActionTimes []time.Time          `json:"NextActionTimes,omitempty"`
	Paused          bool                 `json:"Paused,omitempty"`
	Memo            *Memo                `json:"Memo,omitempty"`
}

type ScheduleInfoSummary struct {
	ID string `json:"id" validate:"required"`

	ScheduleMode   workflowengine.ScheduleMode `json:"schedule_mode,omitempty"`
	WorkflowType   *WorkflowType               `json:"workflowType,omitempty"`
	DisplayName    string                      `json:"display_name,omitempty"`
	PipelineID     string                      `json:"pipeline_id,omitempty"`
	NextActionTime string                      `json:"next_action_time,omitempty"`
	Paused         bool                        `json:"paused,omitempty"`
}

// ListMySchedulesResponse represents a response for listing schedules
type ListMySchedulesResponse struct {
	Schedules []*ScheduleInfoSummary `json:"schedules,omitempty" validate:"required"`
}

// CancelScheduleResponse represents a response for canceling a schedule
type CancelScheduleResponse struct {
	Message    string `json:"message"    validate:"required"`
	ScheduleID string `json:"scheduleId" validate:"required"`
	Status     string `json:"status"     validate:"required"`
	Time       string `json:"time"       validate:"required"`
	Namespace  string `json:"namespace"  validate:"required"`
}

// PauseScheduleResponse represents a response for pausing a schedule
type PauseScheduleResponse struct {
	Message    string `json:"message"    validate:"required"`
	ScheduleID string `json:"scheduleId" validate:"required"`
	Status     string `json:"status"     validate:"required"`
	Time       string `json:"time"       validate:"required"`
	Namespace  string `json:"namespace"  validate:"required"`
}

// ResumeScheduleResponse represents a response for resuming a schedule
type ResumeScheduleResponse struct {
	Message    string `json:"message"    validate:"required"`
	ScheduleID string `json:"scheduleId" validate:"required"`
	Status     string `json:"status"     validate:"required"`
	Time       string `json:"time"       validate:"required"`
	Namespace  string `json:"namespace"  validate:"required"`
}

const RedirectFlagTrue = "true"

func getStringFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func decodeFromTemporalPayload(encoded string) string {
	if encoded == "" {
		return ""
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		// fallback: return original string if decode fails
		decoded = []byte(encoded)
	}

	clean := strings.Trim(string(decoded), `"`)

	return clean
}

const testDataDir = "../../../../test_pb_data"
