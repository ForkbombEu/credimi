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

// memberArray is a JSON []SuiteMember column for suite members.
type memberArray []SuiteMember

func (m *memberArray) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*m = memberArray{}
		return nil
	case []byte:
		if len(v) == 0 {
			*m = memberArray{}
			return nil
		}
		return json.Unmarshal(v, (*[]SuiteMember)(m))
	case string:
		if v == "" {
			*m = memberArray{}
			return nil
		}
		return json.Unmarshal([]byte(v), (*[]SuiteMember)(m))
	default:
		return fmt.Errorf("memberArray: unsupported Scan type %T", value)
	}
}

func (m memberArray) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]SuiteMember(m))
}

// catalogRow is the list/get record shape returned on the fake PB collection URL.
type catalogRow struct {
	ID         string      `db:"id"          json:"id"`
	Path       string      `db:"path"        json:"path"`
	Title      string      `db:"title"       json:"title"`
	FSStandard string      `db:"fs_standard" json:"fs_standard"`
	FSVersion  string      `db:"fs_version"  json:"fs_version"`
	Suite      string      `db:"suite"       json:"suite"`
	File       string      `db:"file"        json:"file"`
	VisibleIn  stringArray `db:"visible_in"  json:"visible_in"`
	Protocol   string      `db:"protocol"    json:"protocol"`
	SUT        string      `db:"sut"         json:"sut"`
	Role       string      `db:"role"        json:"role"`
	Provider   string      `db:"provider"    json:"provider"`
	Standard   string      `db:"standard"    json:"standard"`
	Component  string      `db:"component"   json:"component"`
	Version    string      `db:"version"     json:"version"`
	Created    string      `db:"created"     json:"created"`
	Updated    string      `db:"updated"     json:"updated"`

	// PocketBase client compatibility fields (not stored).
	CollectionID   string `db:"-" json:"collectionId"`
	CollectionName string `db:"-" json:"collectionName"`
}

func (r *catalogRow) withCollectionMeta() *catalogRow {
	r.CollectionID = CollectionID
	r.CollectionName = CollectionName
	return r
}

// suiteRow is the suite-grain list/get shape for conformance_suites.
type suiteRow struct {
	ID               string      `db:"id"                json:"id"`
	Standard         string      `db:"standard"          json:"standard"`
	Component        string      `db:"component"         json:"component"`
	ComponentRank    int         `db:"component_rank"    json:"component_rank"`
	Version          string      `db:"version"           json:"version"`
	Suite            string      `db:"suite"             json:"suite"`
	Provider         string      `db:"provider"          json:"provider"`
	SuiteName        string      `db:"suite_name"        json:"suite_name"`
	SuiteHomepage    string      `db:"suite_homepage"    json:"suite_homepage"`
	SuiteRepository  string      `db:"suite_repository"  json:"suite_repository"`
	SuiteHelp        string      `db:"suite_help"        json:"suite_help"`
	SuiteDescription string      `db:"suite_description" json:"suite_description"`
	SuiteLogo        string      `db:"suite_logo"        json:"suite_logo"`
	CheckCount       int         `db:"check_count"       json:"check_count"`
	Members          memberArray `db:"members"           json:"members"`
	VisibleIn        stringArray `db:"visible_in"        json:"visible_in"`
	FSStandard       string      `db:"fs_standard"       json:"fs_standard"`
	FSVersion        string      `db:"fs_version"        json:"fs_version"`
	PathPrefix       string      `db:"path_prefix"       json:"path_prefix"`
	Created          string      `db:"created"           json:"created"`
	Updated          string      `db:"updated"           json:"updated"`

	CollectionID   string `db:"-" json:"collectionId"`
	CollectionName string `db:"-" json:"collectionName"`
}

func (r *suiteRow) withCollectionMeta() *suiteRow {
	r.CollectionID = SuitesCollectionID
	r.CollectionName = SuitesCollectionName
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
	// Ephemeral only: drop+create so column additions always match this binary.
	// Column lists come from schema.go (single SoT with SELECT/INSERT).
	sqlText := fmt.Sprintf(`
DROP TABLE IF EXISTS conformance_suites;
DROP TABLE IF EXISTS conformance_checks;
%s;
CREATE UNIQUE INDEX IF NOT EXISTS idx_conformance_checks_path ON conformance_checks (path);
%s;
CREATE UNIQUE INDEX IF NOT EXISTS idx_conformance_suites_path_prefix ON conformance_suites (path_prefix);
`,
		createTableSQL(CollectionName, checkColumns),
		createTableSQL(SuitesCollectionName, suiteColumns),
	)
	_, err := db.ExecContext(context.Background(), sqlText)
	if err != nil {
		return fmt.Errorf("ensure ephemeral schema: %w", err)
	}
	return nil
}

func marshalStringSlice(values []string) ([]byte, error) {
	if values == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(values)
}

