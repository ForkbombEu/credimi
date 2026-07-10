// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"time"

	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine/hooks"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/types/known/durationpb"
)

type organizationPublicationCollection struct {
	Collection  string
	OwnerField  string
	PublicField string
}

var (
	newNamespaceClient          = client.NewNamespaceClient
	waitForNamespaceReadyFn     = waitForNamespaceReady
	startWorkersByNamespaceFn   = hooks.StartAllWorkersByNamespace
	ensureNamespaceAndWorkersFn = ensureNamespaceAndWorkers
	startWorkerManagerFn        = hooks.StartWorkerManagerWorkflow
	adminRunnerURLsFn           = hooks.WorkerManagerAdminRunnerURLs
)

var organizationPublicationCollections = []organizationPublicationCollection{
	{Collection: "wallets", OwnerField: "owner", PublicField: "published"},
	{Collection: "credential_issuers", OwnerField: "owner", PublicField: "published"},
	{Collection: "verifiers", OwnerField: "owner", PublicField: "published"},
	{Collection: "custom_checks", OwnerField: "owner", PublicField: "published"},
	{Collection: "pipelines", OwnerField: "owner", PublicField: "published"},
}

const defaultMaxPipelinesInQueue = 1

func HookOrganizations(app core.App) {
	registerOrganizationNamespaceHooks(app)
	registerOrganizationPublicationHooks(app)
	registerOrganizationWorkerManagerPublicationHooks(app)
	registerOrganizationProtectedFieldsHooks(app)
}

// HookNamespaceOrgs is kept as a compatibility wrapper for tests and existing call sites.
func HookNamespaceOrgs(app *pocketbase.PocketBase) {
	registerOrganizationNamespaceHooks(app)
}

// RegisterOrganizationPublicationHooks is kept as a compatibility wrapper for tests.
func RegisterOrganizationPublicationHooks(app core.App) {
	registerOrganizationPublicationHooks(app)
}

func registerOrganizationNamespaceHooks(app core.App) {
	app.OnRecordCreate("organizations").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetInt("max_pipelines_in_queue") == 0 {
			e.Record.Set("max_pipelines_in_queue", defaultMaxPipelinesInQueue)
		}
		return e.Next()
	})

	app.OnRecordAfterCreateSuccess("organizations").BindFunc(func(e *core.RecordEvent) error {
		orgName := e.Record.GetString("canonified_name")
		if orgName != "" {
			ensureNamespaceAndWorkersFn(orgName)
			runnerURLs, err := adminRunnerURLsFn(e.App)
			if err != nil {
				return err
			}
			startWorkerManagerFn(e.App, orgName, "", runnerURLs)
		}

		return e.Next()
	})

	app.OnRecordAfterUpdateSuccess("organizations").BindFunc(func(e *core.RecordEvent) error {
		oldName := e.Record.Original().GetString("canonified_name")
		newName := e.Record.GetString("canonified_name")

		if oldName == newName || newName == "" {
			return e.Next()
		}

		go hooks.StopAllWorkersByNamespace(oldName)

		ensureNamespaceAndWorkersFn(newName)
		runnerURLs, err := adminRunnerURLsFn(e.App)
		if err != nil {
			return err
		}
		startWorkerManagerFn(e.App, newName, oldName, runnerURLs)
		log.Printf("Moved workers from namespace %s to %s", oldName, newName)
		return e.Next()
	})
}

func registerOrganizationPublicationHooks(app core.App) {
	for _, collection := range organizationPublicationCollections {
		app.OnRecordAfterCreateSuccess(collection.Collection).BindFunc(func(e *core.RecordEvent) error {
			if err := syncOrganizationPublishedForRecord(e.App, e.Record); err != nil {
				return err
			}

			return e.Next()
		})

		app.OnRecordAfterUpdateSuccess(collection.Collection).BindFunc(func(e *core.RecordEvent) error {
			if err := syncOrganizationPublishedForRecord(e.App, e.Record); err != nil {
				return err
			}

			return e.Next()
		})

		app.OnRecordAfterDeleteSuccess(collection.Collection).BindFunc(func(e *core.RecordEvent) error {
			if err := syncOrganizationPublishedForRecord(e.App, e.Record); err != nil {
				return err
			}

			return e.Next()
		})
	}

	app.OnRecordUpdate("organizations").BindFunc(func(e *core.RecordEvent) error {
		if !e.Record.Original().GetBool("published") {
			return e.Next()
		}

		if e.Record.GetBool("published") {
			return e.Next()
		}

		hasPublicEntities, err := organizationHasPublicEntities(e.App, e.Record.Id)
		if err != nil {
			return err
		}

		if !hasPublicEntities {
			return e.Next()
		}

		return apis.NewBadRequestError(
			"Organization cannot be unpublished while it owns public records.",
			validation.Errors{
				"organization": validation.NewError(
					"validation_organization_public_entities",
					"unpublish all public records owned by this organization first",
				),
			},
		)
	})
}

