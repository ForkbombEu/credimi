// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/stretchr/testify/require"
)

func TestCanonifyPreviewError(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{
			name:       "exhausted name suffixes",
			err:        canonify.ErrExhaustedAttempts,
			wantStatus: http.StatusConflict,
		},
		{
			name:       "wrapped exhausted name suffixes",
			err:        fmt.Errorf("canonify: %w", canonify.ErrExhaustedAttempts),
			wantStatus: http.StatusConflict,
		},
		{
			name:       "database failure",
			err:        errors.New("canonify: resolve path: no such table"),
			wantStatus: http.StatusInternalServerError,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			apiErr := canonifyPreviewError("failed_to_canonify_device_name", tc.err)
			require.Equal(t, tc.wantStatus, apiErr.Code)
			require.Equal(t, "failed_to_canonify_device_name", apiErr.Reason)
			require.Equal(t, tc.err.Error(), apiErr.Message)
		})
	}
}

func TestPreviewMobileDeviceIDDatabaseFailure(t *testing.T) {
	app := setupMobileRunnerApp(t)
	defer app.Cleanup()

	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	createMobileRunnerRecord(t, app, orgID, "Existing Runner", "http://runner.test", false)
	// The device name lookup now fails with a database error, not a conflict.
	_, err = app.DB().NewQuery("DROP TABLE mobile_devices").Execute()
	require.NoError(t, err)

	event := performMobileRunnerRequest(
		t,
		app,
		user,
		"/api/mobile-device/preview-id",
		PreviewMobileDeviceIDRequest{
			RunnerID: "usera-s-organization/existing-runner",
			Name:     "Pixel-7a",
		},
	)

	err = HandlePreviewMobileDeviceID()(event)
	var apiErr *apierror.APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, http.StatusInternalServerError, apiErr.Code)
	require.Equal(t, "failed_to_canonify_device_name", apiErr.Reason)
}
