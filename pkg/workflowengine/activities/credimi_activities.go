// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"database/sql"
	"errors"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
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

// recordLookupError maps a Credimi record lookup failure to an activity error:
// missing records (errRecordNotFound, sql.ErrNoRows, missing mobile devices or
// runners) are a non-retryable CRE233, inaccessible mobile devices a
// non-retryable CRE234, invalid secrets a non-retryable decode failure, and any
// other error a retryable CRE235.
func recordLookupError(a *workflowengine.BaseActivity, err error) error {
	switch {
	case errors.Is(err, errRecordNotFound),
		errors.Is(err, sql.ErrNoRows),
		errors.Is(err, mobilerunner.ErrDeviceNotFound),
		errors.Is(err, mobilerunner.ErrDeviceRunnerNotFound):
		return a.NewCodedError(errorcodes.RecordNotFound, false, err)
	case errors.Is(err, mobilerunner.ErrDeviceNotAccessible):
		return a.NewCodedError(errorcodes.RecordNotAccessible, false, err)
	case errors.Is(err, errInvalidSecrets):
		return a.NewCodedError(errorcodes.DecodeFailed, false, err)
	default:
		return a.NewCodedError(errorcodes.DatabaseOperationFailed, true, err)
	}
}
