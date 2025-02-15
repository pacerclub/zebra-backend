package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pacerclub/zebra-backend/internal/db"
)

type Session struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	ProjectID   uuid.UUID `json:"project_id"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Description string    `json:"description"`
	DeviceID    string    `json:"device_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	IsDeleted   bool      `json:"is_deleted"`
}

func GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, user_id, project_id, start_time, end_time, description, device_id, created_at, updated_at, is_deleted
		FROM timer_sessions
		WHERE user_id = $1 AND is_deleted = false
		ORDER BY start_time DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var session Session
		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.ProjectID,
			&session.StartTime,
			&session.EndTime,
			&session.Description,
			&session.DeviceID,
			&session.CreatedAt,
			&session.UpdatedAt,
			&session.IsDeleted,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return sessions, nil
}
