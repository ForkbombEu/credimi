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

// stringArray is a JSON string[] column for visible_in / check_* arrays.
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
	ID               string      `db:"id"                json:"id"`
	Path             string      `db:"path"              json:"path"`
	Title            string      `db:"title"             json:"title"`
	Standard         string      `db:"standard"          json:"standard"`
	Version          string      `db:"version"           json:"version"`
	Suite            string      `db:"suite"             json:"suite"`
	File             string      `db:"file"              json:"file"`
	VisibleIn        stringArray `db:"visible_in"        json:"visible_in"`
	Protocol         string      `db:"protocol"          json:"protocol"`
	SUT              string      `db:"sut"               json:"sut"`
	Role             string      `db:"role"              json:"role"`
	Provider         string      `db:"provider"          json:"provider"`
	NormStandard     string      `db:"norm_standard"     json:"norm_standard"`
	Component        string      `db:"component"         json:"component"`
	NormVersion      string      `db:"norm_version"      json:"norm_version"`
	SuiteName        string      `db:"suite_name"        json:"suite_name"`
	SuiteHomepage    string      `db:"suite_homepage"    json:"suite_homepage"`
	SuiteRepository  string      `db:"suite_repository"  json:"suite_repository"`
	SuiteHelp        string      `db:"suite_help"        json:"suite_help"`
	SuiteDescription string      `db:"suite_description" json:"suite_description"`
	SuiteLogo        string      `db:"suite_logo"        json:"suite_logo"`
	Created          string      `db:"created"           json:"created"`
	Updated          string      `db:"updated"           json:"updated"`

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
	CheckPaths       stringArray `db:"check_paths"       json:"check_paths"`
	CheckTitles      stringArray `db:"check_titles"      json:"check_titles"`
	CheckFiles       stringArray `db:"check_files"       json:"check_files"`
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
	_, err := db.ExecContext(context.Background(), `
DROP TABLE IF EXISTS conformance_suites;
DROP TABLE IF EXISTS conformance_checks;
CREATE TABLE conformance_checks (
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
	norm_standard TEXT NOT NULL DEFAULT '',
	component TEXT NOT NULL DEFAULT '',
	norm_version TEXT NOT NULL DEFAULT '',
	suite_name TEXT NOT NULL DEFAULT '',
	suite_homepage TEXT NOT NULL DEFAULT '',
	suite_repository TEXT NOT NULL DEFAULT '',
	suite_help TEXT NOT NULL DEFAULT '',
	suite_description TEXT NOT NULL DEFAULT '',
	suite_logo TEXT NOT NULL DEFAULT '',
	created TEXT NOT NULL DEFAULT '',
	updated TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_conformance_checks_path ON conformance_checks (path);
CREATE TABLE conformance_suites (
	id TEXT PRIMARY KEY NOT NULL,
	standard TEXT NOT NULL,
	component TEXT NOT NULL DEFAULT '',
	component_rank INTEGER NOT NULL DEFAULT 9,
	version TEXT NOT NULL DEFAULT '',
	suite TEXT NOT NULL,
	provider TEXT NOT NULL DEFAULT '',
	suite_name TEXT NOT NULL DEFAULT '',
	suite_homepage TEXT NOT NULL DEFAULT '',
	suite_repository TEXT NOT NULL DEFAULT '',
	suite_help TEXT NOT NULL DEFAULT '',
	suite_description TEXT NOT NULL DEFAULT '',
	suite_logo TEXT NOT NULL DEFAULT '',
	check_count INTEGER NOT NULL DEFAULT 0,
	check_paths TEXT NOT NULL DEFAULT '[]',
	check_titles TEXT NOT NULL DEFAULT '[]',
	check_files TEXT NOT NULL DEFAULT '[]',
	visible_in TEXT NOT NULL DEFAULT '[]',
	fs_standard TEXT NOT NULL DEFAULT '',
	fs_version TEXT NOT NULL DEFAULT '',
	path_prefix TEXT NOT NULL,
	created TEXT NOT NULL DEFAULT '',
	updated TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_conformance_suites_path_prefix ON conformance_suites (path_prefix);
`)
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

