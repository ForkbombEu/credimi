// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package cmd

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "credimi-cli",
	Short: "CLI tool to interact with PocketBase credimi app",
}

func Execute() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
