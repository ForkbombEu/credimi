// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/internal/routing"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

const mobileDevicesCollection = "mobile_devices"

var MobileRunnerRegistrationRoutes = routing.RouteGroup{
	BaseURL:                "/api/mobile-runner",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:         http.MethodPost,
			Path:           "/preview-id",
			Handler:        HandlePreviewMobileRunnerID,
			RequestSchema:  PreviewMobileRunnerIDRequest{},
			ResponseSchema: PreviewMobileRunnerIDResponse{},
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminOrAuth(),
			},
		},
		{
			Method:         http.MethodPost,
			Path:           "",
			Handler:        HandleUpsertMobileRunner,
			RequestSchema:  UpsertMobileRunnerRequest{},
			ResponseSchema: UpsertMobileRunnerResponse{},
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminOrAuth(),
			},
		},
	},
}

var MobileDeviceRegistrationRoutes = routing.RouteGroup{
	BaseURL:                "/api/mobile-device",
	AuthenticationRequired: false,
	Middlewares: []*hook.Handler[*core.RequestEvent]{
		{Func: middlewares.ErrorHandlingMiddleware},
	},
	Routes: []routing.RouteDefinition{
		{
			Method:         http.MethodPost,
			Path:           "/preview-id",
			Handler:        HandlePreviewMobileDeviceID,
			RequestSchema:  PreviewMobileDeviceIDRequest{},
			ResponseSchema: PreviewMobileDeviceIDResponse{},
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminOrAuth(),
			},
		},
		{
			Method:         http.MethodPost,
			Path:           "",
			Handler:        HandleUpsertMobileDevice,
			RequestSchema:  UpsertMobileDeviceRequest{},
			ResponseSchema: UpsertMobileDeviceResponse{},
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminOrAuth(),
			},
		},
		{
			Method:        http.MethodPost,
			Path:          "/reconcile",
			Handler:       HandleReconcileMobileDevices,
			RequestSchema: ReconcileMobileDevicesRequest{},
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminOrAuth(),
			},
		},
		{
			Method:        http.MethodDelete,
			Path:          "",
			Handler:       HandleDeleteMobileDevice,
			RequestSchema: DeleteMobileDeviceRequest{},
			Middlewares: []*hook.Handler[*core.RequestEvent]{
				middlewares.RequireInternalAdminOrAuth(),
			},
		},
	},
}

type PreviewMobileDeviceIDRequest struct {
	Organization string `json:"organization,omitempty"`
	RunnerID     string `json:"runner_id"              validate:"required"`
	Name         string `json:"name"                   validate:"required"`
}

type PreviewMobileDeviceIDResponse struct {
	RunnerID         string `json:"runner_id"`
	DeviceID         string `json:"device_id"`
	ExistingDeviceID string `json:"existing_device_id,omitempty"`
	CanonifiedName   string `json:"canonified_name"`
	Conflict         bool   `json:"conflict"`
}

type UpsertMobileDeviceRequest struct {
	Organization string `json:"organization,omitempty"`
	DeviceID     string `json:"device_id,omitempty"`
	RunnerID     string `json:"runner_id"              validate:"required"`
	Name         string `json:"name"                   validate:"required"`
	Description  string `json:"description,omitempty"`
	Type         string `json:"type"                   validate:"required"`
	Serial       string `json:"serial,omitempty"`
}

type DeleteMobileDeviceRequest struct {
	Organization string `json:"organization,omitempty"`
	RunnerID     string `json:"runner_id"              validate:"required"`
	DeviceID     string `json:"device_id"              validate:"required"`
}

// ReconcileMobileDevicesRequest declares the complete device inventory the
// runner has loaded. Records previously associated with the runner but absent
// from this list are stale and are removed.
type ReconcileMobileDevicesRequest struct {
	Organization string   `json:"organization,omitempty"`
	RunnerID     string   `json:"runner_id"              validate:"required"`
	DeviceIDs    []string `json:"device_ids"`
}

