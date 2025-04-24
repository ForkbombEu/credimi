// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"context"
	"log"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"google.golang.org/protobuf/types/known/durationpb"

	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

// HookNamespaceOrgs sets up a hook for the "organizations" collection in the PocketBase application.
// This hook is triggered after a successful record creation event. It performs the following actions:
//
// 1. Creates a new NamespaceClient to interact with the namespace service.
// 2. Checks if a namespace with the same name as the created record already exists by calling the Describe method.
// 3. If the namespace does not exist, it registers a new namespace with a retention period of 7 days.
// 4. Logs an error if the namespace creation fails or logs a success message if the namespace is created successfully.
//
// Parameters:
// - app: A pointer to the PocketBase application instance.
//
// Note:
// - The function uses the `log.Fatalln` method to terminate the application if the NamespaceClient cannot be created.
// - The hook ensures that the namespace registration process does not block the continuation of the event by calling `e.Next()` at the end.
func HookNamespaceOrgs(app *pocketbase.PocketBase) {
	app.OnRecordAfterCreateSuccess("organizations").BindFunc(func(e *core.RecordEvent) error {
		c, err := client.NewNamespaceClient(client.Options{})
		if err != nil {
			log.Fatalln("Unable to create client", err)
		}
		defer c.Close()

		errDes, _ := c.Describe(context.Background(), e.Record.Get("name").(string))
		if errDes == nil {
			return e.Next()
		}

		err = c.Register(context.Background(), &workflowservice.RegisterNamespaceRequest{
			Namespace:                        e.Record.Get("name").(string),
			WorkflowExecutionRetentionPeriod: durationpb.New(7 * 24 * time.Hour),
		})

		if err != nil {
			log.Printf("Unable to create namespace %s: %v", e.Record.Get("name").(string), err)
		}

		log.Default().Printf("Namespace %s created", e.Record.Get("name").(string))
		return e.Next()
	})
}
