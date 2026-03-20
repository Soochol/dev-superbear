// Package repository provides database access for search presets.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// SearchPreset represents a row in the search_presets table.
type SearchPreset struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Name      string    `json:"name"`
	DSL       string    `json:"dsl"`
	NLQuery   *string   `json:"nlQuery"`
	IsPublic  bool      `json:"isPublic"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreatePresetParams are the parameters for creating a preset.
type CreatePresetParams struct {
	UserID   string
	Name     string
	DSL      string
	NLQuery  *string
	IsPublic bool
}

// PaginatedPresets represents a paginated list of presets.
type PaginatedPresets struct {
	Presets []SearchPreset `json:"presets"`
	Total   int64          `json:"total"`
}

// PresetRepository handles SearchPreset CRUD operations.
type PresetRepository struct {
	db *sql.DB
}

// NewPresetRepository creates a new PresetRepository.
func NewPresetRepository(db *sql.DB) *PresetRepository {
	return &PresetRepository{db: db}
}

// FindMany returns a paginated list of presets visible to the user.
func (r *PresetRepository) FindMany(ctx context.Context, userID string, limit, offset int32) (*PaginatedPresets, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, name, dsl, nl_query, is_public, created_at, updated_at
		 FROM search_presets
		 WHERE user_id = $1 OR is_public = true
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list presets: %w", err)
	}
	defer rows.Close()

	var presets []SearchPreset
	for rows.Next() {
		var p SearchPreset
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.DSL, &p.NLQuery, &p.IsPublic, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan preset: %w", err)
		}
		presets = append(presets, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate presets: %w", err)
	}

	var count int64
	err = r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM search_presets WHERE user_id = $1 OR is_public = true`,
		userID,
	).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("count presets: %w", err)
	}

	return &PaginatedPresets{Presets: presets, Total: count}, nil
}

// Create creates a new search preset.
func (r *PresetRepository) Create(ctx context.Context, params CreatePresetParams) (*SearchPreset, error) {
	var p SearchPreset
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO search_presets (user_id, name, dsl, nl_query, is_public)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, name, dsl, nl_query, is_public, created_at, updated_at`,
		params.UserID, params.Name, params.DSL, params.NLQuery, params.IsPublic,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.DSL, &p.NLQuery, &p.IsPublic, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create preset: %w", err)
	}
	return &p, nil
}

// Delete removes a preset owned by the specified user.
func (r *PresetRepository) Delete(ctx context.Context, id, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM search_presets WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("delete preset: %w", err)
	}
	return nil
}
