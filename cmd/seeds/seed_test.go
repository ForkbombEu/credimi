// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const testDataDir = "../../test_pb_data"

func TestSeedSkipsMissingCollection(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	seedPath := filepath.Join(t.TempDir(), "data.json")
	seedPayload := []byte(`[{"collection":"missing_collection","records":[{"name":"demo"}]}]`)
	require.NoError(t, os.WriteFile(seedPath, seedPayload, 0600))

	err = seed(app, seedPath)
	require.NoError(t, err)
}

func TestSeedRejectsInvalidJSON(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	seedPath := filepath.Join(t.TempDir(), "data.json")
	require.NoError(t, os.WriteFile(seedPath, []byte("{"), 0600))

	err = seed(app, seedPath)
	require.Error(t, err)
}
