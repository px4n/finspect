package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// sqliteStoreBase contains the common functionality for SQLite stores
type sqliteStoreBase struct {
	db      *sql.DB
	withFTS bool
}

// Close closes the database connection.
func (s *sqliteStoreBase) Close() error {
	return s.db.Close()
}

// Get retrieves metadata for a specific file.
func (s *sqliteStoreBase) Get(ctx context.Context, adaptor, path string) (*Metadata, error) {
	var meta Metadata
	var tagsJSON, propertiesJSON, mediaInfoJSON, documentInfoJSON sql.NullString

	var id int64
	query := `
		SELECT
			id, adaptor, path, updated_at, size, mode, mod_time, is_dir,
			content_type, md5, sha256, description, tags, properties,
			media_info, document_info
		FROM metadata
		WHERE adaptor = ? AND path = ?
	`

	err := s.db.QueryRowContext(ctx, query, adaptor, path).Scan(
		&id,
		&meta.Adaptor,
		&meta.Path,
		&meta.UpdatedAt,
		&meta.Size,
		&meta.Mode,
		&meta.ModTime,
		&meta.IsDir,
		&meta.ContentType,
		&meta.MD5,
		&meta.SHA256,
		&meta.Description,
		&tagsJSON,
		&propertiesJSON,
		&mediaInfoJSON,
		&documentInfoJSON,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query metadata: %w", err)
	}

	// Unmarshal JSON fields
	if tagsJSON.Valid && tagsJSON.String != "" {
		_ = json.Unmarshal([]byte(tagsJSON.String), &meta.Tags)
	}
	if propertiesJSON.Valid && propertiesJSON.String != "" {
		_ = json.Unmarshal([]byte(propertiesJSON.String), &meta.Properties)
	}
	if mediaInfoJSON.Valid && mediaInfoJSON.String != "" {
		_ = json.Unmarshal([]byte(mediaInfoJSON.String), &meta.MediaInfo)
	}
	if documentInfoJSON.Valid && documentInfoJSON.String != "" {
		_ = json.Unmarshal([]byte(documentInfoJSON.String), &meta.DocumentInfo)
	}

	return &meta, nil
}

// Put stores or updates metadata.
func (s *sqliteStoreBase) Put(ctx context.Context, metadata *Metadata) error {
	// Marshal JSON fields
	tagsJSON, _ := json.Marshal(metadata.Tags)
	propertiesJSON, _ := json.Marshal(metadata.Properties)
	mediaInfoJSON, _ := json.Marshal(metadata.MediaInfo)
	documentInfoJSON, _ := json.Marshal(metadata.DocumentInfo)

	query := `
		INSERT INTO metadata (
			adaptor, path, updated_at, size, mode, mod_time, is_dir,
			content_type, md5, sha256, description, tags, properties,
			media_info, document_info
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(adaptor, path) DO UPDATE SET
			updated_at = excluded.updated_at,
			size = excluded.size,
			mode = excluded.mode,
			mod_time = excluded.mod_time,
			is_dir = excluded.is_dir,
			content_type = excluded.content_type,
			md5 = excluded.md5,
			sha256 = excluded.sha256,
			description = excluded.description,
			tags = excluded.tags,
			properties = excluded.properties,
			media_info = excluded.media_info,
			document_info = excluded.document_info
	`

	_, err := s.db.ExecContext(ctx, query,
		metadata.Adaptor,
		metadata.Path,
		time.Now(),
		metadata.Size,
		metadata.Mode,
		metadata.ModTime,
		metadata.IsDir,
		metadata.ContentType,
		metadata.MD5,
		metadata.SHA256,
		metadata.Description,
		string(tagsJSON),
		string(propertiesJSON),
		string(mediaInfoJSON),
		string(documentInfoJSON),
	)

	return err
}

// Delete removes metadata for a file.
func (s *sqliteStoreBase) Delete(ctx context.Context, adaptor, path string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM metadata WHERE adaptor = ? AND path = ?", adaptor, path)
	return err
}

