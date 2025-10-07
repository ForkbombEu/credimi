// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine/hooks"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/types/known/durationpb"
)

// HookNamespaceOrgs sets up hooks for the "organizations" collection.
// - After create → ensure namespace + start workers
// - After update → stop workers for old namespace, ensure namespace + start workers for new one
func HookNamespaceOrgs(app *pocketbase.PocketBase) {
	app.OnRecordAfterCreateSuccess("organizations").BindFunc(func(e *core.RecordEvent) error {
		orgName := e.Record.GetString("canonified_name")
		if orgName != "" {
			ensureNamespaceAndWorkers(orgName)
		}
		return e.Next()
	})

	app.OnRecordAfterUpdateSuccess("organizations").BindFunc(func(e *core.RecordEvent) error {
		oldName := e.Record.Original().GetString("canonified_name")
		newName := e.Record.GetString("canonified_name")

		if oldName == newName || newName == "" {
			return e.Next()
		}

		// Stop workers for old namespace
		go hooks.StopAllWorkersByNamespace(oldName)

		ensureNamespaceAndWorkers(newName)

		log.Printf("Moved workers from namespace %s to %s", oldName, newName)
		return e.Next()
	})
}

// ensureNamespaceAndWorkers ensures the given namespace exists in Temporal.
// If not, it creates it with a retention period of 7 days.
// It then starts all workers for that namespace in a goroutine.
func ensureNamespaceAndWorkers(namespace string) {
	hostPort := utils.GetEnvironmentVariable("TEMPORAL_ADDRESS", client.DefaultHostPort)
	c, err := client.NewNamespaceClient(client.Options{
		HostPort: hostPort,
		ConnectionOptions: client.ConnectionOptions{
			TLS: nil,
		},
	})
	if err != nil {
		log.Printf("Unable to create namespace client: %v", err)
		return
	}
	defer c.Close()

	var created bool

	_, err = c.Describe(context.Background(), namespace)
	if err != nil {
		var notFound *serviceerror.NamespaceNotFound
		if errors.As(err, &notFound) {
			// Register the new namespace
			err = c.Register(context.Background(), &workflowservice.RegisterNamespaceRequest{
				Namespace:                        namespace,
				WorkflowExecutionRetentionPeriod: durationpb.New(7 * 24 * time.Hour),
			})
			if err != nil {
				log.Printf("Unable to create namespace %s: %v", namespace, err)
				return
			}
			log.Printf("Created namespace %s", namespace)
			created = true
		}
	}
	if !created {
		return
	}
	if err := waitForNamespaceReady(c, namespace, 90*time.Second); err != nil {
		log.Printf("Namespace %s not ready after retries: %v", namespace, err)
		return
	}

	go hooks.StartAllWorkersByNamespace(namespace)
}
func waitForNamespaceReady(c client.NamespaceClient, namespace string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	attempt := 0

	for {
		attempt++
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err := c.Describe(ctx, namespace)
		cancel()

		if err == nil {
			log.Printf("Namespace %q ready after %d attempt(s)", namespace, attempt)
			return nil
		}

		if time.Now().After(deadline) {
			return err
		}

		backoff := time.Duration(attempt) * time.Second
		if backoff > 5*time.Second {
			backoff = 5 * time.Second
		}
		log.Printf("Waiting %v before retrying namespace readiness (attempt %d)...", backoff, attempt)
		time.Sleep(backoff)
	}
}
