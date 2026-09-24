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

// Process-default shared-memory catalog URI. Not attached to PocketBase data.db.
// Tests open private caches with a distinct URI via openCatalogCache.
const catalogMemoryURI = "file:credimi_conformance_catalog?mode=memory&cache=shared"

// catalogCache is one ephemeral :memory: query-cache handle (sole post-rebuild
// projection for that handle). Production uses the process default behind
// Rebuild / catalogDB; tests construct private handles with openCatalogCache.
type catalogCache struct {
	sql *sql.DB
	dbx *dbx.DB
}

var (
	defaultOnce  sync.Once
	defaultCache *catalogCache
	defaultErr   error
)

func openCatalogCache(uri string) (*catalogCache, error) {
	sqlDB, err := sql.Open("sqlite", uri)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	if err := sqlDB.PingContext(context.Background()); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	if err := ensureEphemeralSchema(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return &catalogCache{
		sql: sqlDB,
		dbx: dbx.NewFromDB(sqlDB, "sqlite"),
	}, nil
}

func (c *catalogCache) Close() error {
	if c == nil || c.sql == nil {
		return nil
	}
	return c.sql.Close()
}

func defaultCatalog() (*catalogCache, error) {
	defaultOnce.Do(func() {
		defaultCache, defaultErr = openCatalogCache(catalogMemoryURI)
	})
	if defaultErr != nil {
		return nil, defaultErr
	}
	return defaultCache, nil
}

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

// checkHTTPRecord is the list/get JSON shape: Check plus PocketBase collection meta.
type checkHTTPRecord struct {
	Check
	CollectionID   string `db:"-" json:"collectionId"`
	CollectionName string `db:"-" json:"collectionName"`
}

func (r *checkHTTPRecord) withCollectionMeta() *checkHTTPRecord {
	r.CollectionID = CollectionID
	r.CollectionName = ChecksCollectionName
	return r
}

// suiteHTTPRecord is the list/get JSON shape: SuiteRecord plus PocketBase collection meta.
type suiteHTTPRecord struct {
	SuiteRecord
	CollectionID   string `db:"-" json:"collectionId"`
	CollectionName string `db:"-" json:"collectionName"`
}

func (r *suiteHTTPRecord) withCollectionMeta() *suiteHTTPRecord {
	r.CollectionID = SuitesCollectionID
	r.CollectionName = SuitesCollectionName
	return r
}

// catalogDB returns the process-default cache's dbx handle (HTTP list/get facade).
func catalogDB() (*dbx.DB, error) {
	c, err := defaultCatalog()
	if err != nil {
		return nil, err
	}
	return c.dbx, nil
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
		createTableSQL(ChecksCollectionName, checkColumns),
		createTableSQL(SuitesCollectionName, suiteColumns),
	)
	_, err := db.ExecContext(context.Background(), sqlText)
	if err != nil {
		return fmt.Errorf("ensure ephemeral schema: %w", err)
	}
	return nil
}

func (c *catalogCache) countChecks() (int, error) {
	var n int
	if err := c.dbx.NewQuery("SELECT COUNT(*) FROM conformance_checks").Row(&n); err != nil {
		return 0, fmt.Errorf("count ephemeral checks: %w", err)
	}
	return n, nil
}

// countEphemeralChecks returns the check count on the process-default cache
// (rebuild HTTP response; boot tests).
func countEphemeralChecks() (int, error) {
	c, err := defaultCatalog()
	if err != nil {
		return 0, err
	}
	return c.countChecks()
}

// Rebuild walks templatesDir (or TemplatesDir() when empty) and fully replaces
// the process-default :memory: query cache used by the fake PocketBase
// collection URL. That cache is the sole post-rebuild projection for the
// default handle (ADR-0001).
//
// Refresh path for local template edits: call Rebuild, or POST
// /api/conformance-catalog/rebuild with X-Api-Key = CREDIMI_INTERNAL_ADMIN_KEY,
// or restart the process (Register hooks rebuild on bootstrap).
func Rebuild(templatesDir string) error {
	if templatesDir == "" {
		templatesDir = TemplatesDir()
	}

	loaded, err := LoadFromDir(templatesDir)
	if err != nil {
		return err
	}

	c, err := defaultCatalog()
	if err != nil {
		return err
	}
	if err := c.replaceEphemeralRows(loaded); err != nil {
		return fmt.Errorf("project ephemeral catalog: %w", err)
	}
	return nil
}

// replaceEphemeralRows fully replaces check and suite tables on this handle.
func (c *catalogCache) replaceEphemeralRows(loaded LoadedCatalog) error {
	tx, err := c.dbx.Begin()
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
	timestamps := dbx.Params{"created": now, "updated": now}
	for _, ch := range loaded.Checks {
		params, err := bindParams(checkColumns, ch, timestamps)
		if err != nil {
			return fmt.Errorf("bind ephemeral check %s: %w", ch.Path, err)
		}
		if _, err := tx.NewQuery(insertSQL(ChecksCollectionName, checkColumns)).Bind(params).Execute(); err != nil {
			return fmt.Errorf("insert ephemeral check %s: %w", ch.Path, err)
		}
	}

	for _, s := range loaded.Suites {
		params, err := bindParams(suiteColumns, s, timestamps)
		if err != nil {
			return fmt.Errorf("bind ephemeral suite %s: %w", s.PathPrefix, err)
		}
		if _, err := tx.NewQuery(insertSQL(SuitesCollectionName, suiteColumns)).Bind(params).Execute(); err != nil {
			return fmt.Errorf("insert ephemeral suite %s: %w", s.PathPrefix, err)
		}
	}

	return tx.Commit()
}
