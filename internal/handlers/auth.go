package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/pacerclub/zebra-backend/internal/auth"
	"github.com/pacerclub/zebra-backend/internal/models"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	DeviceID string `json:"device_id"`
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	DeviceID string `json:"device_id"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func sendError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(errorResponse{Error: message})
}

func Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := models.CreateUser(r.Context(), req.Email, req.Password)
	if err != nil {
		sendError(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Email, req.DeviceID)
	if err != nil {
		sendError(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}

func Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := models.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		sendError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if !user.ValidatePassword(req.Password) {
		sendError(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Email, req.DeviceID)
	if err != nil {
		sendError(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}
