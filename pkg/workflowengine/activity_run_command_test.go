// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBaseActivityNewMissingOrInvalidPayloadError(t *testing.T) {
	activity := BaseActivity{Name: "Example"}
	err := activity.NewMissingOrInvalidPayloadError(assertError("invalid payload"))

	require.ErrorContains(t, err, "CRE202")
	require.ErrorContains(t, err, "Missing or invalid value in payload")
}

func TestRunCommandWithCancellation(t *testing.T) {
	t.Run("returns start error", func(t *testing.T) {
		cmd := exec.Command("command-that-does-not-exist")
		err := RunCommandWithCancellation(context.Background(), cmd, time.Second)
		require.Error(t, err)
	})

	t.Run("returns nil on successful completion", func(t *testing.T) {
		cmd := exec.Command("bash", "-lc", "exit 0")
		err := RunCommandWithCancellation(context.Background(), cmd, time.Second)
		require.NoError(t, err)
	})

	t.Run("returns context error on cancellation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
		defer cancel()

		cmd := exec.Command("bash", "-lc", "sleep 1")
		err := RunCommandWithCancellation(ctx, cmd, time.Second)
		require.Error(t, err)
		require.ErrorContains(t, err, context.DeadlineExceeded.Error())
	})
}

type assertError string

func (e assertError) Error() string { return string(e) }
