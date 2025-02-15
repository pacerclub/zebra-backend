package models

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pacerclub/zebra-backend/internal/db"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never send password in JSON
	StorageMode  string    `json:"storage_mode"`
	IsOnboarded  bool      `json:"is_onboarded"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CreateUser creates a new user in the database
func CreateUser(ctx context.Context, email, password string) (*User, error) {
	log.Printf("Creating user with email: %s", email)
	
	// Check if user already exists
	existingUser, err := GetUserByEmail(ctx, email)
	if err == nil && existingUser != nil {
		log.Printf("User with email %s already exists", email)
		return nil, errors.New("email already exists")
	}
	
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		return nil, err
	}
	log.Printf("Password hashed successfully, length: %d", len(hashedPassword))

	user := &User{
		ID:          uuid.New(),
		StorageMode: "cloud",
		IsOnboarded: false,
	}
	
	log.Printf("Inserting user into database with ID: %s", user.ID)
	err = db.Pool.QueryRow(ctx,
		`INSERT INTO users (id, email, password_hash, storage_mode, is_onboarded) 
		VALUES ($1, $2, $3, $4, $5) 
		RETURNING id, email, password_hash, storage_mode, is_onboarded, created_at, updated_at`,
		user.ID, email, string(hashedPassword), user.StorageMode, user.IsOnboarded,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.StorageMode, &user.IsOnboarded, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		log.Printf("Database error creating user: %v", err)
		return nil, err
	}

	log.Printf("Successfully created user in database with ID: %s", user.ID)
	return user, nil
}

// GetUserByEmail retrieves a user by email
func GetUserByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	err := db.Pool.QueryRow(ctx,
		`SELECT id, email, password_hash, 
		COALESCE(storage_mode, 'cloud') as storage_mode,
		COALESCE(is_onboarded, false) as is_onboarded,
		created_at, updated_at 
		FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.StorageMode, &user.IsOnboarded, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return user, nil
}

// ValidatePassword checks if the provided password matches the stored hash
func (u *User) ValidatePassword(password string) bool {
	log.Printf("Validating password for user %s", u.Email)
	log.Printf("Password hash length: %d", len(u.PasswordHash))
	
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	if err != nil {
		log.Printf("Password validation failed: %v", err)
		return false
	}
	return true
}

// UpdateLastSync updates the last sync time for a user's device
func UpdateLastSync(ctx context.Context, userID uuid.UUID, deviceID, deviceType, deviceName string) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO device_sync (user_id, device_id, device_type, device_name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, device_id)
		DO UPDATE SET last_sync_time = CURRENT_TIMESTAMP,
		              device_type = EXCLUDED.device_type,
		              device_name = EXCLUDED.device_name`,
		userID, deviceID, deviceType, deviceName)
	return err
}