func registerOrganizationProtectedFieldsHooks(app core.App) {
	app.OnRecordUpdateRequest("organizations").BindFunc(func(e *core.RecordRequestEvent) error {
		if e.HasSuperuserAuth() {
			return e.Next()
		}

		original := e.Record.Original()
		if original == nil {
			return e.Next()
		}

		if e.Record.GetInt("max_pipelines_in_queue") != original.GetInt("max_pipelines_in_queue") {
			e.Record.Set("max_pipelines_in_queue", original.GetInt("max_pipelines_in_queue"))
		}

		if e.Record.GetBool("published") != original.GetBool("published") {
			e.Record.Set("published", original.GetBool("published"))
		}

		return e.Next()
	})
}

func registerOrganizationWorkerManagerPublicationHooks(app core.App) {
	app.OnRecordAfterUpdateSuccess("organizations").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.Original().GetBool("published") || !e.Record.GetBool("published") {
			return e.Next()
		}

		namespace := e.Record.GetString("canonified_name")
		if namespace == "" {
			return e.Next()
		}

		runnerURLs, err := hooks.WorkerManagerPublishedNonAdminRunnerURLs(e.App)
		if err != nil {
			return err
		}

		startWorkerManagerFn(e.App, namespace, "", runnerURLs)
		return e.Next()
	})
}

func syncOrganizationPublishedForRecord(app core.App, record *core.Record) error {
	collection := organizationPublicationCollectionByName(record.Collection())
	if collection == nil {
		return nil
	}

	ownerID := record.GetString(collection.OwnerField)
	if ownerID != "" {
		if err := syncOrganizationPublished(app, ownerID); err != nil {
			return err
		}
	}

	original := record.Original()
	if original == nil {
		return nil
	}

	originalOwnerID := original.GetString(collection.OwnerField)
	if originalOwnerID == "" || originalOwnerID == ownerID {
		return nil
	}

	return syncOrganizationPublished(app, originalOwnerID)
}

func syncOrganizationPublished(app core.App, ownerID string) error {
	org, err := app.FindRecordById("organizations", ownerID)
	if err != nil {
		return err
	}

	hasPublicEntities, err := organizationHasPublicEntities(app, ownerID)
	if err != nil {
		return err
	}

	if org.GetBool("published") == hasPublicEntities {
		return nil
	}

	org.Set("published", hasPublicEntities)
	return app.Save(org)
}

func organizationHasPublicEntities(app core.App, ownerID string) (bool, error) {
	for _, collection := range organizationPublicationCollections {
		_, err := app.FindFirstRecordByFilter(
			collection.Collection,
			collection.OwnerField+"={:owner} && "+collection.PublicField+"=true",
			dbx.Params{"owner": ownerID},
		)
		if err == nil {
			return true, nil
		}
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}

		return false, err
	}

	return false, nil
}

func organizationPublicationCollectionByName(
	collection *core.Collection,
) *organizationPublicationCollection {
	if collection == nil {
		return nil
	}

	for i := range organizationPublicationCollections {
		if organizationPublicationCollections[i].Collection == collection.Name {
			return &organizationPublicationCollections[i]
		}
	}

	return nil
}

// ensureNamespaceAndWorkers ensures the given namespace exists in Temporal.
// If not, it creates it.
// It then starts all workers for that namespace in a goroutine.
func ensureNamespaceAndWorkers(namespace string) {
	hostPort := utils.GetEnvironmentVariable("TEMPORAL_ADDRESS", client.DefaultHostPort)
	c, err := newNamespaceClient(client.Options{
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
			err = c.Register(context.Background(), &workflowservice.RegisterNamespaceRequest{
				Namespace:                        namespace,
				WorkflowExecutionRetentionPeriod: durationpb.New(365 * 24 * time.Hour),
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
	if err := waitForNamespaceReadyFn(c, namespace, 90*time.Second); err != nil {
		log.Printf("Namespace %s not ready after retries: %v", namespace, err)
		return
	}

	go startWorkersByNamespaceFn(namespace)
}

func waitForNamespaceReady(
	c client.NamespaceClient,
	namespace string,
	timeout time.Duration,
) error {
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
		log.Printf(
			"Waiting %v before retrying namespace readiness (attempt %d)...",
			backoff,
			attempt,
		)
		time.Sleep(backoff)
	}
}