// listEphemeralChecks reads every check row from the process-private cache.
func listEphemeralChecks() ([]Check, error) {
	db, err := catalogDB()
	if err != nil {
		return nil, err
	}
	var rows []*catalogRow
	if err := db.Select(catalogSelectColumns...).
		From(CollectionName).
		OrderBy("path ASC").
		All(&rows); err != nil {
		return nil, fmt.Errorf("list ephemeral checks: %w", err)
	}
	out := make([]Check, 0, len(rows))
	for _, r := range rows {
		out = append(out, checkFromRow(r))
	}
	return out, nil
}

// countEphemeralChecks returns the number of rows in the ephemeral checks table.
func countEphemeralChecks() (int, error) {
	db, err := catalogDB()
	if err != nil {
		return 0, err
	}
	var n int
	if err := db.NewQuery("SELECT COUNT(*) FROM conformance_checks").Row(&n); err != nil {
		return 0, fmt.Errorf("count ephemeral checks: %w", err)
	}
	return n, nil
}

func checkFromRow(r *catalogRow) Check {
	if r == nil {
		return Check{}
	}
	return Check{
		ID:         r.ID,
		Path:       r.Path,
		Title:      r.Title,
		FSStandard: r.FSStandard,
		FSVersion:  r.FSVersion,
		Suite:      r.Suite,
		File:       r.File,
		VisibleIn:  append([]string(nil), r.VisibleIn...),
		Protocol:   r.Protocol,
		SUT:        r.SUT,
		Role:       r.Role,
		Provider:   r.Provider,
		Standard:   r.Standard,
		Component:  r.Component,
		Version:    r.Version,
	}
}

// replaceEphemeralRows fully replaces process-private check and suite tables.
func replaceEphemeralRows(loaded LoadedCatalog) error {
	db, err := catalogDB()
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.NewQuery("DELETE FROM conformance_suites").Execute(); err != nil {
		return fmt.Errorf("clear ephemeral conformance_suites: %w", err)
	}
	if _, err := tx.NewQuery("DELETE FROM conformance_checks").Execute(); err != nil {
		return fmt.Errorf("clear ephemeral conformance_checks: %w", err)
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05.000Z")
	for _, ch := range loaded.Checks {
		vis, err := marshalStringSlice(ch.VisibleIn)
		if err != nil {
			return fmt.Errorf("marshal visible_in for %s: %w", ch.Path, err)
		}
		_, err = tx.NewQuery(insertSQL(CollectionName, checkColumns)).Bind(dbx.Params{
			"id":          ch.ID,
			"path":        ch.Path,
			"title":       ch.Title,
			"fs_standard": ch.FSStandard,
			"fs_version":  ch.FSVersion,
			"suite":       ch.Suite,
			"file":        ch.File,
			"visible_in":  string(vis),
			"protocol":    ch.Protocol,
			"sut":         ch.SUT,
			"role":        ch.Role,
			"provider":    ch.Provider,
			"standard":    ch.Standard,
			"component":   ch.Component,
			"version":     ch.Version,
			"created":     now,
			"updated":     now,
		}).Execute()
		if err != nil {
			return fmt.Errorf("insert ephemeral check %s: %w", ch.Path, err)
		}
	}

	for _, s := range ProjectSuites(loaded.Checks, loaded.SuiteDisplay) {
		vis, err := marshalStringSlice(s.VisibleIn)
		if err != nil {
			return fmt.Errorf("marshal suite visible_in for %s: %w", s.PathPrefix, err)
		}
		members, err := json.Marshal(s.Members)
		if err != nil {
			return fmt.Errorf("marshal members for %s: %w", s.PathPrefix, err)
		}
		if s.Members == nil {
			members = []byte("[]")
		}
		_, err = tx.NewQuery(insertSQL(SuitesCollectionName, suiteColumns)).Bind(dbx.Params{
			"id":                s.ID,
			"standard":          s.Standard,
			"component":         s.Component,
			"component_rank":    s.ComponentRank,
			"version":           s.Version,
			"suite":             s.Suite,
			"provider":          s.Provider,
			"suite_name":        s.SuiteName,
			"suite_homepage":    s.SuiteHomepage,
			"suite_repository":  s.SuiteRepository,
			"suite_help":        s.SuiteHelp,
			"suite_description": s.SuiteDescription,
			"suite_logo":        s.SuiteLogo,
			"check_count":       s.CheckCount,
			"members":           string(members),
			"visible_in":        string(vis),
			"fs_standard":       s.FSStandard,
			"fs_version":        s.FSVersion,
			"path_prefix":       s.PathPrefix,
			"created":           now,
			"updated":           now,
		}).Execute()
		if err != nil {
			return fmt.Errorf("insert ephemeral suite %s: %w", s.PathPrefix, err)
		}
	}

	return tx.Commit()
}
