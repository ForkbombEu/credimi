// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
)

// errInvalidStoredCredential reports a stored credential whose JSON cannot be
// parsed to compare its display values.
var errInvalidStoredCredential = errors.New("invalid stored credential json")

type StoreCredentialIssuerInput struct {
	URL   string `json:"url"    validate:"required"`
	OrgID string `json:"org_id" validate:"required"`
	Name  string `json:"name"`
	Logo  string `json:"logo"`
}

type StoreCredentialIssuerOutput struct {
	ID string `json:"id"`
}

// StoreCredentialIssuerActivity creates or updates an imported credential
// issuer of an organization.
type StoreCredentialIssuerActivity struct {
	workflowengine.BaseActivity
	app core.App
}

func NewStoreCredentialIssuerActivity(app core.App) *StoreCredentialIssuerActivity {
	return &StoreCredentialIssuerActivity{
		BaseActivity: workflowengine.BaseActivity{Name: StoreCredentialIssuerActivityName},
		app:          app,
	}
}

func (a *StoreCredentialIssuerActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *StoreCredentialIssuerActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[StoreCredentialIssuerInput](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	record, err := storeCredentialIssuer(a.app, payload)
	if err != nil {
		return result, a.NewCodedError(errorcodes.DatabaseOperationFailed, true, err)
	}
	result.Output = StoreCredentialIssuerOutput{ID: record.Id}
	return result, nil
}

// storeCredentialIssuer upserts the credential issuer identified by URL and
// owner. A new issuer is marked imported; non-empty name and logo overwrite
// the stored ones.
func storeCredentialIssuer(app core.App, in StoreCredentialIssuerInput) (*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId("credential_issuers")
	if err != nil {
		return nil, fmt.Errorf("find credential_issuers collection: %w", err)
	}
	record, err := app.FindFirstRecordByFilter(
		collection,
		"url = {:url} && owner = {:owner}",
		map[string]any{"url": in.URL, "owner": in.OrgID},
	)
	if err != nil {
		record = core.NewRecord(collection)
		record.Set("url", in.URL)
		record.Set("owner", in.OrgID)
		record.Set("imported", true)
	}
	if in.Name != "" {
		record.Set("name", in.Name)
	}
	if in.Logo != "" {
		record.Set("logo_url", in.Logo)
	}
	if err := app.Save(record); err != nil {
		return nil, fmt.Errorf("save credential issuer: %w", err)
	}
	return record, nil
}

type StoreIssuerCredentialInput struct {
	IssuerID   string         `json:"issuer_id"  validate:"required"`
	CredKey    string         `json:"cred_key"   validate:"required"`
	Credential map[string]any `json:"credential"`
	Conformant bool           `json:"conformant"`
	OrgID      string         `json:"org_id"     validate:"required"`
}

type StoreIssuerCredentialOutput struct {
	Key string `json:"key"`
}

// StoreIssuerCredentialActivity creates or updates a credential extracted from
// a credential issuer.
type StoreIssuerCredentialActivity struct {
	workflowengine.BaseActivity
	app core.App
}

func NewStoreIssuerCredentialActivity(app core.App) *StoreIssuerCredentialActivity {
	return &StoreIssuerCredentialActivity{
		BaseActivity: workflowengine.BaseActivity{Name: StoreIssuerCredentialActivityName},
		app:          app,
	}
}

func (a *StoreIssuerCredentialActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *StoreIssuerCredentialActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[StoreIssuerCredentialInput](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	if err := storeIssuerCredential(a.app, payload); err != nil {
		if errors.Is(err, errInvalidStoredCredential) {
			return result, a.NewCodedError(
				errorcodes.MissingOrInvalidPayload,
				false,
				err,
			)
		}
		return result, a.NewCodedError(errorcodes.DatabaseOperationFailed, true, err)
	}
	result.Output = StoreIssuerCredentialOutput{Key: payload.CredKey}
	return result, nil
}

// storeIssuerCredential upserts the credential named in.CredKey of issuer
// in.IssuerID. On update, display name and logo are replaced only when they
// still match the values of the previously stored credential JSON.
func storeIssuerCredential(app core.App, in StoreIssuerCredentialInput) error {
	name, locale, logo, description := parseCredentialDisplay(in.Credential)
	format, _ := in.Credential["format"].(string)

	collection, err := app.FindCollectionByNameOrId("credentials")
	if err != nil {
		return fmt.Errorf("find credentials collection: %w", err)
	}
	record, err := app.FindFirstRecordByFilter(
		collection,
		"name = {:key} && credential_issuer = {:issuerID}",
		map[string]any{"key": in.CredKey, "issuerID": in.IssuerID},
	)
	if err != nil {
		record = core.NewRecord(collection)
		record.Set("display_name", name)
		record.Set("logo_url", logo)
		record.Set("imported", true)
	} else {
		var saved map[string]any
		if err := json.Unmarshal([]byte(record.GetString("json")), &saved); err != nil {
			return fmt.Errorf("%w: %w", errInvalidStoredCredential, err)
		}
		originalName, originalLogo := storedCredentialDisplay(saved)
		if record.GetString("display_name") == originalName {
			record.Set("display_name", name)
		}
		if record.GetString("logo_url") == originalLogo {
			record.Set("logo_url", logo)
		}
	}

	credJSON, err := json.Marshal(in.Credential)
	if err != nil {
		return fmt.Errorf("marshal credential: %w", err)
	}
	record.Set("format", format)
	record.Set("locale", locale)
	record.Set("description", description)
	record.Set("json", string(credJSON))
	record.Set("name", in.CredKey)
	record.Set("credential_issuer", in.IssuerID)
	record.Set("conformant", in.Conformant)
	record.Set("owner", in.OrgID)
	if err := app.Save(record); err != nil {
		return fmt.Errorf("save credential: %w", err)
	}
	return nil
}

func parseCredentialDisplay(cred map[string]any) (name, locale, logo, description string) {
	displayList := credentialDisplayList(cred)
	if len(displayList) == 0 {
		return
	}
	first, ok := displayList[0].(map[string]any)
	if !ok {
		return
	}
	name, _ = first["name"].(string)
	locale, _ = first["locale"].(string)
	description, _ = first["description"].(string)
	if logoMap, ok := first["logo"].(map[string]any); ok {
		if uri, ok := logoMap["uri"].(string); ok {
			logo = uri
		} else if urlValue, ok := logoMap["url"].(string); ok {
			logo = urlValue
		}
	}
	return
}

// storedCredentialDisplay returns the display name and logo URI of a stored
// credential's top-level display list.
func storedCredentialDisplay(saved map[string]any) (name, logo string) {
	displayList, ok := saved["display"].([]any)
	if !ok || len(displayList) == 0 {
		return
	}
	first, ok := displayList[0].(map[string]any)
	if !ok {
		return
	}
	name, _ = first["name"].(string)
	if displayLogo, ok := first["logo"].(map[string]any); ok {
		logo, _ = displayLogo["uri"].(string)
	}
	return
}

func credentialDisplayList(cred map[string]any) []any {
	if metadata, ok := cred["credential_metadata"].(map[string]any); ok {
		if displayList, ok := metadata["display"].([]any); ok {
			return displayList
		}
	}
	if displayList, ok := cred["display"].([]any); ok {
		return displayList
	}
	return nil
}
