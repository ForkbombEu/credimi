// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

type MessageState struct {
	State   string `json:"state"`
	Details string `json:"details"`
}

type WorkflowExecutionInfo struct {
	Name                 string `json:"name"`
	ID                   string `json:"id"`
	RunID                string `json:"runId"`
	Status               string `json:"status"` 
	StateTransitionCount string `json:"stateTransitionCount"`
	StartTime            string `json:"startTime"`
	CloseTime            string `json:"closeTime"`
	ExecutionTime        string `json:"executionTime"`
	HistorySizeBytes     string `json:"historySizeBytes"`
	HistoryLength        string `json:"historyLength"`
	AssignedBuildID      string `json:"assignedBuildId"`
	// SearchAttributes    *WorkflowSearchAttributes  `json:"searchAttributes,omitempty"`
	// Memo                *Memo                      `json:"memo,omitempty"`
	// VersioningInfo      *VersioningInfo            `json:"versioningInfo,omitempty"`
}

type WorkflowExecution struct {
	Name             string       `json:"name"`
	ID               string       `json:"id"`
	RunID            string       `json:"runId"`
	StartTime        string       `json:"startTime"`
	EndTime          string       `json:"endTime"`
	ExecutionTime    string       `json:"executionTime"`
	Status           MessageState `json:"status"`
	TaskQueue        *string      `json:"taskQueue,omitempty"`
	HistoryEvents    string       `json:"historyEvents"`
	HistorySizeBytes string       `json:"historySizeBytes"`
	// MostRecentWorkerVersionStamp *MostRecentWorkflowVersionStamp `json:"mostRecentWorkerVersionStamp,omitempty"`
	AssignedBuildID *string `json:"assignedBuildId,omitempty"`
	// SearchAttributes          *DecodedWorkflowSearchAttributes `json:"searchAttributes,omitempty"`
	// Memo                      Memo                             `json:"memo"`
	// RootExecution             *WorkflowIdentifier              `json:"rootExecution,omitempty"`
	// PendingChildren           []PendingChildren                `json:"pendingChildren"`
	// PendingNexusOperations    []PendingNexusOperation          `json:"pendingNexusOperations"`
	// PendingActivities         []PendingActivity                `json:"pendingActivities"`
	// PendingWorkflowTask       PendingWorkflowTaskInfo          `json:"pendingWorkflowTask"`
	// StateTransitionCount      string                           `json:"stateTransitionCount"`
	ParentNamespaceID *string `json:"parentNamespaceId,omitempty"`
	// Parent                    *WorkflowIdentifier              `json:"parent,omitempty"`
	URL       string `json:"url"`
	IsRunning bool   `json:"isRunning"`
	// DefaultWorkflowTaskTimeout Duration                        `json:"defaultWorkflowTaskTimeout"`
	CanBeTerminated bool `json:"canBeTerminated"`
	// Callbacks                 Callbacks                        `json:"callbacks"`
	// VersioningInfo            *VersioningInfo                  `json:"versioningInfo,omitempty"`
	// Summary                   *Payload                         `json:"summary,omitempty"`
	// Details                   *Payload                         `json:"details,omitempty"`
}

type Event struct {
	EventID    string `json:"eventId"`
	EventType  string `json:"eventType"`
	EventTime  string `json:"eventTime"`
	Details    string `json:"details"`
	Attributes string `json:"attributes"`
}

type WorkflowExecutionAPIResponse struct {
	WorkflowExecutionInfo *WorkflowExecutionInfo `json:"workflowExecutionInfo,omitempty"`
	// PendingActivities     []PendingActivityInfo          `json:"pendingActivities,omitempty"`
	// PendingChildren       []PendingChildren              `json:"pendingChildren,omitempty"`
	// PendingNexusOperations []PendingNexusOperation       `json:"pendingNexusOperations,omitempty"`
	// ExecutionConfig       WorkflowExecutionConfigWithMetadata `json:"executionConfig"`
	// Callbacks             Callbacks                      `json:"callbacks"`
	// PendingWorkflowTask   PendingWorkflowTaskInfo        `json:"pendingWorkflowTask"`
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
