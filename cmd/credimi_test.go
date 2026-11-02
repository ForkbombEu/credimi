// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemoteCommandsDoNotBootstrapPocketBase(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		wantErr string
	}{
		{name: "fcaf sync", args: "fcaf sync", wantErr: "CREDIMI_API_KEY is required"},
		{name: "fcaf run", args: "fcaf run", wantErr: "CREDIMI_API_KEY is required"},
		{name: "pipeline", args: "pipeline", wantErr: `required flag(s) "api-key" not set`},
		{
			name:    "pipeline store",
			args:    "pipeline store",
			wantErr: `required flag(s) "api-key" not set`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestRemoteCommandHelperProcess$")
			cmd.Dir = t.TempDir()
			cmd.Env = append(
				os.Environ(),
				"CREDIMI_CLI_HELPER_PROCESS=1",
				"CREDIMI_CLI_HELPER_ARGS="+tt.args,
				"CREDIMI_API_KEY=",
			)

			output, err := cmd.CombinedOutput()
			var exitErr *exec.ExitError
			require.True(t, errors.As(err, &exitErr), "command error: %v\n%s", err, output)
			require.Equal(t, 1, exitErr.ExitCode(), string(output))
			require.Contains(t, string(output), tt.wantErr)
			require.NoDirExists(t, filepath.Join(cmd.Dir, "pb_data"))
		})
	}
}

func TestRemoteCommandHelperProcess(_ *testing.T) {
	if os.Getenv("CREDIMI_CLI_HELPER_PROCESS") != "1" {
		return
	}

	os.Args = append([]string{"credimi"}, strings.Fields(os.Getenv("CREDIMI_CLI_HELPER_ARGS"))...)
	Start()
}
