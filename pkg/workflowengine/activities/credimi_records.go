// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
)

// errInvalidSecrets reports a record whose secrets YAML cannot be parsed.
var errInvalidSecrets = errors.New("invalid secrets yaml")

// ResolveRecordInput selects a record by canonified name for an organization.
type ResolveRecordInput struct {
	CanonifiedName string `json:"canonified_name" validate:"required"`
	Collection     string `json:"collection"      validate:"required,oneof=pipelines custom_checks wallet_actions"`
	OwnerNamespace string `json:"owner_namespace" validate:"required"`
}

// ResolveRecordActivity returns the fields of a published record, or of an
// unpublished record owned by the requesting organization.
type ResolveRecordActivity struct {
	workflowengine.BaseActivity
	app core.App
}

func NewResolveRecordActivity(app core.App) *ResolveRecordActivity {
	return &ResolveRecordActivity{
		BaseActivity: workflowengine.BaseActivity{Name: ResolveRecordActivityName},
		app:          app,
	}
}

func (a *ResolveRecordActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *ResolveRecordActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[ResolveRecordInput](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	record, err := resolveRecord(
		a.app,
		payload.CanonifiedName,
		payload.Collection,
		payload.OwnerNamespace,
	)
	if err != nil {
		return result, recordLookupError(&a.BaseActivity, err)
	}
	result.Output = record.FieldsData()
	return result, nil
}

// resolveRecord returns the record of collection at canonifiedName when it is
// published or owned by the organization whose canonified name is
// ownerNamespace. Any other record is reported as ErrRecordNotFound.
func resolveRecord(
	app core.App,
	canonifiedName string,
	collection string,
	ownerNamespace string,
) (*core.Record, error) {
	notFound := fmt.Errorf(
		"%w: %s record %s not found",
		errRecordNotFound,
		collection,
		canonifiedName,
	)
	record, err := canonify.Resolve(app, canonifiedName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notFound
		}
		return nil, fmt.Errorf("resolve %s record %s: %w", collection, canonifiedName, err)
	}
	if record.Collection().Name != collection {
		return nil, notFound
	}
	if record.GetBool("published") {
		return record, nil
	}
	owner, err := app.FindFirstRecordByFilter(
		"organizations",
		"canonified_name = {:namespace}",
		map[string]any{"namespace": strings.TrimSpace(ownerNamespace)},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notFound
		}
		return nil, fmt.Errorf("find organization %s: %w", ownerNamespace, err)
	}
	if record.GetString("owner") != owner.Id {
		return nil, notFound
	}
	return record, nil
}

// CredentialOfferOutput is either a static credential offer or the code of a
// dynamic credential.
type CredentialOfferOutput struct {
	CredentialOffer string `json:"credential_offer,omitempty"`
	Dynamic         bool   `json:"dynamic"`
	Code            string `json:"code,omitempty"`
}

type GetCredentialOfferInput struct {
	CredentialIdentifier string `json:"credential_identifier" validate:"required"`
}

// GetCredentialOfferActivity returns the credential offer of a credential.
type GetCredentialOfferActivity struct {
	workflowengine.BaseActivity
	app core.App
}

func NewGetCredentialOfferActivity(app core.App) *GetCredentialOfferActivity {
	return &GetCredentialOfferActivity{
		BaseActivity: workflowengine.BaseActivity{Name: GetCredentialOfferActivityName},
		app:          app,
	}
}