func HandleReconcileMobileDevices() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[ReconcileMobileDevicesRequest](e)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"mobile_device",
				"invalid_request",
				err.Error(),
			)
		}
		owner, apiErr := resolveMobileDeviceOwner(e.App, e.Auth, input.Organization, input.RunnerID)
		if apiErr != nil {
			return apiErr
		}
		runner, apiErr := resolveExistingMobileRunner(
			e.App,
			owner,
			canonify.NormalizePath(input.RunnerID),
		)
		if apiErr != nil {
			return apiErr
		}
		if runner == nil {
			return apierror.New(
				http.StatusNotFound,
				"runner_id",
				"runner_not_found",
				"runner_id does not reference a mobile runner",
			)
		}
		configured := make(map[string]struct{}, len(input.DeviceIDs))
		for _, deviceID := range input.DeviceIDs {
			configured[canonify.NormalizePath(deviceID)] = struct{}{}
		}
		records, findErr := e.App.FindRecordsByFilter(
			mobileDevicesCollection,
			"runner = {:runner}",
			"",
			0,
			0,
			dbx.Params{"runner": runner.Id},
		)
		if findErr != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"mobile_devices",
				"failed_to_list_mobile_devices",
				findErr.Error(),
			)
		}
		removed := make([]string, 0)
		for _, record := range records {
			deviceID, idErr := mobileDeviceIdentifier(e.App, record)
			if idErr != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"device_id",
					"failed_to_build_device_id",
					idErr.Error(),
				)
			}
			if _, present := configured[deviceID]; present {
				continue
			}
			if deleteErr := e.App.Delete(record); deleteErr != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"mobile_device",
					"failed_to_delete_mobile_device",
					deleteErr.Error(),
				)
			}
			removed = append(removed, deviceID)
		}
		return e.JSON(http.StatusOK, map[string]any{"removed_device_ids": removed})
	}
}

func HandleDeleteMobileDevice() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[DeleteMobileDeviceRequest](e)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"mobile_device",
				"invalid_request",
				err.Error(),
			)
		}
		owner, apiErr := resolveMobileDeviceOwner(e.App, e.Auth, input.Organization, input.RunnerID)
		if apiErr != nil {
			return apiErr
		}
		record, apiErr := resolveExistingMobileDevice(
			e.App,
			owner,
			canonify.NormalizePath(input.DeviceID),
		)
		if apiErr != nil {
			return apiErr
		}
		if record == nil {
			return apierror.New(
				http.StatusNotFound,
				"device_id",
				"device_not_found",
				"device_id does not reference a mobile device",
			)
		}
		if err := e.App.Delete(record); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"mobile_device",
				"failed_to_delete_mobile_device",
				err.Error(),
			)
		}
		return e.JSON(
			http.StatusOK,
			map[string]string{"device_id": canonify.NormalizePath(input.DeviceID)},
		)
	}
}

type UpsertMobileDeviceResponse struct {
	ID             string `json:"id"`
	RunnerID       string `json:"runner_id"`
	DeviceID       string `json:"device_id"`
	Name           string `json:"name"`
	CanonifiedName string `json:"canonified_name"`
	Description    string `json:"description,omitempty"`
	Type           string `json:"type"`
	Serial         string `json:"serial,omitempty"`
}

func HandlePreviewMobileDeviceID() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[PreviewMobileDeviceIDRequest](e)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"mobile_device",
				"invalid_request",
				err.Error(),
			)
		}
		owner, apiErr := resolveMobileDeviceOwner(e.App, e.Auth, input.Organization, input.RunnerID)
		if apiErr != nil {
			return apiErr
		}
		runner, apiErr := resolveExistingMobileRunner(
			e.App,
			owner,
			canonify.NormalizePath(input.RunnerID),
		)
		if apiErr != nil {
			return apiErr
		}
		var preview PreviewMobileDeviceIDResponse
		if runner == nil {
			preview, apiErr = previewPendingMobileDeviceIdentifier(
				owner,
				input.RunnerID,
				input.Name,
			)
		} else {
			preview, apiErr = previewMobileDeviceIdentifier(e.App, runner, input.Name)
		}
		if apiErr != nil {
			return apiErr
		}
		return e.JSON(http.StatusOK, preview)
	}
}

