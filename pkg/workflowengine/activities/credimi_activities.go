// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"errors"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
)

// Names of the activities that read and write Credimi records directly.
const (
	ResolveRecordActivityName                  = "Resolve a Credimi record"
	GetCredentialOfferActivityName             = "Get a credential offer"
	GetUseCaseVerificationDeeplinkActivityName = "Get a use case verification deeplink"
	DeleteTempRecordActivityName               = "Delete a temporary record"
	GetMobileDeviceActivityName                = "Get a mobile device"
	ValidateDeviceAccessActivityName           = "Validate mobile device access"
	SendRealtimeLogsActivityName               = "Send realtime logs"
	StoreCredentialIssuerActivityName          = "Store a credential issuer"
	StoreIssuerCredentialActivityName          = "Store an issuer credential"
	DeletePipelineResultFilesActivityName      = "Delete old pipeline result files"
)

// errRecordNotFound reports a Credimi record that does not exist or that the
// caller may not read.
var errRecordNotFound = errors.New("credimi record not found")

// CredimiActivities returns one instance of every activity that works on the
// Credimi database. It is the single registration source for workers.
func CredimiActivities(app core.App) []workflowengine.ExecutableActivity {
	return []workflowengine.ExecutableActivity{
		NewResolveRecordActivity(app),
		NewGetCredentialOfferActivity(app),
		NewGetUseCaseVerificationDeeplinkActivity(app),
		NewDeleteTempRecordActivity(app),
		NewGetMobileDeviceActivity(app),
		NewValidateDeviceAccessActivity(app),
		NewSendRealtimeLogsActivity(app),
		NewStoreCredentialIssuerActivity(app),
		NewStoreIssuerCredentialActivity(app),
		NewDeletePipelineResultFilesActivity(app),
	}
}

// credimiActivityError builds an activity error for errorcodes key code.
func credimiActivityError(
	a *workflowengine.BaseActivity,
	code string,
	retryable bool,
	err error,
) error {
	errCode := errorcodes.Codes[code]
	failure := workflowengine.ActivityError{
		Code:    errCode.Code,
		Summary: errCode.Description,
		Message: err.Error(),
	}
	if retryable {
		return a.NewActivityError(failure)
	}
	return a.NewNonRetryableActivityError(failure)
}

// recordLookupError maps ErrRecordNotFound to a non-retryable CRE233 and any
// other error to a retryable CRE235.
func recordLookupError(a *workflowengine.BaseActivity, err error) error {
	if errors.Is(err, errRecordNotFound) {
		return credimiActivityError(a, errorcodes.RecordNotFound, false, err)
	}
	return credimiActivityError(a, errorcodes.DatabaseOperationFailed, true, err)
}
