package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pacerclub/zebra-backend/internal/auth"
	"github.com/pacerclub/zebra-backend/internal/db"
)

type Session struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	ProjectID   *uuid.UUID `json:"project_id,omitempty"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Description string    `json:"description"`
	DeviceID    string    `json:"device_id"`
	IsDeleted   bool      `json:"is_deleted"`
}

func CreateSession(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var session Session
	if err := json.NewDecoder(r.Body).Decode(&session); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	session.UserID = userID

	query := `
		INSERT INTO timer_sessions (id, user_id, project_id, start_time, end_time, description, device_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, project_id, start_time, end_time, description, device_id, is_deleted
	`

	err := db.Pool.QueryRow(r.Context(), query,
		session.ID,
		session.UserID,
		session.ProjectID,
		session.StartTime,
		session.EndTime,
		session.Description,
		session.DeviceID,
	).Scan(
		&session.ID,
		&session.UserID,
		&session.ProjectID,
		&session.StartTime,
		&session.EndTime,
		&session.Description,
		&session.DeviceID,
		&session.IsDeleted,
	)

	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func ListSessions(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	query := `
		SELECT id, user_id, project_id, start_time, end_time, description, device_id, is_deleted
		FROM timer_sessions
		WHERE user_id = $1 AND is_deleted = false
		ORDER BY start_time DESC
	`

	rows, err := db.Pool.Query(r.Context(), query, userID)
	if err != nil {
		http.Error(w, "Failed to fetch sessions", http.StatusInternalServerError)
		return
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
			&session.IsDeleted,
		)
		if err != nil {
			http.Error(w, "Failed to scan session", http.StatusInternalServerError)
			return
		}
		sessions = append(sessions, session)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func UpdateSession(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}

	var session Session
	if err := json.NewDecoder(r.Body).Decode(&session); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `
		UPDATE timer_sessions
		SET project_id = $1, start_time = $2, end_time = $3, description = $4
		WHERE id = $5 AND user_id = $6
		RETURNING id, user_id, project_id, start_time, end_time, description, device_id, is_deleted
	`

	err = db.Pool.QueryRow(r.Context(), query,
		session.ProjectID,
		session.StartTime,
		session.EndTime,
		session.Description,
		sessionID,
		userID,
	).Scan(
		&session.ID,
		&session.UserID,
		&session.ProjectID,
		&session.StartTime,
		&session.EndTime,
		&session.Description,
		&session.DeviceID,
		&session.IsDeleted,
	)

	if err != nil {
		http.Error(w, "Failed to update session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func DeleteSession(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}

	query := `
		UPDATE timer_sessions
		SET is_deleted = true
		WHERE id = $1 AND user_id = $2
	`

	result, err := db.Pool.Exec(r.Context(), query, sessionID, userID)
	if err != nil {
		http.Error(w, "Failed to delete session", http.StatusInternalServerError)
		return
	}

	if result.RowsAffected() == 0 {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
