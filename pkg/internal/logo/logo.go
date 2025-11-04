// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package logo

import (
	"log"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func LogoHooks(app core.App) {
	app.OnRecordCreate().BindFunc(HandleLogo)
	app.OnRecordUpdate().BindFunc(HandleLogo)
}

func HandleLogo(e *core.RecordEvent) error {
	logoURL := e.Record.GetString("logo_url")
	if logoURL != "" {
		file, err := filesystem.NewFileFromURL(e.Context, logoURL)
		if err != nil {
			log.Println("Error during the logo download:", err)
			return e.Next()
		}
		e.Record.Set("logo", []*filesystem.File{file})
	}
	return e.Next()
}
