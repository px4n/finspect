//go:build !fts5
// +build !fts5

package metadata

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// SQLiteStore implements the Store interface using SQLite without FTS5.
type SQLiteStore struct {
	sqliteStoreBase
}

// NewSQLiteStore creates a new SQLite metadata store.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Enable foreign keys and WAL mode for better performance
	if _, err := db.Exec("PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set pragmas: %w", err)
	}

	return &SQLiteStore{
		sqliteStoreBase: sqliteStoreBase{
			db:      db,
			withFTS: false,
		},
	}, nil
}

// Init initializes the database schema without FTS5.
func (s *SQLiteStore) Init(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS metadata (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		adaptor TEXT NOT NULL,
		path TEXT NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		
		-- File attributes
		size INTEGER,
		mode INTEGER,
		mod_time TIMESTAMP,
		is_dir BOOLEAN,
		content_type TEXT,
		
		-- Content hashes
		md5 TEXT,
		sha256 TEXT,
		
		-- Extended attributes
		tags TEXT, -- JSON array
		description TEXT,
		properties TEXT, -- JSON object
		
		-- Media/Document info
		media_info TEXT, -- JSON object
		document_info TEXT, -- JSON object
		
		-- Unique constraint
		UNIQUE(adaptor, path)
	);
	
	-- Indexes for common queries
	CREATE INDEX IF NOT EXISTS idx_metadata_adaptor_path ON metadata(adaptor, path);
	CREATE INDEX IF NOT EXISTS idx_metadata_content_type ON metadata(content_type);
	CREATE INDEX IF NOT EXISTS idx_metadata_mod_time ON metadata(mod_time);
	CREATE INDEX IF NOT EXISTS idx_metadata_size ON metadata(size);
	CREATE INDEX IF NOT EXISTS idx_metadata_description ON metadata(description);
	`

	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// The Get, Put, Delete, BatchPut, and BatchDelete methods are inherited from sqliteStoreBase

// Search queries metadata without FTS5.
func (s *SQLiteStore) Search(ctx context.Context, query Query) ([]*Metadata, error) {
	whereClause, args, _ := s.buildSearchQuery(query)

	// Add text search for non-FTS
	if query.Text != "" {
		if whereClause == "" {
			whereClause = "WHERE "
		} else {
			whereClause += " AND "
		}
		whereClause += "(path LIKE ? OR description LIKE ? OR tags LIKE ?)"
		searchTerm := "%" + query.Text + "%"
		args = append(args, searchTerm, searchTerm, searchTerm)
	}

	// Build query
	q := `
		SELECT
			id, adaptor, path, updated_at, size, mode, mod_time, is_dir,
			content_type, md5, sha256, description, tags, properties,
			media_info, document_info
		FROM metadata
	` + whereClause

	// Add sorting
	switch query.SortBy {
	case "path", "size", "modified":
		if query.SortBy == "modified" {
			q += " ORDER BY mod_time"
		} else {
			q += " ORDER BY " + query.SortBy
		}
	case "name":
		q += " ORDER BY substr(path, instr(path, '/') + 1)"
	default:
		q += " ORDER BY path"
	}

	if query.SortOrder == "desc" {
		q += " DESC"
	}

	return s.executeSearch(ctx, q, args, query.Limit)
}

// BatchDelete removes multiple metadata entries.
func (s *SQLiteStore) BatchDelete(ctx context.Context, paths []PathKey) error {
	// Convert PathKey to array pairs for base implementation
	pairs := make([][2]string, len(paths))
	for i, pk := range paths {
		pairs[i] = [2]string{pk.Adaptor, pk.Path}
	}
	return s.sqliteStoreBase.BatchDelete(ctx, pairs)
}