// previewPendingMobileDeviceIdentifier derives the child path during first-run
// setup. At this stage the runner has not been persisted yet, so there cannot
// be any sibling devices to disambiguate. Device creation still performs the
// authoritative uniqueness and ownership checks after the runner is saved.
func previewPendingMobileDeviceIdentifier(
	owner *core.Record,
	runnerID string,
	name string,
) (PreviewMobileDeviceIDResponse, *apierror.APIError) {
	normalizedRunnerID := canonify.NormalizePath(runnerID)
	organizationID := canonify.NormalizePath(owner.GetString("canonified_name"))
	prefix := organizationID + "/"
	if organizationID == "" || !strings.HasPrefix(normalizedRunnerID, prefix) ||
		strings.TrimPrefix(normalizedRunnerID, prefix) == "" ||
		strings.Contains(strings.TrimPrefix(normalizedRunnerID, prefix), "/") {
		return PreviewMobileDeviceIDResponse{}, apierror.New(
			http.StatusForbidden,
			"runner_id",
			"runner_id_owner_mismatch",
			"runner_id does not belong to the resolved organization",
		)
	}

	canonifiedName := canonify.CanonifyPlain(strings.TrimSpace(name))
	return PreviewMobileDeviceIDResponse{
		RunnerID:       normalizedRunnerID,
		DeviceID:       normalizedRunnerID + "/" + canonifiedName,
		CanonifiedName: canonifiedName,
	}, nil
}

func HandleUpsertMobileDevice() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[UpsertMobileDeviceRequest](e)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"mobile_device",
				"invalid_request",
				err.Error(),
			)
		}
		owner, apiErr := resolveMobileDeviceOwner(e.App, e.Auth, input.Organization, input.RunnerID)
		if apiErr != nil {
			return apiErr
		}
		runner, apiErr := resolveExistingMobileRunner(
			e.App,
			owner,
			canonify.NormalizePath(input.RunnerID),
		)
		if apiErr != nil {
			return apiErr
		}
		if runner == nil {
			return apierror.New(
				http.StatusNotFound,
				"runner_id",
				"runner_not_found",
				"runner_id does not reference a mobile runner",
			)
		}
		deviceID := canonify.NormalizePath(input.DeviceID)
		record, apiErr := resolveExistingMobileDevice(e.App, owner, deviceID)
		if apiErr != nil {
			return apiErr
		}
		if record == nil {
			preview, previewErr := previewMobileDeviceIdentifier(e.App, runner, input.Name)
			if previewErr != nil {
				return previewErr
			}
			if deviceID != "" && deviceID != preview.DeviceID {
				return apierror.New(
					http.StatusConflict,
					"device_id",
					"device_id_conflict",
					"device_id does not match the next available id",
				)
			}
			collection, err := e.App.FindCollectionByNameOrId(mobileDevicesCollection)
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"collection",
					"mobile_devices_not_found",
					err.Error(),
				)
			}
			record = core.NewRecord(collection)
			record.Set("owner", owner.Id)
			record.Set("runner", runner.Id)
		} else if record.GetString("runner") != runner.Id || strings.TrimSpace(record.GetString("name")) != strings.TrimSpace(input.Name) {
			return apierror.New(
				http.StatusConflict,
				"device_id",
				"device_identity_immutable",
				"device_id cannot be moved or renamed",
			)
		}
		record.Set("name", strings.TrimSpace(input.Name))
		if constraintErr := validateMobileDeviceConstraints(
			e.App,
			runner,
			record.Id,
			strings.TrimSpace(input.Name),
			strings.TrimSpace(input.Type),
			strings.TrimSpace(input.Serial),
		); constraintErr != nil {
			return constraintErr
		}
		record.Set("description", strings.TrimSpace(input.Description))
		record.Set("type", strings.TrimSpace(input.Type))
		record.Set("serial", strings.TrimSpace(input.Serial))
		if err := e.App.Save(record); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"mobile_device",
				"failed_to_save_mobile_device",
				err.Error(),
			)
		}
		resolvedID, err := mobileDeviceIdentifier(e.App, record)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"device_id",
				"failed_to_build_device_id",
				err.Error(),
			)
		}
		return e.JSON(
			http.StatusOK,
			UpsertMobileDeviceResponse{
				ID:             record.Id,
				RunnerID:       input.RunnerID,
				DeviceID:       resolvedID,
				Name:           record.GetString("name"),
				CanonifiedName: record.GetString("canonified_name"),
				Description:    record.GetString("description"),
				Type:           record.GetString("type"),
				Serial:         record.GetString("serial"),
			},
		)
	}
}

