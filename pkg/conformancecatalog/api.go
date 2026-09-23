// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/search"
)

const (
	writeRejectMessage      = "conformance_checks is a read-only catalog projection of config_templates"
	suitesWriteRejectMessage = "conformance_suites is a read-only catalog projection of config_templates"
)

var catalogSelectColumns = []string{
	"id",
	"path",
	"title",
	"standard",
	"version",
	"suite",
	"file",
	"visible_in",
	"protocol",
	"sut",
	"role",
	"provider",
	"norm_standard",
	"component",
	"norm_version",
	"suite_name",
	"suite_homepage",
	"suite_repository",
	"suite_help",
	"suite_description",
	"suite_logo",
	"created",
	"updated",
}

var catalogSearchFields = catalogSelectColumns

var suiteSelectColumns = []string{
	"id",
	"standard",
	"component",
	"component_rank",
	"version",
	"suite",
	"provider",
	"suite_name",
	"suite_homepage",
	"suite_repository",
	"suite_help",
	"suite_description",
	"suite_logo",
	"check_count",
	"check_paths",
	"check_titles",
	"check_files",
	"visible_in",
	"fs_standard",
	"fs_version",
	"path_prefix",
	"created",
	"updated",
}

var suiteSearchFields = suiteSelectColumns

// RecordsListHTTP serves GET /api/collections/conformance_checks/records
// using PocketBase tools/search against the process-private :memory: catalog.
func RecordsListHTTP() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		db, err := catalogDB()
		if err != nil {
			return e.InternalServerError("conformance catalog unavailable", err)
		}

		rows := []*catalogRow{}
		resolver := search.NewSimpleFieldResolver(catalogSearchFields...)
		base := db.Select(catalogSelectColumns...).From(CollectionName)

		result, err := search.NewProvider(resolver).
			Query(base).
			CountCol("id").
			ParseAndExec(e.Request.URL.Query().Encode(), &rows)
		if err != nil {
			return firstSearchAPIError(e, err)
		}

		for _, row := range rows {
			row.withCollectionMeta()
		}
		result.Items = rows
		return e.JSON(http.StatusOK, result)
	}
}

// RecordViewHTTP serves GET /api/collections/conformance_checks/records/{id}.
func RecordViewHTTP() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.NotFoundError("", nil)
		}

		db, err := catalogDB()
		if err != nil {
			return e.InternalServerError("conformance catalog unavailable", err)
		}

		row := &catalogRow{}
		err = db.Select(catalogSelectColumns...).
			From(CollectionName).
			AndWhere(dbx.HashExp{"id": id}).
			One(row)
		if err != nil {
			return e.NotFoundError("", err)
		}
		return e.JSON(http.StatusOK, row.withCollectionMeta())
	}
}

// RecordsWriteRejectHTTP rejects create/update/delete on the fake collection URL.
func RecordsWriteRejectHTTP() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		return e.BadRequestError(writeRejectMessage, nil)
	}
}

// SuitesListHTTP serves GET /api/collections/conformance_suites/records.
func SuitesListHTTP() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		db, err := catalogDB()
		if err != nil {
			return e.InternalServerError("conformance catalog unavailable", err)
		}

		rows := []*suiteRow{}
		resolver := search.NewSimpleFieldResolver(suiteSearchFields...)
		base := db.Select(suiteSelectColumns...).From(SuitesCollectionName)

		result, err := search.NewProvider(resolver).
			Query(base).
			CountCol("id").
			ParseAndExec(e.Request.URL.Query().Encode(), &rows)
		if err != nil {
			return firstSearchAPIError(e, err)
		}

		for _, row := range rows {
			row.withCollectionMeta()
		}
		result.Items = rows
		return e.JSON(http.StatusOK, result)
	}
}

// SuiteViewHTTP serves GET /api/collections/conformance_suites/records/{id}.
func SuiteViewHTTP() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.NotFoundError("", nil)
		}

		db, err := catalogDB()
		if err != nil {
			return e.InternalServerError("conformance catalog unavailable", err)
		}

		row := &suiteRow{}
		err = db.Select(suiteSelectColumns...).
			From(SuitesCollectionName).
			AndWhere(dbx.HashExp{"id": id}).
			One(row)
		if err != nil {
			return e.NotFoundError("", err)
		}
		return e.JSON(http.StatusOK, row.withCollectionMeta())
	}
}

// SuitesWriteRejectHTTP rejects writes on the suites collection URL.
func SuitesWriteRejectHTTP() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		return e.BadRequestError(suitesWriteRejectMessage, nil)
	}
}

func firstSearchAPIError(e *core.RequestEvent, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, search.ErrFilterLengthLimit),
		errors.Is(err, search.ErrFilterExprLimit),
		errors.Is(err, search.ErrSortExprLimit),
		errors.Is(err, search.ErrSortFieldLengthLimit),
		errors.Is(err, search.ErrEmptyQuery):
		return e.BadRequestError("", err)
	default:
		return apis.NewBadRequestError(
			fmt.Sprintf("Something went wrong while processing your search request. %v", err),
			err,
		)
	}
}
