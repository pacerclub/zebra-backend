package models

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pacerclub/zebra-backend/internal/db"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // Never send password in JSON
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateUser creates a new user in the database
func CreateUser(ctx context.Context, email, password string) (*User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &User{ID: uuid.New()}
	err = db.GetDB().QueryRow(ctx,
		`INSERT INTO users (id, email, password_hash) 
		VALUES ($1, $2, $3) 
		RETURNING id, email, created_at, updated_at`,
		user.ID, email, string(hashedPassword),
	).Scan(&user.ID, &user.Email, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email
func GetUserByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	err := db.GetDB().QueryRow(ctx,
		`SELECT id, email, password_hash, created_at, updated_at 
		FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

// ValidatePassword checks if the provided password matches the stored hash
func (u *User) ValidatePassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// UpdateLastSync updates the last sync time for a user's device
func UpdateLastSync(ctx context.Context, userID uuid.UUID, deviceID, deviceType, deviceName string) error {
	_, err := db.GetDB().Exec(ctx,
		`INSERT INTO device_sync (user_id, device_id, device_type, device_name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, device_id)
		DO UPDATE SET last_sync_time = CURRENT_TIMESTAMP,
		              device_type = EXCLUDED.device_type,
		              device_name = EXCLUDED.device_name`,
		userID, deviceID, deviceType, deviceName)
	return err
}