func validateMobileDeviceConstraints(
	app core.App,
	runner *core.Record,
	currentRecordID, deviceName, deviceType, serial string,
) *apierror.APIError {
	if deviceType != "android_emulator" && deviceType != "ios_simulator" &&
		deviceType != "android_phone" &&
		deviceType != "redroid" {
		return apierror.New(
			http.StatusBadRequest,
			"type",
			"invalid_device_type",
			"device type must be android_emulator, ios_simulator, android_phone, or redroid",
		)
	}
	records, err := app.FindRecordsByFilter(
		mobileDevicesCollection,
		"runner = {:runner}",
		"",
		0,
		0,
		dbx.Params{"runner": runner.Id},
	)
	if err != nil {
		return apierror.New(
			http.StatusInternalServerError,
			"mobile_devices",
			"failed_to_list_mobile_devices",
			err.Error(),
		)
	}
	for _, existing := range records {
		if existing.Id == currentRecordID {
			continue
		}
		if existing.GetString("type") == deviceType &&
			(deviceType == "android_emulator" || deviceType == "ios_simulator") {
			return apierror.New(
				http.StatusConflict,
				"type",
				"device_type_limit",
				fmt.Sprintf(
					"runner already has %s device %q; only one is allowed",
					deviceType,
					existing.GetString("name"),
				),
			)
		}
		if serial != "" && (deviceType == "android_phone" || deviceType == "redroid") &&
			(existing.GetString("type") == "android_phone" || existing.GetString("type") == "redroid") &&
			existing.GetString("serial") == serial {
			return apierror.New(
				http.StatusConflict,
				"serial",
				"device_serial_conflict",
				fmt.Sprintf(
					"serial %q is already registered for device %q; choose a different serial for device %q",
					serial,
					existing.GetString("name"),
					deviceName,
				),
			)
		}
	}
	return nil
}

type PreviewMobileRunnerIDRequest struct {
	Organization string `json:"organization,omitempty"`
	Name         string `json:"name"                   validate:"required"`
}

type PreviewMobileRunnerIDResponse struct {
	Organization     string `json:"organization"`
	CanonifiedName   string `json:"canonified_name"`
	RunnerID         string `json:"runner_id"`
	ExistingRunnerID string `json:"existing_runner_id,omitempty"`
	Conflict         bool   `json:"conflict"`
}

type UpsertMobileRunnerRequest struct {
	RunnerID     string `json:"runner_id,omitempty"`
	Organization string `json:"organization,omitempty"`
	Name         string `json:"name"                   validate:"required"`
	IP           string `json:"ip"                     validate:"required"`
	Description  string `json:"description,omitempty"`
	Type         string `json:"type,omitempty"`
	Port         string `json:"port,omitempty"`
	Serial       string `json:"serial,omitempty"`
	Published    *bool  `json:"published,omitempty"`
}

type UpsertMobileRunnerResponse struct {
	ID             string `json:"id"`
	Organization   string `json:"organization"`
	Name           string `json:"name"`
	CanonifiedName string `json:"canonified_name"`
	RunnerID       string `json:"runner_id"`
	IP             string `json:"ip"`
	Description    string `json:"description,omitempty"`
	Type           string `json:"type,omitempty"`
	Port           string `json:"port,omitempty"`
	Serial         string `json:"serial,omitempty"`
	Published      bool   `json:"published"`
	AdminManaged   bool   `json:"admin_managed"`
}

