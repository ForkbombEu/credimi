// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/pocketbase/dbx"
	_ "modernc.org/sqlite"
)

// Process-private shared-memory catalog DB. Not attached to PocketBase data.db.
const catalogMemoryURI = "file:credimi_conformance_catalog?mode=memory&cache=shared"

var (
	memOnce sync.Once
	memSQL  *sql.DB
	memDBX  *dbx.DB
	errMem  error
)

// stringArray is a JSON string[] column for visible_in.
type stringArray []string

func (s *stringArray) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*s = stringArray{}
		return nil
	case []byte:
		if len(v) == 0 {
			*s = stringArray{}
			return nil
		}
		return json.Unmarshal(v, (*[]string)(s))
	case string:
		if v == "" {
			*s = stringArray{}
			return nil
		}
		return json.Unmarshal([]byte(v), (*[]string)(s))
	default:
		return fmt.Errorf("stringArray: unsupported Scan type %T", value)
	}
}

func (s stringArray) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]string(s))
}

// catalogRow is the list/get record shape returned on the fake PB collection URL.
type catalogRow struct {
	ID        string      `db:"id"         json:"id"`
	Path      string      `db:"path"       json:"path"`
	Title     string      `db:"title"      json:"title"`
	Standard  string      `db:"standard"   json:"standard"`
	Version   string      `db:"version"    json:"version"`
	Suite     string      `db:"suite"      json:"suite"`
	File      string      `db:"file"       json:"file"`
	VisibleIn stringArray `db:"visible_in" json:"visible_in"`
	Protocol  string      `db:"protocol"   json:"protocol"`
	SUT       string      `db:"sut"        json:"sut"`
	Role      string      `db:"role"       json:"role"`
	Provider  string      `db:"provider"   json:"provider"`
	Created   string      `db:"created"    json:"created"`
	Updated   string      `db:"updated"    json:"updated"`

	// PocketBase client compatibility fields (not stored).
	CollectionID   string `db:"-" json:"collectionId"`
	CollectionName string `db:"-" json:"collectionName"`
}

func (r *catalogRow) withCollectionMeta() *catalogRow {
	r.CollectionID = CollectionID
	r.CollectionName = CollectionName
	return r
}

func catalogDB() (*dbx.DB, error) {
	memOnce.Do(func() {
		memSQL, errMem = sql.Open("sqlite", catalogMemoryURI)
		if errMem != nil {
			return
		}
		memSQL.SetMaxOpenConns(1)
		memSQL.SetMaxIdleConns(1)
		if errMem = memSQL.PingContext(context.Background()); errMem != nil {
			return
		}
		memDBX = dbx.NewFromDB(memSQL, "sqlite")
		errMem = ensureEphemeralSchema(memSQL)
	})
	if errMem != nil {
		return nil, errMem
	}
	return memDBX, nil
}

func ensureEphemeralSchema(db *sql.DB) error {
	_, err := db.ExecContext(context.Background(), `
CREATE TABLE IF NOT EXISTS conformance_checks (
	id TEXT PRIMARY KEY NOT NULL,
	path TEXT NOT NULL,
	title TEXT NOT NULL,
	standard TEXT NOT NULL,
	version TEXT NOT NULL,
	suite TEXT NOT NULL,
	file TEXT NOT NULL,
	visible_in TEXT NOT NULL DEFAULT '[]',
	protocol TEXT NOT NULL DEFAULT '',
	sut TEXT NOT NULL DEFAULT '',
	role TEXT NOT NULL DEFAULT '',
	provider TEXT NOT NULL DEFAULT '',
	created TEXT NOT NULL DEFAULT '',
	updated TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_conformance_checks_path ON conformance_checks (path);
`)
	if err != nil {
		return fmt.Errorf("ensure ephemeral schema: %w", err)
	}
	return nil
}

// replaceEphemeralRows fully replaces the process-private catalog table.
func replaceEphemeralRows(checks []Check) error {
	db, err := catalogDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.NewQuery("DELETE FROM conformance_checks").Execute(); err != nil {
		return fmt.Errorf("clear ephemeral conformance_checks: %w", err)
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05.000Z")
	for _, ch := range checks {
		vis, err := json.Marshal(ch.VisibleIn)
		if err != nil {
			return fmt.Errorf("marshal visible_in for %s: %w", ch.Path, err)
		}
		if ch.VisibleIn == nil {
			vis = []byte("[]")
		}
		_, err = tx.NewQuery(`
INSERT INTO conformance_checks (
	id, path, title, standard, version, suite, file, visible_in,
	protocol, sut, role, provider, created, updated
) VALUES (
	{:id}, {:path}, {:title}, {:standard}, {:version}, {:suite}, {:file}, {:visible_in},
	{:protocol}, {:sut}, {:role}, {:provider}, {:created}, {:updated}
)`).Bind(dbx.Params{
			"id":         ch.ID,
			"path":       ch.Path,
			"title":      ch.Title,
			"standard":   ch.Standard,
			"version":    ch.Version,
			"suite":      ch.Suite,
			"file":       ch.File,
			"visible_in": string(vis),
			"protocol":   ch.Protocol,
			"sut":        ch.SUT,
			"role":       ch.Role,
			"provider":   ch.Provider,
			"created":    now,
			"updated":    now,
		}).Execute()
		if err != nil {
			return fmt.Errorf("insert ephemeral check %s: %w", ch.Path, err)
		}
	}
	return tx.Commit()
}
