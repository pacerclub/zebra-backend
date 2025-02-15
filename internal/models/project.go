package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pacerclub/zebra-backend/internal/db"
)

type Project struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Color       string    `json:"color"`
	DeviceID    string    `json:"device_id"`
	IsDeleted   bool      `json:"is_deleted"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func GetProjectsByUserID(ctx context.Context, userID uuid.UUID) ([]Project, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, user_id, name, description, color, device_id, created_at, updated_at, is_deleted
		FROM projects
		WHERE user_id = $1 AND is_deleted = false
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var project Project
		err := rows.Scan(
			&project.ID,
			&project.UserID,
			&project.Name,
			&project.Description,
			&project.Color,
			&project.DeviceID,
			&project.CreatedAt,
			&project.UpdatedAt,
			&project.IsDeleted,
		)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return projects, nil
}
