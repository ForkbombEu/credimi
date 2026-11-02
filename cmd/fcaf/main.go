// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Command fcaf runs remote FCAF commands without starting PocketBase.
package main

import (
	"log"

	"github.com/forkbombeu/credimi/cmd/cli"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	if err := cli.NewFCAFCommand().Execute(); err != nil {
		log.Fatal(err)
	}
}
