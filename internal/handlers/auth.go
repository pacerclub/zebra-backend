package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"

	"github.com/google/uuid"
	"github.com/pacerclub/zebra-backend/internal/auth"
	"github.com/pacerclub/zebra-backend/internal/db"
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

type loginResponse struct {
	Token       string    `json:"token"`
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	StorageMode string    `json:"storage_mode"`
	IsOnboarded bool      `json:"is_onboarded"`
}

type updatePreferencesRequest struct {
	StorageMode string `json:"storage_mode"`
	IsOnboarded bool   `json:"is_onboarded"`
}

func dumpRequest(r *http.Request) string {
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		return fmt.Sprintf("Error dumping request: %v", err)
	}
	return string(dump)
}

func sendError(w http.ResponseWriter, message string, code int) {
	log.Printf("Sending error response: %s (code: %d)", message, code)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, X-Requested-With, Origin")
}

func Register(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request body: %v", err)
		sendError(w, "Invalid request format. Please check your input.", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		log.Printf("Missing required fields in registration request")
		sendError(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	log.Printf("Attempting to create user with email: %s", req.Email)
	user, err := models.CreateUser(r.Context(), req.Email, req.Password)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		if err.Error() == "email already exists" {
			sendError(w, "Email is already registered", http.StatusConflict)
			return
		}
		sendError(w, "Failed to create user. Please try again later.", http.StatusInternalServerError)
		return
	}

	log.Printf("User created successfully, generating token")
	token, err := auth.GenerateToken(user.ID, user.Email, req.DeviceID)
	if err != nil {
		log.Printf("Error generating token: %v", err)
		sendError(w, "Account created but failed to generate login token. Please try logging in.", http.StatusInternalServerError)
		return
	}

	response := loginResponse{
		Token:       token,
		UserID:      user.ID,
		Email:       user.Email,
		StorageMode: user.StorageMode,
		IsOnboarded: user.IsOnboarded,
	}

	log.Printf("Registration successful for email: %s", req.Email)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode registration response: %v", err)
		sendError(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func Login(w http.ResponseWriter, r *http.Request) {
	log.Printf("=== Starting Login Request ===")
	log.Printf("Request details:\n%s", dumpRequest(r))

	setCORSHeaders(w)
	if r.Method == "OPTIONS" {
		log.Printf("Handling OPTIONS request")
		w.WriteHeader(http.StatusOK)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode login request: %v", err)
		sendError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	log.Printf("Login attempt - Email: %s, DeviceID: %s", req.Email, req.DeviceID)

	if req.Email == "" {
		log.Printf("Missing email in login request")
		sendError(w, "Email is required", http.StatusBadRequest)
		return
	}

	if req.Password == "" {
		log.Printf("Missing password in login request")
		sendError(w, "Password is required", http.StatusBadRequest)
		return
	}

	if req.DeviceID == "" {
		log.Printf("Missing device_id in login request")
		sendError(w, "Device ID is required", http.StatusBadRequest)
		return
	}

	user, err := models.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		log.Printf("Failed to find user with email %s: %v", req.Email, err)
		if err.Error() == "user not found" {
			sendError(w, "Invalid email or password", http.StatusUnauthorized)
		} else {
			sendError(w, "Failed to process login request", http.StatusInternalServerError)
		}
		return
	}

	log.Printf("Found user with email %s, ID: %s", user.Email, user.ID)
	log.Printf("User details - StorageMode: %s, IsOnboarded: %v", user.StorageMode, user.IsOnboarded)

	if !user.ValidatePassword(req.Password) {
		log.Printf("Invalid password for user %s", user.Email)
		sendError(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	log.Printf("Password validated successfully for user %s", user.Email)
	tokenString, err := auth.GenerateToken(user.ID, user.Email, req.DeviceID)
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		sendError(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	log.Printf("Generated token for user %s", user.Email)
	response := loginResponse{
		Token:       tokenString,
		UserID:      user.ID,
		Email:       user.Email,
		StorageMode: user.StorageMode,
		IsOnboarded: user.IsOnboarded,
	}

	log.Printf("Preparing response for user %s: %+v", user.Email, response)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode login response: %v", err)
		sendError(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	log.Printf("User %s successfully logged in", user.Email)
}

func UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	log.Printf("=== Starting Update Preferences Request ===")
	log.Printf("Request details:\n%s", dumpRequest(r))

	setCORSHeaders(w)
	if r.Method == "OPTIONS" {
		log.Printf("Handling OPTIONS request")
		w.WriteHeader(http.StatusOK)
		return
	}

	var req updatePreferencesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode update preferences request: %v", err)
		sendError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	log.Printf("Update preferences request - StorageMode: %s, IsOnboarded: %v", req.StorageMode, req.IsOnboarded)

	// Get user ID from context
	userID, ok := r.Context().Value("user_id").(uuid.UUID)
	if !ok {
		log.Printf("Failed to get user ID from context")
		sendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Update user preferences in database
	_, err := db.Pool.Exec(r.Context(),
		`UPDATE users 
		SET storage_mode = $1, 
		    is_onboarded = $2,
		    updated_at = CURRENT_TIMESTAMP 
		WHERE id = $3`,
		req.StorageMode, req.IsOnboarded, userID)
	if err != nil {
		log.Printf("Failed to update user preferences: %v", err)
		sendError(w, "Failed to update preferences", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully updated preferences for user %s", userID)
	w.WriteHeader(http.StatusOK)
}
