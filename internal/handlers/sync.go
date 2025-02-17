package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pacerclub/zebra-backend/internal/auth"
	"github.com/pacerclub/zebra-backend/internal/db"
	"github.com/pacerclub/zebra-backend/internal/models"
)

type SyncRequest struct {
	DeviceID        string           `json:"device_id"`
	LastSyncTime    time.Time        `json:"last_sync_time"`
	LocalSessions   []models.Session `json:"local_sessions"`
	LocalProjects   []models.Project `json:"local_projects"`
	DeletedSessions []uuid.UUID      `json:"deleted_sessions"`
	DeletedProjects []uuid.UUID      `json:"deleted_projects"`
}

type SyncResponse struct {
	LastSyncTime    time.Time        `json:"last_sync_time"`
	ServerSessions  []models.Session `json:"server_sessions"`
	ServerProjects  []models.Project `json:"server_projects"`
}

func SyncData(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	userID := auth.GetUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		log.Printf("Unauthorized request to sync endpoint")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Handle GET request for fetching latest data
	if r.Method == "GET" {
		deviceID := r.URL.Query().Get("device_id")
		if deviceID == "" {
			http.Error(w, "device_id is required", http.StatusBadRequest)
			return
		}

		// Get sessions and projects
		sessions, err := models.GetSessionsByUserID(r.Context(), userID)
		if err != nil {
			log.Printf("Failed to get sessions: %v", err)
			http.Error(w, "Failed to get sessions", http.StatusInternalServerError)
			return
		}

		projects, err := models.GetProjectsByUserID(r.Context(), userID)
		if err != nil {
			log.Printf("Failed to get projects: %v", err)
			http.Error(w, "Failed to get projects", http.StatusInternalServerError)
			return
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sessions": sessions,
			"projects": projects,
		})
		return
	}

	// Handle POST request for syncing changes
	var req SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode sync request: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Processing sync request for user %s with device %s", userID, req.DeviceID)

	// Start a transaction
	tx, err := db.Pool.Begin(r.Context())
	if err != nil {
		log.Printf("Failed to start transaction: %v", err)
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

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
			log.Printf("Failed to sync session %s: %v", session.ID, err)
			http.Error(w, fmt.Sprintf("Failed to sync session: %v", err), http.StatusInternalServerError)
			return
		}
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
			log.Printf("Failed to sync project %s: %v", project.ID, err)
			http.Error(w, fmt.Sprintf("Failed to sync project: %v", err), http.StatusInternalServerError)
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
			log.Printf("Failed to mark sessions as deleted: %v", err)
			http.Error(w, fmt.Sprintf("Failed to mark sessions as deleted: %v", err), http.StatusInternalServerError)
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
			log.Printf("Failed to mark projects as deleted: %v", err)
			http.Error(w, fmt.Sprintf("Failed to mark projects as deleted: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Commit transaction
	if err = tx.Commit(r.Context()); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		http.Error(w, fmt.Sprintf("Failed to commit transaction: %v", err), http.StatusInternalServerError)
		return
	}

	// Get updated server data
	var serverSessions []models.Session
	sessionQuery := `
		SELECT id, user_id, project_id, start_time, end_time, description, device_id, is_deleted
		FROM timer_sessions
		WHERE user_id = $1
	`
	rows, err := tx.Query(r.Context(), sessionQuery, userID)
	if err != nil {
		log.Printf("Failed to fetch server sessions: %v", err)
		http.Error(w, fmt.Sprintf("Failed to fetch server sessions: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var session models.Session
		var projectID uuid.UUID
		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&projectID,
			&session.StartTime,
			&session.EndTime,
			&session.Description,
			&session.DeviceID,
			&session.IsDeleted,
		)
		if err != nil {
			log.Printf("Failed to scan session: %v", err)
			http.Error(w, fmt.Sprintf("Failed to scan session: %v", err), http.StatusInternalServerError)
			return
		}
		session.ProjectID = projectID
		serverSessions = append(serverSessions, session)
	}

	var serverProjects []models.Project
	projectQuery := `
		SELECT id, user_id, name, description, color, device_id, is_deleted
		FROM projects
		WHERE user_id = $1
	`
	rows, err = tx.Query(r.Context(), projectQuery, userID)
	if err != nil {
		log.Printf("Failed to fetch server projects: %v", err)
		http.Error(w, fmt.Sprintf("Failed to fetch server projects: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var project models.Project
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
			log.Printf("Failed to scan project: %v", err)
			http.Error(w, fmt.Sprintf("Failed to scan project: %v", err), http.StatusInternalServerError)
			return
		}
		serverProjects = append(serverProjects, project)
	}

	// Prepare and send response
	response := SyncResponse{
		LastSyncTime:    time.Now(),
		ServerSessions:  serverSessions,
		ServerProjects:  serverProjects,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
	
	log.Printf("Successfully processed sync request for user %s", userID)
}

func SyncStatus(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		log.Printf("Unauthorized request to sync status endpoint")
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
	if err := json.NewEncoder(w).Encode(map[string]string{
		"last_sync_time": lastSyncTime,
	}); err != nil {
		log.Printf("Failed to encode sync status response: %v", err)
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}
