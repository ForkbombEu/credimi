// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apierror

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIError_Error(t *testing.T) {
	apiErr := New(http.StatusBadRequest, "test-domain", "bad", "something went wrong")

	require.Equal(t, "[test-domain:bad] something went wrong", apiErr.Error())
}