// BatchPut stores multiple metadata entries in a transaction.
func (s *sqliteStoreBase) BatchPut(ctx context.Context, metadataList []*Metadata) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO metadata (
			adaptor, path, updated_at, size, mode, mod_time, is_dir,
			content_type, md5, sha256, description, tags, properties,
			media_info, document_info
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(adaptor, path) DO UPDATE SET
			updated_at = excluded.updated_at,
			size = excluded.size,
			mode = excluded.mode,
			mod_time = excluded.mod_time,
			is_dir = excluded.is_dir,
			content_type = excluded.content_type,
			md5 = excluded.md5,
			sha256 = excluded.sha256,
			description = excluded.description,
			tags = excluded.tags,
			properties = excluded.properties,
			media_info = excluded.media_info,
			document_info = excluded.document_info
	`)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, metadata := range metadataList {
		// Marshal JSON fields
		tagsJSON, _ := json.Marshal(metadata.Tags)
		propertiesJSON, _ := json.Marshal(metadata.Properties)
		mediaInfoJSON, _ := json.Marshal(metadata.MediaInfo)
		documentInfoJSON, _ := json.Marshal(metadata.DocumentInfo)

		_, err = stmt.ExecContext(ctx,
			metadata.Adaptor,
			metadata.Path,
			time.Now(),
			metadata.Size,
			metadata.Mode,
			metadata.ModTime,
			metadata.IsDir,
			metadata.ContentType,
			metadata.MD5,
			metadata.SHA256,
			metadata.Description,
			string(tagsJSON),
			string(propertiesJSON),
			string(mediaInfoJSON),
			string(documentInfoJSON),
		)
		if err != nil {
			return fmt.Errorf("exec statement: %w", err)
		}
	}

	return tx.Commit()
}

// BatchDelete removes multiple metadata entries in a transaction.
func (s *sqliteStoreBase) BatchDelete(ctx context.Context, adaptorPathPairs [][2]string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, "DELETE FROM metadata WHERE adaptor = ? AND path = ?")
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, pair := range adaptorPathPairs {
		_, err = stmt.ExecContext(ctx, pair[0], pair[1])
		if err != nil {
			return fmt.Errorf("exec statement: %w", err)
		}
	}

	return tx.Commit()
}

// buildSearchQuery builds the common parts of a search query
func (s *sqliteStoreBase) buildSearchQuery(query Query) (string, []interface{}, string) {
	conditions := []string{}
	args := []interface{}{}

	if query.Adaptor != "" {
		conditions = append(conditions, "adaptor = ?")
		args = append(args, query.Adaptor)
	}

	if query.PathPrefix != "" {
		conditions = append(conditions, "path LIKE ?")
		args = append(args, query.PathPrefix+"%")
	}

	if query.ContentType != "" {
		conditions = append(conditions, "content_type = ?")
		args = append(args, query.ContentType)
	}

	if len(query.Tags) > 0 {
		tagConditions := []string{}
		for _, tag := range query.Tags {
			tagConditions = append(tagConditions, "tags LIKE ?")
			args = append(args, "%\""+tag+"\"%")
		}
		conditions = append(conditions, "("+strings.Join(tagConditions, " OR ")+")")
	}

	if query.MinSize > 0 {
		conditions = append(conditions, "size >= ?")
		args = append(args, query.MinSize)
	}

	if query.MaxSize > 0 {
		conditions = append(conditions, "size <= ?")
		args = append(args, query.MaxSize)
	}

	if query.ModifiedAfter != nil {
		conditions = append(conditions, "mod_time >= ?")
		args = append(args, query.ModifiedAfter)
	}

	if query.ModifiedBefore != nil {
		conditions = append(conditions, "mod_time <= ?")
		args = append(args, query.ModifiedBefore)
	}

	if query.IsDir != nil {
		conditions = append(conditions, "is_dir = ?")
		args = append(args, *query.IsDir)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	return whereClause, args, strings.Join(conditions, " AND ")
}

// executeSearch executes the search query and returns results
func (s *sqliteStoreBase) executeSearch(
	ctx context.Context, query string, args []interface{}, limit int,
) ([]*Metadata, error) {
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query metadata: %w", err)
	}
	defer rows.Close()

	var results []*Metadata
	for rows.Next() {
		var meta Metadata
		var tagsJSON, propertiesJSON, mediaInfoJSON, documentInfoJSON sql.NullString

		var id int64
		err := rows.Scan(
			&id,
			&meta.Adaptor,
			&meta.Path,
			&meta.UpdatedAt,
			&meta.Size,
			&meta.Mode,
			&meta.ModTime,
			&meta.IsDir,
			&meta.ContentType,
			&meta.MD5,
			&meta.SHA256,
			&meta.Description,
			&tagsJSON,
			&propertiesJSON,
			&mediaInfoJSON,
			&documentInfoJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		// Unmarshal JSON fields
		if tagsJSON.Valid && tagsJSON.String != "" {
			_ = json.Unmarshal([]byte(tagsJSON.String), &meta.Tags)
		}
		if propertiesJSON.Valid && propertiesJSON.String != "" {
			_ = json.Unmarshal([]byte(propertiesJSON.String), &meta.Properties)
		}
		if mediaInfoJSON.Valid && mediaInfoJSON.String != "" {
			_ = json.Unmarshal([]byte(mediaInfoJSON.String), &meta.MediaInfo)
		}
		if documentInfoJSON.Valid && documentInfoJSON.String != "" {
			_ = json.Unmarshal([]byte(documentInfoJSON.String), &meta.DocumentInfo)
		}

		results = append(results, &meta)
	}

	return results, rows.Err()
}
