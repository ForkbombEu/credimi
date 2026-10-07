// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package mobilerunner

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/core"
)

const (
	mobileDevicesCollection = "mobile_devices"
	mobileRunnersCollection = "mobile_runners"
)

var (
	ErrDeviceNotFound       = errors.New("mobile device not found")
	ErrDeviceRunnerNotFound = errors.New("mobile device runner not found")
	ErrDeviceNotAccessible  = errors.New("mobile device is not accessible")
	ErrRunnerNotFound       = errors.New("mobile runner not found")
)

// RunnerURL returns the runner base URL built from its ip and optional port.
func RunnerURL(runner *core.Record) string {
	runnerURL := strings.TrimSpace(runner.GetString("ip"))
	if runnerURL == "" {
		return ""
	}
	if port := strings.TrimSpace(runner.GetString("port")); port != "" {
		runnerURL = strings.TrimRight(runnerURL, "/") + ":" + port
	}

	return runnerURL
}

// RunnerIdentifier returns the normalized canonified path of a runner.
func RunnerIdentifier(app core.App, runner *core.Record) (string, error) {
	runnerID, err := canonify.BuildPath(
		app,
		runner,
		canonify.CanonifyPaths["mobile_runners"],
		"",
	)
	if err != nil {
		return "", err
	}

	return canonify.NormalizePath(runnerID), nil
}

// NormalizeDeviceIDs splits comma lists, normalizes, deduplicates and sorts device IDs.
func NormalizeDeviceIDs(values []string) []string {
	unique := map[string]struct{}{}
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			candidate := canonify.NormalizePath(part)
			if candidate == "" {
				continue
			}
			unique[candidate] = struct{}{}
		}
	}
	out := make([]string, 0, len(unique))
	for candidate := range unique {
		out = append(out, candidate)
	}
	sort.Strings(out)
	return out
}

// SharedWith reports whether another organization's runner may be used by an
// organization: only published runners are shared, and unpublished
// organizations only get admin-managed ones.
func SharedWith(runner *core.Record, orgPublished bool) bool {
	return runner.GetBool("published") && (orgPublished || runner.GetBool("admin_managed"))
}

// OrganizationPublishedLoader returns a function that reads the organization's
// published flag on first use and reuses it afterwards.
func OrganizationPublishedLoader(app core.App, orgID string) func() (bool, error) {
	loaded := false
	published := false
	return func() (bool, error) {
		if loaded {
			return published, nil
		}
		org, err := app.FindRecordById("organizations", orgID)
		if err != nil {
			return false, err
		}
		loaded = true
		published = org.GetBool("published")
		return published, nil
	}
}

// ResolveRunner returns the mobile runner at the canonified path id. It
// returns an error wrapping ErrRunnerNotFound when id is not a mobile runner;
// any other error is a lookup failure.
func ResolveRunner(app core.App, id string) (*core.Record, error) {
	runner, err := canonify.Resolve(app, canonify.NormalizePath(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: %s", ErrRunnerNotFound, id)
		}
		return nil, fmt.Errorf("resolve mobile runner %s: %w", id, err)
	}
	if runner.Collection() == nil || runner.Collection().Name != mobileRunnersCollection {
		return nil, fmt.Errorf("%w: %s", ErrRunnerNotFound, id)
	}
	return runner, nil
}

// ResolveDevice returns the mobile device at the canonified path id and its
// runner. It returns an error wrapping ErrDeviceNotFound when id is not a
// mobile device and ErrDeviceRunnerNotFound when its runner record is missing;
// any other error is a lookup failure.
func ResolveDevice(app core.App, id string) (device, runner *core.Record, err error) {
	device, err = canonify.Resolve(app, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, fmt.Errorf("%w: mobile device %s was not found", ErrDeviceNotFound, id)
		}
		return nil, nil, fmt.Errorf("resolve mobile device %s: %w", id, err)
	}
	if device.Collection() == nil || device.Collection().Name != mobileDevicesCollection {
		return nil, nil, fmt.Errorf("%w: mobile device %s was not found", ErrDeviceNotFound, id)
	}
	runnerID := device.GetString("runner")
	runner, err = app.FindRecordById("mobile_runners", runnerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, fmt.Errorf(
				"%w: runner %s of mobile device %s",
				ErrDeviceRunnerNotFound,
				runnerID,
				id,
			)
		}
		return nil, nil, fmt.Errorf("find runner %s of mobile device %s: %w", runnerID, id, err)
	}
	return device, runner, nil
}

// ValidateDeviceAccess checks that every device exists, has a runner and is
// either owned by ownerID or shared with that organization.
func ValidateDeviceAccess(app core.App, ownerID string, deviceIDs []string) error {
	ownerPublished := OrganizationPublishedLoader(app, ownerID)
	for _, deviceID := range NormalizeDeviceIDs(deviceIDs) {
		record, runner, err := ResolveDevice(app, deviceID)
		if err != nil {
			return err
		}
		if record.GetString("owner") == ownerID {
			continue
		}
		orgPublished, err := ownerPublished()
		if err != nil {
			return fmt.Errorf("load organization %s: %w", ownerID, err)
		}
		if SharedWith(runner, orgPublished) {
			continue
		}
		return fmt.Errorf(
			"%w: mobile device %s is private and does not belong to the caller organization",
			ErrDeviceNotAccessible,
			deviceID,
		)
	}
	return nil
}