func HandlePreviewMobileRunnerID() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[PreviewMobileRunnerIDRequest](e)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"mobile_runner",
				"invalid_request",
				err.Error(),
			)
		}

		owner, apiErr := resolveMobileRunnerOwner(e.App, e.Auth, input.Organization)
		if apiErr != nil {
			return apiErr
		}

		preview, apiErr := previewMobileRunnerIdentifier(e.App, owner, input.Name)
		if apiErr != nil {
			return apiErr
		}

		return e.JSON(http.StatusOK, preview)
	}
}

func HandleUpsertMobileRunner() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		input, err := routing.GetValidatedInput[UpsertMobileRunnerRequest](e)
		if err != nil {
			return apierror.New(
				http.StatusBadRequest,
				"mobile_runner",
				"invalid_request",
				err.Error(),
			)
		}

		owner, apiErr := resolveMobileRunnerOwner(e.App, e.Auth, input.Organization)
		if apiErr != nil {
			return apiErr
		}

		normalizedRunnerID := canonify.NormalizePath(input.RunnerID)
		record, apiErr := resolveExistingMobileRunner(e.App, owner, normalizedRunnerID)
		if apiErr != nil {
			return apiErr
		}
		creating := record == nil

		if normalizedRunnerID != "" && record == nil {
			preview, previewErr := previewMobileRunnerIdentifier(e.App, owner, input.Name)
			if previewErr != nil {
				return previewErr
			}
			if preview.RunnerID != normalizedRunnerID {
				return apierror.New(
					http.StatusConflict,
					"runner_id",
					"runner_id_conflict",
					fmt.Sprintf(
						"runner_id %q does not match the next available id %q",
						normalizedRunnerID,
						preview.RunnerID,
					),
				)
			}
		}

		if record != nil &&
			strings.TrimSpace(record.GetString("name")) != strings.TrimSpace(input.Name) {
			return apierror.New(
				http.StatusConflict,
				"name",
				"runner_name_conflict",
				"name does not match the existing runner_id",
			)
		}

		if record == nil {
			collection, err := e.App.FindCollectionByNameOrId("mobile_runners")
			if err != nil {
				return apierror.New(
					http.StatusInternalServerError,
					"collection",
					"mobile_runners collection not found",
					err.Error(),
				)
			}
			record = core.NewRecord(collection)
			record.Set("owner", owner.Id)
		}
		if creating && isSuperuserAuth(e.Auth) {
			record.Set("admin_managed", true)
		}

		record.Set("name", strings.TrimSpace(input.Name))
		record.Set("ip", strings.TrimSpace(input.IP))
		record.Set("description", strings.TrimSpace(input.Description))
		record.Set("type", strings.TrimSpace(input.Type))
		record.Set("port", strings.TrimSpace(input.Port))
		record.Set("serial", strings.TrimSpace(input.Serial))
		if input.Published != nil {
			record.Set("published", *input.Published)
		}

		if err := e.App.Save(record); err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"mobile_runner",
				"failed_to_save_mobile_runner",
				err.Error(),
			)
		}

		runnerID, err := mobileRunnerIdentifier(e.App, record)
		if err != nil {
			return apierror.New(
				http.StatusInternalServerError,
				"mobile_runner",
				"failed_to_build_runner_id",
				err.Error(),
			)
		}

		return e.JSON(http.StatusOK, UpsertMobileRunnerResponse{
			ID:             record.Id,
			Organization:   owner.GetString("canonified_name"),
			Name:           record.GetString("name"),
			CanonifiedName: record.GetString("canonified_name"),
			RunnerID:       runnerID,
			IP:             record.GetString("ip"),
			Description:    record.GetString("description"),
			Type:           record.GetString("type"),
			Port:           record.GetString("port"),
			Serial:         record.GetString("serial"),
			Published:      record.GetBool("published"),
			AdminManaged:   record.GetBool("admin_managed"),
		})
	}
}

