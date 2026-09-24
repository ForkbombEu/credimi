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
	writeRejectMessage       = "conformance_checks is a read-only catalog projection of config_templates"
	suitesWriteRejectMessage = "conformance_suites is a read-only catalog projection of config_templates"
)

// collectionHTTP owns PocketBase-compatible list/get/write-reject for one
// ephemeral fake-collection table (conformance check or conformance suite).
type collectionHTTP[R any] struct {
	table      string
	columns    []string
	fields     []string
	attachMeta func(*R)
	writeMsg   string
}

func checksCollectionHTTP() collectionHTTP[checkHTTPRecord] {
	return collectionHTTP[checkHTTPRecord]{
		table:      CollectionName,
		columns:    catalogSelectColumns,
		fields:     catalogSearchFields,
		attachMeta: func(r *checkHTTPRecord) { r.withCollectionMeta() },
		writeMsg:   writeRejectMessage,
	}
}

func suitesCollectionHTTP() collectionHTTP[suiteRow] {
	return collectionHTTP[suiteRow]{
		table:      SuitesCollectionName,
		columns:    suiteSelectColumns,
		fields:     suiteSearchFields,
		attachMeta: func(r *suiteRow) { r.withCollectionMeta() },
		writeMsg:   suitesWriteRejectMessage,
	}
}

func (c collectionHTTP[R]) list() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		db, err := catalogDB()
		if err != nil {
			return e.InternalServerError("conformance catalog unavailable", err)
		}

		rows := []*R{}
		resolver := search.NewSimpleFieldResolver(c.fields...)
		base := db.Select(c.columns...).From(c.table)

		result, err := search.NewProvider(resolver).
			Query(base).
			CountCol("id").
			ParseAndExec(e.Request.URL.Query().Encode(), &rows)
		if err != nil {
			return firstSearchAPIError(e, err)
		}

		for _, row := range rows {
			c.attachMeta(row)
		}
		result.Items = rows
		return e.JSON(http.StatusOK, result)
	}
}

func (c collectionHTTP[R]) view() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.NotFoundError("", nil)
		}

		db, err := catalogDB()
		if err != nil {
			return e.InternalServerError("conformance catalog unavailable", err)
		}

		row := new(R)
		err = db.Select(c.columns...).
			From(c.table).
			AndWhere(dbx.HashExp{"id": id}).
			One(row)
		if err != nil {
			return e.NotFoundError("", err)
		}
		c.attachMeta(row)
		return e.JSON(http.StatusOK, row)
	}
}

func (c collectionHTTP[R]) writeReject() func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		return e.BadRequestError(c.writeMsg, nil)
	}
}

// RecordsListHTTP serves GET /api/collections/conformance_checks/records
// using PocketBase tools/search against the process-private :memory: catalog.
func RecordsListHTTP() func(*core.RequestEvent) error {
	return checksCollectionHTTP().list()
}

// RecordViewHTTP serves GET /api/collections/conformance_checks/records/{id}.
func RecordViewHTTP() func(*core.RequestEvent) error {
	return checksCollectionHTTP().view()
}

// RecordsWriteRejectHTTP rejects create/update/delete on the fake collection URL.
func RecordsWriteRejectHTTP() func(*core.RequestEvent) error {
	return checksCollectionHTTP().writeReject()
}

// SuitesListHTTP serves GET /api/collections/conformance_suites/records.
func SuitesListHTTP() func(*core.RequestEvent) error {
	return suitesCollectionHTTP().list()
}

// SuiteViewHTTP serves GET /api/collections/conformance_suites/records/{id}.
func SuiteViewHTTP() func(*core.RequestEvent) error {
	return suitesCollectionHTTP().view()
}

// SuitesWriteRejectHTTP rejects writes on the suites collection URL.
func SuitesWriteRejectHTTP() func(*core.RequestEvent) error {
	return suitesCollectionHTTP().writeReject()
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
