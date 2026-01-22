// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"errors"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine/avdpool"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/spf13/cobra"
	"go.temporal.io/api/serviceerror"
)

var poolNamespace string

// NewPoolCmd creates the "pool" command group.
func NewPoolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pool",
		Short: "Manage the AVD pool",
	}
	cmd.PersistentFlags().StringVar(&poolNamespace, "namespace", "default", "Temporal namespace")

	cmd.AddCommand(newPoolStatusCmd())
	cmd.AddCommand(newPoolResetCmd())
	return cmd
}

func newPoolStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show pool status",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := temporalclient.GetTemporalClientWithNamespace(poolNamespace)
			if err != nil {
				return err
			}
			defer c.Close()

			ctx := cmd.Context()
			queryResp, err := c.QueryWorkflow(ctx, avdpool.DefaultPoolWorkflowID, "", avdpool.PoolStatusQuery)
			if err != nil {
				return err
			}
			var status avdpool.PoolStatus
			if err := queryResp.Get(&status); err != nil {
				return err
			}
			return printJSON(status)
		},
	}
	return cmd
}

func newPoolResetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Restart the pool manager workflow",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := temporalclient.GetTemporalClientWithNamespace(poolNamespace)
			if err != nil {
				return err
			}
			defer c.Close()

			ctx := cmd.Context()
			if err := c.TerminateWorkflow(ctx, avdpool.DefaultPoolWorkflowID, "", "operator reset"); err != nil {
				var notFound *serviceerror.NotFound
				if !errors.As(err, &notFound) {
					return fmt.Errorf("failed to terminate pool manager: %w", err)
				}
			}

			config := avdpool.ConfigFromEnv()
			result, err := avdpool.StartPoolManagerWorkflow(poolNamespace, pipeline.PipelineTaskQueue, config)
			if err != nil {
				return err
			}

			return printJSON(map[string]any{
				"status":   "restarted",
				"result":   result,
				"config":   config,
				"workflow": avdpool.DefaultPoolWorkflowID,
			})
		},
	}
	return cmd
}