func (a *GetCredentialOfferActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *GetCredentialOfferActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[GetCredentialOfferInput](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	offer, secrets, err := credentialOffer(a.app, payload.CredentialIdentifier)
	if err != nil {
		if errors.Is(err, errInvalidSecrets) {
			return result, credimiActivityError(
				&a.BaseActivity,
				errorcodes.DecodeFailed,
				false,
				err,
			)
		}
		return result, recordLookupError(&a.BaseActivity, err)
	}
	result.Output = offer
	result.Secrets = secretsOutput(secrets)
	return result, nil
}

// credentialOffer returns the offer of the credential at identifier and, for
// dynamic credentials, their parsed secrets.
func credentialOffer(
	app core.App,
	identifier string,
) (CredentialOfferOutput, map[string]string, error) {
	var output CredentialOfferOutput
	record, err := resolveByIdentifier(app, "credential", identifier)
	if err != nil {
		return output, nil, err
	}

	if code := record.GetString("yaml"); code != "" {
		secrets, err := parseRecordSecrets(record)
		if err != nil {
			return output, nil, err
		}
		output.Dynamic = true
		output.Code = code
		return output, secrets, nil
	}

	if deeplink := record.GetString("deeplink"); deeplink != "" {
		output.CredentialOffer = deeplink
		return output, nil, nil
	}

	issuerID := record.GetString("credential_issuer")
	issuer, err := app.FindRecordById("credential_issuers", issuerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return output, nil, fmt.Errorf("%w: credential issuer %s", errRecordNotFound, issuerID)
		}
		return output, nil, fmt.Errorf("find credential issuer %s: %w", issuerID, err)
	}
	offer, err := json.Marshal(map[string]any{
		"credential_configuration_ids": []string{record.GetString("name")},
		"credential_issuer":            issuer.GetString("url"),
	})
	if err != nil {
		return output, nil, fmt.Errorf("marshal credential offer: %w", err)
	}
	output.CredentialOffer = "openid-credential-offer://?credential_offer=" +
		url.QueryEscape(string(offer))
	return output, nil, nil
}

// UseCaseVerificationDeeplinkOutput carries the code of a use case verification.
type UseCaseVerificationDeeplinkOutput struct {
	Code string `json:"code"`
}

type GetUseCaseVerificationDeeplinkInput struct {
	UseCaseIdentifier string `json:"use_case_identifier" validate:"required"`
}

// GetUseCaseVerificationDeeplinkActivity returns the code and secrets of a use
// case verification.
type GetUseCaseVerificationDeeplinkActivity struct {
	workflowengine.BaseActivity
	app core.App
}

func NewGetUseCaseVerificationDeeplinkActivity(
	app core.App,
) *GetUseCaseVerificationDeeplinkActivity {
	return &GetUseCaseVerificationDeeplinkActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: GetUseCaseVerificationDeeplinkActivityName,
		},
		app: app,
	}
}

func (a *GetUseCaseVerificationDeeplinkActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *GetUseCaseVerificationDeeplinkActivity) Execute(
	_ context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult
	payload, err := workflowengine.DecodePayload[GetUseCaseVerificationDeeplinkInput](
		input.Payload,
	)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}
	code, secrets, err := useCaseVerificationDeeplink(a.app, payload.UseCaseIdentifier)
	if err != nil {
		if errors.Is(err, errInvalidSecrets) {
			return result, credimiActivityError(
				&a.BaseActivity,
				errorcodes.DecodeFailed,
				false,
				err,
			)
		}
		return result, recordLookupError(&a.BaseActivity, err)
	}
	result.Output = UseCaseVerificationDeeplinkOutput{Code: code}
	result.Secrets = secretsOutput(secrets)
	return result, nil
}

// useCaseVerificationDeeplink returns the code and parsed secrets of the use
// case verification at identifier.
func useCaseVerificationDeeplink(
	app core.App,
	identifier string,
) (string, map[string]string, error) {
	record, err := resolveByIdentifier(app, "use case verification", identifier)
	if err != nil {
		return "", nil, err
	}
	secrets, err := parseRecordSecrets(record)
	if err != nil {
		return "", nil, err
	}
	return record.GetString("yaml"), secrets, nil
}

func resolveByIdentifier(app core.App, kind, identifier string) (*core.Record, error) {
	record, err := canonify.Resolve(app, identifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s %s", errRecordNotFound, kind, identifier)
		}
		return nil, fmt.Errorf("resolve %s %s: %w", kind, identifier, err)
	}
	return record, nil
}

func parseRecordSecrets(record *core.Record) (map[string]string, error) {
	secrets, err := utils.ParseSecretsYAML(record.GetString("secrets"))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errInvalidSecrets, err)
	}
	return secrets, nil
}

func secretsOutput(secrets map[string]string) map[string]any {
	if len(secrets) == 0 {
		return nil
	}
	out := make(map[string]any, len(secrets))
	for key, value := range secrets {
		out[key] = value
	}
	return out
}
