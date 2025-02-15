package models

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	ProjectID   *uuid.UUID `json:"project_id,omitempty"`
	StartTime   time.Time  `json:"start_time"`
	EndTime     time.Time  `json:"end_time"`
	Description string     `json:"description"`
	DeviceID    string     `json:"device_id"`
	IsDeleted   bool       `json:"is_deleted"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