func resolveMobileRunnerOwner(
	app core.App,
	auth *core.Record,
	requestedOrganization string,
) (*core.Record, *apierror.APIError) {
	if auth == nil {
		return nil, apierror.New(
			http.StatusUnauthorized,
			"auth",
			"authentication_required",
			"authentication is required",
		)
	}

	if isSuperuserAuth(auth) {
		orgCanon := strings.TrimSpace(requestedOrganization)
		if orgCanon == "" {
			return nil, apierror.New(
				http.StatusBadRequest,
				"organization",
				"organization_required",
				"organization is required for admin authentication",
			)
		}

		record, err := app.FindFirstRecordByFilter(
			"organizations",
			"canonified_name={:canonified_name}",
			dbx.Params{"canonified_name": orgCanon},
		)
		if err != nil {
			status := http.StatusInternalServerError
			reason := "failed_to_find_organization"
			message := err.Error()
			if errors.Is(err, sql.ErrNoRows) {
				status = http.StatusNotFound
				reason = "organization_not_found"
				message = "organization not found"
			}
			return nil, apierror.New(status, "organization", reason, message)
		}

		return record, nil
	}

	record, err := pbutils.GetUserOrganization(app, auth.Id)
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"organization",
			"failed_to_find_user_organization",
			err.Error(),
		)
	}

	return record, nil
}

// resolveMobileDeviceOwner derives the organization from a canonical runner
// identity for superuser device registration. The runner is the device's
// immutable parent, so accepting a conflicting organization would be unsafe.
func resolveMobileDeviceOwner(
	app core.App,
	auth *core.Record,
	requestedOrganization string,
	runnerID string,
) (*core.Record, *apierror.APIError) {
	if !isSuperuserAuth(auth) || strings.TrimSpace(requestedOrganization) != "" {
		return resolveMobileRunnerOwner(app, auth, requestedOrganization)
	}

	runner, err := canonify.Resolve(app, canonify.NormalizePath(runnerID))
	if err != nil || runner.Collection() == nil || runner.Collection().Name != "mobile_runners" {
		return nil, apierror.New(
			http.StatusNotFound,
			"runner_id",
			"runner_not_found",
			"runner_id does not reference a mobile runner",
		)
	}
	owner, err := app.FindRecordById("organizations", runner.GetString("owner"))
	if err != nil {
		return nil, apierror.New(
			http.StatusInternalServerError,
			"organization",
			"failed_to_find_organization",
			err.Error(),
		)
	}
	return owner, nil
}

func isSuperuserAuth(auth *core.Record) bool {
	if auth == nil || auth.Collection() == nil {
		return false
	}

	return auth.Collection().Name == "_superusers"
}

func previewMobileRunnerIdentifier(
	app core.App,
	owner *core.Record,
	name string,
) (PreviewMobileRunnerIDResponse, *apierror.APIError) {
	collection, err := app.FindCollectionByNameOrId("mobile_runners")
	if err != nil {
		return PreviewMobileRunnerIDResponse{}, apierror.New(
			http.StatusInternalServerError,
			"collection",
			"mobile_runners collection not found",
			err.Error(),
		)
	}

	record := core.NewRecord(collection)
	record.Set("owner", owner.Id)
	record.Set("name", strings.TrimSpace(name))

	canonifiedName, err := canonify.Canonify(
		record.GetString("name"),
		canonify.MakeExistsFunc(app, "mobile_runners", record, ""),
	)
	if err != nil {
		return PreviewMobileRunnerIDResponse{}, apierror.New(
			http.StatusInternalServerError,
			"name",
			"failed_to_canonify_runner_name",
			err.Error(),
		)
	}

	record.Set("canonified_name", canonifiedName)
	runnerID, err := mobileRunnerIdentifier(app, record)
	if err != nil {
		return PreviewMobileRunnerIDResponse{}, apierror.New(
			http.StatusInternalServerError,
			"runner_id",
			"failed_to_build_runner_id",
			err.Error(),
		)
	}

	baseRunnerID := canonify.NormalizePath(
		owner.GetString("canonified_name") + "/" + canonify.CanonifyPlain(strings.TrimSpace(name)),
	)
	response := PreviewMobileRunnerIDResponse{
		Organization:   owner.GetString("canonified_name"),
		CanonifiedName: canonifiedName,
		RunnerID:       runnerID,
		Conflict:       baseRunnerID != runnerID,
	}
	if response.Conflict {
		response.ExistingRunnerID = baseRunnerID
	}
	return response, nil
}