// replaceEphemeralRows fully replaces process-private check and suite tables.
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

	if _, err := tx.NewQuery("DELETE FROM conformance_suites").Execute(); err != nil {
		return fmt.Errorf("clear ephemeral conformance_suites: %w", err)
	}
	if _, err := tx.NewQuery("DELETE FROM conformance_checks").Execute(); err != nil {
		return fmt.Errorf("clear ephemeral conformance_checks: %w", err)
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05.000Z")
	for _, ch := range checks {
		vis, err := marshalStringSlice(ch.VisibleIn)
		if err != nil {
			return fmt.Errorf("marshal visible_in for %s: %w", ch.Path, err)
		}
		_, err = tx.NewQuery(`
INSERT INTO conformance_checks (
	id, path, title, standard, version, suite, file, visible_in,
	protocol, sut, role, provider,
	norm_standard, component, norm_version,
	suite_name, suite_homepage, suite_repository, suite_help, suite_description, suite_logo,
	created, updated
) VALUES (
	{:id}, {:path}, {:title}, {:standard}, {:version}, {:suite}, {:file}, {:visible_in},
	{:protocol}, {:sut}, {:role}, {:provider},
	{:norm_standard}, {:component}, {:norm_version},
	{:suite_name}, {:suite_homepage}, {:suite_repository}, {:suite_help}, {:suite_description}, {:suite_logo},
	{:created}, {:updated}
)`).Bind(dbx.Params{
			"id":                ch.ID,
			"path":              ch.Path,
			"title":             ch.Title,
			"standard":          ch.Standard,
			"version":           ch.Version,
			"suite":             ch.Suite,
			"file":              ch.File,
			"visible_in":        string(vis),
			"protocol":          ch.Protocol,
			"sut":               ch.SUT,
			"role":              ch.Role,
			"provider":          ch.Provider,
			"norm_standard":     ch.NormStandard,
			"component":         ch.Component,
			"norm_version":      ch.NormVersion,
			"suite_name":        ch.SuiteName,
			"suite_homepage":    ch.SuiteHomepage,
			"suite_repository":  ch.SuiteRepository,
			"suite_help":        ch.SuiteHelp,
			"suite_description": ch.SuiteDescription,
			"suite_logo":        ch.SuiteLogo,
			"created":           now,
			"updated":           now,
		}).Execute()
		if err != nil {
			return fmt.Errorf("insert ephemeral check %s: %w", ch.Path, err)
		}
	}

	for _, s := range ProjectSuites(checks) {
		vis, err := marshalStringSlice(s.VisibleIn)
		if err != nil {
			return fmt.Errorf("marshal suite visible_in for %s: %w", s.PathPrefix, err)
		}
		paths, err := marshalStringSlice(s.CheckPaths)
		if err != nil {
			return fmt.Errorf("marshal check_paths for %s: %w", s.PathPrefix, err)
		}
		titles, err := marshalStringSlice(s.CheckTitles)
		if err != nil {
			return fmt.Errorf("marshal check_titles for %s: %w", s.PathPrefix, err)
		}
		files, err := marshalStringSlice(s.CheckFiles)
		if err != nil {
			return fmt.Errorf("marshal check_files for %s: %w", s.PathPrefix, err)
		}
		_, err = tx.NewQuery(`
INSERT INTO conformance_suites (
	id, standard, component, component_rank, version, suite, provider,
	suite_name, suite_homepage, suite_repository, suite_help, suite_description, suite_logo,
	check_count, check_paths, check_titles, check_files, visible_in,
	fs_standard, fs_version, path_prefix, created, updated
) VALUES (
	{:id}, {:standard}, {:component}, {:component_rank}, {:version}, {:suite}, {:provider},
	{:suite_name}, {:suite_homepage}, {:suite_repository}, {:suite_help}, {:suite_description}, {:suite_logo},
	{:check_count}, {:check_paths}, {:check_titles}, {:check_files}, {:visible_in},
	{:fs_standard}, {:fs_version}, {:path_prefix}, {:created}, {:updated}
)`).Bind(dbx.Params{
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
			"check_paths":       string(paths),
			"check_titles":      string(titles),
			"check_files":       string(files),
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
