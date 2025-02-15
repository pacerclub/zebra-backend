package models

import (
	"context"
	"github.com/google/uuid"
	"github.com/pacerclub/zebra-backend/internal/db"
)

func GetUserPreferences(ctx context.Context, userID uuid.UUID) (string, bool, error) {
	var storageMode string
	var isOnboarded bool

	err := db.Pool.QueryRow(ctx,
		`SELECT COALESCE(storage_mode, 'cloud') as storage_mode,
		        COALESCE(is_onboarded, false) as is_onboarded
		 FROM users WHERE id = $1`,
		userID).Scan(&storageMode, &isOnboarded)
	if err != nil {
		return "", false, err
	}

	return storageMode, isOnboarded, nil
}

func UpdateUserPreferences(ctx context.Context, userID uuid.UUID, storageMode string, isOnboarded bool) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE users 
		 SET storage_mode = $2, is_onboarded = $3, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $1`,
		userID, storageMode, isOnboarded)
	return err
}