func resolveExistingMobileRunner(
	app core.App,
	owner *core.Record,
	runnerID string,
) (*core.Record, *apierror.APIError) {
	if runnerID == "" {
		return nil, nil
	}

	record, err := canonify.Resolve(app, runnerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, apierror.New(
			http.StatusInternalServerError,
			"runner_id",
			"failed_to_resolve_runner_id",
			err.Error(),
		)
	}

	if record.Collection() == nil || record.Collection().Name != "mobile_runners" {
		return nil, apierror.New(
			http.StatusBadRequest,
			"runner_id",
			"invalid_runner_id",
			"runner_id does not reference a mobile runner",
		)
	}

	if record.GetString("owner") != owner.Id {
		return nil, apierror.New(
			http.StatusForbidden,
			"runner_id",
			"runner_id_owner_mismatch",
			"runner_id does not belong to the resolved organization",
		)
	}

	return record, nil
}

func previewMobileDeviceIdentifier(
	app core.App,
	runner *core.Record,
	name string,
) (PreviewMobileDeviceIDResponse, *apierror.APIError) {
	collection, err := app.FindCollectionByNameOrId(mobileDevicesCollection)
	if err != nil {
		return PreviewMobileDeviceIDResponse{}, apierror.New(
			http.StatusInternalServerError,
			"collection",
			"mobile_devices_not_found",
			err.Error(),
		)
	}
	record := core.NewRecord(collection)
	record.Set("runner", runner.Id)
	record.Set("name", strings.TrimSpace(name))
	canonifiedName, err := canonify.Canonify(
		record.GetString("name"),
		canonify.MakeExistsFunc(app, mobileDevicesCollection, record, ""),
	)
	if err != nil {
		return PreviewMobileDeviceIDResponse{}, apierror.New(
			http.StatusConflict,
			"name",
			"failed_to_canonify_device_name",
			err.Error(),
		)
	}
	record.Set("canonified_name", canonifiedName)
	deviceID, err := mobileDeviceIdentifier(app, record)
	if err != nil {
		return PreviewMobileDeviceIDResponse{}, apierror.New(
			http.StatusInternalServerError,
			"device_id",
			"failed_to_build_device_id",
			err.Error(),
		)
	}
	runnerID, _ := mobileRunnerIdentifier(app, runner)
	baseCanonifiedName := canonify.CanonifyPlain(strings.TrimSpace(name))
	baseDeviceID := canonify.NormalizePath(runnerID + "/" + baseCanonifiedName)
	response := PreviewMobileDeviceIDResponse{
		RunnerID:       runnerID,
		DeviceID:       deviceID,
		CanonifiedName: canonifiedName,
		Conflict:       baseDeviceID != deviceID,
	}
	if response.Conflict {
		response.ExistingDeviceID = baseDeviceID
	}
	return response, nil
}

func resolveExistingMobileDevice(
	app core.App,
	owner *core.Record,
	deviceID string,
) (*core.Record, *apierror.APIError) {
	if deviceID == "" {
		return nil, nil
	}
	record, err := canonify.Resolve(app, deviceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, apierror.New(
			http.StatusInternalServerError,
			"device_id",
			"failed_to_resolve_device_id",
			err.Error(),
		)
	}
	if record.Collection() == nil || record.Collection().Name != mobileDevicesCollection {
		return nil, apierror.New(
			http.StatusBadRequest,
			"device_id",
			"invalid_device_id",
			"device_id does not reference a mobile device",
		)
	}
	if record.GetString("owner") != owner.Id {
		return nil, apierror.New(
			http.StatusForbidden,
			"device_id",
			"device_id_owner_mismatch",
			"device_id does not belong to the resolved organization",
		)
	}
	return record, nil
}

func mobileDeviceIdentifier(app core.App, record *core.Record) (string, error) {
	deviceID, err := canonify.BuildPath(
		app,
		record,
		canonify.CanonifyPaths[mobileDevicesCollection],
		"",
	)
	if err != nil {
		return "", err
	}
	return canonify.NormalizePath(deviceID), nil
}
