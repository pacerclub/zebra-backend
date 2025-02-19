package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pacerclub/zebra-backend/internal/auth"
	"github.com/pacerclub/zebra-backend/internal/db"
)

type SyncRequest struct {
	DeviceID        string     `json:"device_id"`
	LastSyncTime    time.Time  `json:"last_sync_time"`
	LocalSessions   []Session  `json:"local_sessions"`
	LocalProjects   []Project  `json:"local_projects"`
	DeletedSessions []uuid.UUID `json:"deleted_sessions"`
	DeletedProjects []uuid.UUID `json:"deleted_projects"`
}

type SyncResponse struct {
	LastSyncTime    time.Time  `json:"last_sync_time"`
	ServerSessions  []Session  `json:"server_sessions"`
	ServerProjects  []Project  `json:"server_projects"`
}

func SyncData(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Start a transaction
	tx, err := db.Pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	// Get or create device sync record
	var deviceLastSyncTime time.Time
	err = tx.QueryRow(r.Context(), `
		INSERT INTO device_sync (user_id, device_id, last_sync_time)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, device_id) 
		DO UPDATE SET last_sync_time = EXCLUDED.last_sync_time
		RETURNING last_sync_time
	`, userID, req.DeviceID, req.LastSyncTime).Scan(&deviceLastSyncTime)
	if err != nil {
		http.Error(w, "Failed to update device sync", http.StatusInternalServerError)
		return
	}

	// Process local projects
	for _, project := range req.LocalProjects {
		if project.ID == uuid.Nil {
			project.ID = uuid.New()
		}
		project.UserID = userID

		query := `
			INSERT INTO projects (id, user_id, name, description, color, device_id)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO UPDATE
			SET name = EXCLUDED.name,
				description = EXCLUDED.description,
				color = EXCLUDED.color,
				device_id = EXCLUDED.device_id,
				updated_at = CURRENT_TIMESTAMP
			WHERE projects.user_id = $2
		`

		_, err = tx.Exec(r.Context(), query,
			project.ID,
			project.UserID,
			project.Name,
			project.Description,
			project.Color,
			project.DeviceID,
		)
		if err != nil {
			http.Error(w, "Failed to sync project", http.StatusInternalServerError)
			return
		}
	}

	// Process local sessions
	for _, session := range req.LocalSessions {
		if session.ID == uuid.Nil {
			session.ID = uuid.New()
		}
		session.UserID = userID

		query := `
			INSERT INTO timer_sessions (id, user_id, project_id, start_time, end_time, description, device_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (id) DO UPDATE
			SET project_id = EXCLUDED.project_id,
				start_time = EXCLUDED.start_time,
				end_time = EXCLUDED.end_time,
				description = EXCLUDED.description,
				device_id = EXCLUDED.device_id,
				updated_at = CURRENT_TIMESTAMP
			WHERE timer_sessions.user_id = $2
		`

		_, err = tx.Exec(r.Context(), query,
			session.ID,
			session.UserID,
			session.ProjectID,
			session.StartTime,
			session.EndTime,
			session.Description,
			session.DeviceID,
		)
		if err != nil {
			http.Error(w, "Failed to sync session", http.StatusInternalServerError)
			return
		}
	}

	// Process deleted sessions
	if len(req.DeletedSessions) > 0 {
		query := `
			UPDATE timer_sessions
			SET is_deleted = true,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = ANY($1) AND user_id = $2
		`
		_, err = tx.Exec(r.Context(), query, req.DeletedSessions, userID)
		if err != nil {
			http.Error(w, "Failed to delete sessions", http.StatusInternalServerError)
			return
		}
	}

	// Process deleted projects
	if len(req.DeletedProjects) > 0 {
		query := `
			UPDATE projects
			SET is_deleted = true,
				updated_at = CURRENT_TIMESTAMP
			WHERE id = ANY($1) AND user_id = $2
		`
		_, err = tx.Exec(r.Context(), query, req.DeletedProjects, userID)
		if err != nil {
			http.Error(w, "Failed to delete projects", http.StatusInternalServerError)
			return
		}
	}

	// Get updated server data
	var serverSessions []Session
	sessionQuery := `
		SELECT id, user_id, project_id, start_time, end_time, description, device_id, is_deleted
		FROM timer_sessions
		WHERE user_id = $1 AND updated_at > $2
	`
	rows, err := tx.Query(r.Context(), sessionQuery, userID, deviceLastSyncTime)
	if err != nil {
		http.Error(w, "Failed to fetch server sessions", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

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
		serverSessions = append(serverSessions, session)
	}

	var serverProjects []Project
	projectQuery := `
		SELECT id, user_id, name, description, color, device_id, is_deleted
		FROM projects
		WHERE user_id = $1 AND updated_at > $2
	`
	rows, err = tx.Query(r.Context(), projectQuery, userID, deviceLastSyncTime)
	if err != nil {
		http.Error(w, "Failed to fetch server projects", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var project Project
		err := rows.Scan(
			&project.ID,
			&project.UserID,
			&project.Name,
			&project.Description,
			&project.Color,
			&project.DeviceID,
			&project.IsDeleted,
		)
		if err != nil {
			http.Error(w, "Failed to scan project", http.StatusInternalServerError)
			return
		}
		serverProjects = append(serverProjects, project)
	}

	// Update device's sync time
	now := time.Now()
	syncQuery := `
		UPDATE device_sync
		SET last_sync_time = $1,
			updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $2 AND device_id = $3
	`
	_, err = tx.Exec(r.Context(), syncQuery, now, userID, req.DeviceID)
	if err != nil {
		http.Error(w, "Failed to update sync status", http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// Send response
	response := SyncResponse{
		LastSyncTime:    now,
		ServerSessions:  serverSessions,
		ServerProjects:  serverProjects,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func SyncStatus(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get the last sync time for the user
	var lastSyncTime string
	err := db.Pool.QueryRow(r.Context(),
		"SELECT last_sync_time FROM user_sync_status WHERE user_id = $1",
		userID).Scan(&lastSyncTime)
	if err != nil {
		lastSyncTime = time.Time{}.UTC().Format(time.RFC3339)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"last_sync_time": lastSyncTime,
	})
}
