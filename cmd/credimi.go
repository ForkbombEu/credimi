// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package cmd credimi is your companion tool for be compliant with your SSI system.
package cmd

import (
	"context"
	"log"
	"os"
	"strconv"

	// Blank import to initialize database migrations
	"github.com/forkbombeu/credimi/cmd/cli"
	"github.com/forkbombeu/credimi/internal/telemetry"
	_ "github.com/forkbombeu/credimi/migrations"
	"github.com/forkbombeu/credimi/pkg/routes"
	"github.com/joho/godotenv"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"
)

// Start initializes and starts the PocketBase application.
func Start() {
	app := pocketbase.New()
	app.RootCmd.Short = "\033[38;2;255;100;0m .o88b. d8888b. d88888b d8888b. d888888b .88b  d88. d888888b \033[0m\n" +
		"\033[38;2;255;71;43md8P  Y8 88  `8D 88'     88  `8D   `88'   88'YbdP`88   `88'   \033[0m\n" +
		"\033[38;2;255;43;86m8P      88oobY' 88ooooo 88   88    88    88  88  88    88    \033[0m\n" +
		"\033[38;2;255;14;129m8b      88`8b   88~~~~~ 88   88    88    88  88  88    88    \033[0m\n" +
		"\033[38;2;236;0;157mY8b  d8 88 `88. 88.     88  .8D   .88.   88  88  88   .88.   \033[0m\n" +
		"\033[38;2;197;0;171m `Y88P' 88   YD Y88888P Y8888D' Y888888P YP  YP  YP Y888888P \033[0m\n" +
		"\033[38;2;159;0;186m                                                             \033[0m\n" +
		"                                 \033[48;2;0;0;139m\033[38;2;255;255;255m              :(){ :|:& };: \033[0m\n" +
		"                                 \033[48;2;0;0;139m\033[38;2;255;255;255m with ❤ by Forkbomb hackers \033[0m\n"

	poolMax := app.RootCmd.PersistentFlags().
		Int("pool-max", 0, "Max concurrent AVDs (overrides AVD_POOL_MAX_CONCURRENT)")
	poolQueueDepth := app.RootCmd.PersistentFlags().
		Int("pool-queue-depth", 0, "Max AVD pool queue depth (overrides AVD_POOL_MAX_QUEUE)")

	existingPreRun := app.RootCmd.PersistentPreRun
	app.RootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if existingPreRun != nil {
			existingPreRun(cmd, args)
		}
		if poolMax != nil && *poolMax > 0 {
			os.Setenv("AVD_POOL_MAX_CONCURRENT", strconv.Itoa(*poolMax))
		}
		if poolQueueDepth != nil && *poolQueueDepth > 0 {
			os.Setenv("AVD_POOL_MAX_QUEUE", strconv.Itoa(*poolQueueDepth))
		}
	}

	shutdownTracing, err := telemetry.SetupTracing(context.Background())
	if err != nil {
		log.Printf("Tracing initialization failed: %v", err)
	}
	if shutdownTracing != nil {
		app.OnTerminate().BindFunc(func(_ *core.TerminateEvent) error {
			return shutdownTracing(context.Background())
		})
	}

	routes.Setup(app)

	app.RootCmd.AddCommand(cli.NewPipelineCmd())
	app.RootCmd.AddCommand(cli.NewDebugCmd())
	app.RootCmd.AddCommand(cli.NewPoolCmd())
	app.RootCmd.AddCommand(cli.NewCleanupCmd())

	godotenv.Load()
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
