// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"net/http"
	"os"
	"path/filepath"
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

func TestRewriteFCAFOrganization(t *testing.T) {
	t.Parallel()

	input := []byte(
		"global_device_id: forkbomb-bv-andrea/usb/device\n" +
			"action_id: forkbomb-bv-andrea/eudiw-beta-wallet/onboarding-1\n",
	)

	t.Run("rewrites to target org", func(t *testing.T) {
		got := string(rewriteFCAFOrganization(input, "acme-org"))
		require.Contains(t, got, "global_device_id: acme-org/usb/device")
		require.Contains(t, got, "action_id: acme-org/eudiw-beta-wallet/onboarding-1")
		require.NotContains(t, got, "forkbomb-bv-andrea/")
	})

	t.Run("keeps source org unchanged", func(t *testing.T) {
		require.Equal(t, string(input), string(rewriteFCAFOrganization(input, fcafSourceOrg)))
	})

	t.Run("empty target org is a no-op", func(t *testing.T) {
		require.Equal(t, string(input), string(rewriteFCAFOrganization(input, "")))
	})
}

func TestWalkFCAFImports(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(root, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	write("bundle/credentials/issuer/cred.yaml", "yaml: x\n")
	write("bundle/wallet/action.yaml", "yaml: y\n")
	write("bundle/credential-issuers/issuer.yaml", "yaml: z\n")

	paths, err := walkFCAFImports(root, "/credentials/")
	require.NoError(t, err)
	require.Len(t, paths, 1)
	require.Contains(t, paths[0], "credentials")
}
