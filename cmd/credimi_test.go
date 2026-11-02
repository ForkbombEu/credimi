// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFCAFCommandsDoNotBootstrapPocketBase(t *testing.T) {
	for _, subcommand := range []string{"sync", "run"} {
		t.Run(subcommand, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestFCAFHelperProcess$")
			cmd.Dir = t.TempDir()
			cmd.Env = append(
				os.Environ(),
				"CREDIMI_FCAF_HELPER_PROCESS=1",
				"CREDIMI_FCAF_HELPER_COMMAND="+subcommand,
				"CREDIMI_API_KEY=",
			)

			output, err := cmd.CombinedOutput()
			var exitErr *exec.ExitError
			require.True(t, errors.As(err, &exitErr), "FCAF command error: %v\n%s", err, output)
			require.Equal(t, 1, exitErr.ExitCode(), string(output))
			require.Contains(t, string(output), "CREDIMI_API_KEY is required")
			require.NoDirExists(t, filepath.Join(cmd.Dir, "pb_data"))
		})
	}
}

func TestFCAFHelperProcess(_ *testing.T) {
	if os.Getenv("CREDIMI_FCAF_HELPER_PROCESS") != "1" {
		return
	}

	os.Args = []string{"credimi", "fcaf", os.Getenv("CREDIMI_FCAF_HELPER_COMMAND")}
	Start()
}
