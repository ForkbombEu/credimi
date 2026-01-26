// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

var cleanupForce bool
var cleanupAvdctlPath string

// NewCleanupCmd creates the "cleanup" command group.
func NewCleanupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Run cleanup utilities",
	}
	cmd.AddCommand(newCleanupOrphansCmd())
	return cmd
}

func newCleanupOrphansCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "orphans",
		Short: "Detect and optionally remove orphaned emulators",
		RunE: func(cmd *cobra.Command, _ []string) error {
			bin := cleanupAvdctlPath
			if bin == "" {
				bin = "avdctl"
			}
			path, err := exec.LookPath(bin)
			if err != nil {
				return fmt.Errorf("avdctl not found: %w", err)
			}

			args := []string{"cleanup"}
			if cleanupForce {
				args = append(args, "--force")
			} else {
				args = append(args, "--dry-run")
			}

			command := exec.CommandContext(cmd.Context(), path, args...)
			command.Stdout = cmd.OutOrStdout()
			command.Stderr = cmd.ErrOrStderr()
			if err := command.Run(); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&cleanupForce, "force", false, "delete orphans instead of dry run")
	cmd.Flags().StringVar(&cleanupAvdctlPath, "avdctl", "avdctl", "path to avdctl binary")
	return cmd
}
