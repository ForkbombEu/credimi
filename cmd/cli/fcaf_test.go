// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRetryableFCAFQueueStatus(t *testing.T) {
	t.Parallel()

	require.False(t, retryableFCAFQueueStatus(http.StatusOK))
	require.False(t, retryableFCAFQueueStatus(http.StatusBadRequest))
	require.False(t, retryableFCAFQueueStatus(http.StatusUnauthorized))
	require.True(t, retryableFCAFQueueStatus(http.StatusRequestTimeout))
	require.True(t, retryableFCAFQueueStatus(http.StatusTooManyRequests))
	require.True(t, retryableFCAFQueueStatus(http.StatusInternalServerError))
	require.True(t, retryableFCAFQueueStatus(http.StatusServiceUnavailable))
}
