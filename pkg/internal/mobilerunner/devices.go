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

const mobileDevicesCollection = "mobile_devices"

var (
	ErrDeviceNotFound       = errors.New("mobile device not found")
	ErrDeviceRunnerNotFound = errors.New("mobile device runner not found")
	ErrDeviceNotAccessible  = errors.New("mobile device is not accessible")
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

// ValidateDeviceAccess checks that every device exists, has a runner and is
// either owned by ownerID or shared with that organization.
func ValidateDeviceAccess(app core.App, ownerID string, deviceIDs []string) error {
	ownerPublished := OrganizationPublishedLoader(app, ownerID)
	for _, deviceID := range NormalizeDeviceIDs(deviceIDs) {
		record, err := canonify.Resolve(app, deviceID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: mobile device %s was not found", ErrDeviceNotFound, deviceID)
			}
			return fmt.Errorf("resolve mobile device %s: %w", deviceID, err)
		}
		if record.Collection() == nil || record.Collection().Name != mobileDevicesCollection {
			return fmt.Errorf("%w: mobile device %s was not found", ErrDeviceNotFound, deviceID)
		}
		runner, err := app.FindRecordById("mobile_runners", record.GetString("runner"))
		if err != nil {
			return fmt.Errorf("%w: %w", ErrDeviceRunnerNotFound, err)
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
