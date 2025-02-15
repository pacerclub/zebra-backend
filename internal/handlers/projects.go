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

func CreateProject(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var project Project
	if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	project.UserID = userID
	project.ID = uuid.New()
	project.CreatedAt = time.Now()
	project.UpdatedAt = time.Now()

	query := `
		INSERT INTO projects (id, user_id, name, description, color, device_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, user_id, name, description, color, device_id, is_deleted, created_at, updated_at
	`

	err := db.Pool.QueryRow(r.Context(), query,
		project.ID,
		project.UserID,
		project.Name,
		project.Description,
		project.Color,
		project.DeviceID,
		project.CreatedAt,
		project.UpdatedAt,
	).Scan(
		&project.ID,
		&project.UserID,
		&project.Name,
		&project.Description,
		&project.Color,
		&project.DeviceID,
		&project.IsDeleted,
		&project.CreatedAt,
		&project.UpdatedAt,
	)

	if err != nil {
		http.Error(w, "Failed to create project", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

func ListProjects(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	query := `
		SELECT id, user_id, name, description, color, device_id, is_deleted, created_at, updated_at
		FROM projects
		WHERE user_id = $1 AND is_deleted = false
		ORDER BY created_at DESC
	`

	rows, err := db.Pool.Query(r.Context(), query, userID)
	if err != nil {
		http.Error(w, "Failed to fetch projects", http.StatusInternalServerError)
		return
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
			&project.IsDeleted,
			&project.CreatedAt,
			&project.UpdatedAt,
		)
		if err != nil {
			http.Error(w, "Failed to scan project", http.StatusInternalServerError)
			return
		}
		projects = append(projects, project)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

func UpdateProject(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	var project Project
	if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	project.UpdatedAt = time.Now()

	query := `
		UPDATE projects
		SET name = $1, description = $2, color = $3, updated_at = $4
		WHERE id = $5 AND user_id = $6
		RETURNING id, user_id, name, description, color, device_id, is_deleted, created_at, updated_at
	`

	err = db.Pool.QueryRow(r.Context(), query,
		project.Name,
		project.Description,
		project.Color,
		project.UpdatedAt,
		projectID,
		userID,
	).Scan(
		&project.ID,
		&project.UserID,
		&project.Name,
		&project.Description,
		&project.Color,
		&project.DeviceID,
		&project.IsDeleted,
		&project.CreatedAt,
		&project.UpdatedAt,
	)

	if err != nil {
		http.Error(w, "Failed to update project", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

func DeleteProject(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserIDFromContext(r.Context())
	if userID == uuid.Nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	query := `
		UPDATE projects
		SET is_deleted = true
		WHERE id = $1 AND user_id = $2
	`

	result, err := db.Pool.Exec(r.Context(), query, projectID, userID)
	if err != nil {
		http.Error(w, "Failed to delete project", http.StatusInternalServerError)
		return
	}

	if result.RowsAffected() == 0 {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
