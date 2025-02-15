package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

// InitDB initializes the database connection pool
func InitDB() error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/zebra?sslmode=disable"
	}

	log.Printf("DATABASE_URL: %s", strings.Replace(databaseURL, "postgres://", "postgres://****:****@", 1))
	log.Println("Starting database initialization...")

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("error parsing database URL: %v", err)
	}

	// Set reasonable pool limits
	config.MaxConns = 10
	config.MinConns = 2

	Pool, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return fmt.Errorf("unable to create connection pool: %v", err)
	}

	// Test the connection
	if err := Pool.Ping(context.Background()); err != nil {
		return fmt.Errorf("unable to ping database: %v", err)
	}

	// Run migrations
	if err := runMigrations(); err != nil {
		// If migrations fail, try to create tables directly
		log.Printf("Warning: Failed to run migrations: %v", err)
		log.Println("Attempting to create tables directly...")
		if err := createTables(); err != nil {
			return fmt.Errorf("failed to initialize database schema: %v", err)
		}
	}

	return nil
}

// createTables creates the database tables directly if migrations fail
func createTables() error {
	ctx := context.Background()
	
	// Enable UUID extension
	_, err := Pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`)
	if err != nil {
		return fmt.Errorf("failed to create uuid extension: %v", err)
	}

	// Create users table
	_, err = Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			storage_mode VARCHAR(50) DEFAULT 'cloud',
			is_onboarded BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create users table: %v", err)
	}

	// Create projects table
	_, err = Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS projects (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			color VARCHAR(50) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			device_id VARCHAR(255),
			is_deleted BOOLEAN DEFAULT FALSE
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create projects table: %v", err)
	}

	// Create timer_sessions table
	_, err = Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS timer_sessions (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
			start_time TIMESTAMP WITH TIME ZONE NOT NULL,
			end_time TIMESTAMP WITH TIME ZONE NOT NULL,
			description TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			device_id VARCHAR(255),
			is_deleted BOOLEAN DEFAULT FALSE
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create timer_sessions table: %v", err)
	}

	// Create user_sync_status table
	_, err = Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS user_sync_status (
			user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			last_sync_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			device_id VARCHAR(255),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create user_sync_status table: %v", err)
	}

	// Create device_sync table
	_, err = Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS device_sync (
			user_id UUID NOT NULL REFERENCES users(id),
			device_id TEXT NOT NULL,
			last_sync_time TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, device_id)
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create device_sync table: %v", err)
	}

	// Create indexes
	_, err = Pool.Exec(ctx, `
		CREATE INDEX IF NOT EXISTS idx_timer_sessions_user_id ON timer_sessions(user_id);
		CREATE INDEX IF NOT EXISTS idx_timer_sessions_project_id ON timer_sessions(project_id);
		CREATE INDEX IF NOT EXISTS idx_projects_user_id ON projects(user_id);
		
		-- Add storage_mode and is_onboarded columns to users table if they don't exist
		DO $$ 
		BEGIN 
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'storage_mode') THEN
				ALTER TABLE users ADD COLUMN storage_mode VARCHAR(50) DEFAULT 'cloud';
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'is_onboarded') THEN
				ALTER TABLE users ADD COLUMN is_onboarded BOOLEAN DEFAULT FALSE;
			END IF;
		END $$;
	`)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %v", err)
	}

	// Create update trigger function
	_, err = Pool.Exec(ctx, `
		CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = CURRENT_TIMESTAMP;
			RETURN NEW;
		END;
		$$ language 'plpgsql';
	`)
	if err != nil {
		return fmt.Errorf("failed to create trigger function: %v", err)
	}

	// Create triggers
	triggers := []string{
		`CREATE TRIGGER update_users_updated_at
			BEFORE UPDATE ON users
			FOR EACH ROW
			EXECUTE FUNCTION update_updated_at_column();`,
		`CREATE TRIGGER update_projects_updated_at
			BEFORE UPDATE ON projects
			FOR EACH ROW
			EXECUTE FUNCTION update_updated_at_column();`,
		`CREATE TRIGGER update_timer_sessions_updated_at
			BEFORE UPDATE ON timer_sessions
			FOR EACH ROW
			EXECUTE FUNCTION update_updated_at_column();`,
		`CREATE TRIGGER update_sync_status_updated_at
			BEFORE UPDATE ON user_sync_status
			FOR EACH ROW
			EXECUTE FUNCTION update_updated_at_column();`,
		`CREATE TRIGGER update_device_sync_updated_at
			BEFORE UPDATE ON device_sync
			FOR EACH ROW
			EXECUTE FUNCTION update_updated_at_column();`,
	}

	for _, trigger := range triggers {
		_, err = Pool.Exec(ctx, trigger)
		if err != nil {
			// Ignore errors about triggers already existing
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("failed to create trigger: %v", err)
			}
		}
	}

	log.Println("Successfully created database schema")
	return nil
}

// runMigrations executes all SQL migration files in order
func runMigrations() error {
	log.Println("Running database migrations...")
	
	migrationsDir := "internal/db/migrations"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return fmt.Errorf("failed to read migration directory: %v", err)
	}

	// Read all SQL files from migrations directory
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("error reading migration files: %v", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no migration files found in %s", migrationsDir)
	}

	// Sort files by name to ensure correct order
	for _, file := range files {
		log.Printf("Applying migration: %s", filepath.Base(file))
		
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("error reading migration file %s: %v", file, err)
		}

		// Execute the migration
		_, err = Pool.Exec(context.Background(), string(content))
		if err != nil {
			return fmt.Errorf("error executing migration %s: %v", file, err)
		}
		
		log.Printf("Successfully applied migration: %s", filepath.Base(file))
	}

	return nil
}

// GetDB returns the database pool
func GetDB() *pgxpool.Pool {
	return Pool
}

// CloseDB closes the database connection pool
func CloseDB() {
	if Pool != nil {
		Pool.Close()
	}
}
