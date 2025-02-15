-- Drop existing triggers
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
DROP TRIGGER IF EXISTS update_timer_sessions_updated_at ON timer_sessions;
DROP TRIGGER IF EXISTS update_sync_status_updated_at ON user_sync_status;
DROP TRIGGER IF EXISTS update_device_sync_updated_at ON device_sync;

-- Drop existing tables
DROP TABLE IF EXISTS user_sync_status CASCADE;
DROP TABLE IF EXISTS timer_sessions CASCADE;
DROP TABLE IF EXISTS projects CASCADE;
DROP TABLE IF EXISTS device_sync CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Drop existing functions
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create device sync table
CREATE TABLE device_sync (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id VARCHAR(255) NOT NULL,
    last_sync_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, device_id)
);

-- Create projects table first (since timer_sessions depends on it)
CREATE TABLE projects (
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

-- Create sessions table for storing timer sessions
CREATE TABLE timer_sessions (
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

-- Create user sync status table for managing user sync status
CREATE TABLE user_sync_status (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    last_sync_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    device_id VARCHAR(255),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Add indexes for better query performance
CREATE INDEX idx_timer_sessions_user_id ON timer_sessions(user_id);
CREATE INDEX idx_timer_sessions_project_id ON timer_sessions(project_id);
CREATE INDEX idx_projects_user_id ON projects(user_id);
CREATE INDEX idx_device_sync_user_id ON device_sync(user_id);
CREATE INDEX idx_device_sync_device_id ON device_sync(device_id);

-- Update timestamp triggers
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_projects_updated_at
    BEFORE UPDATE ON projects
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_timer_sessions_updated_at
    BEFORE UPDATE ON timer_sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_sync_status_updated_at
    BEFORE UPDATE ON user_sync_status
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_device_sync_updated_at
    BEFORE UPDATE ON device_sync
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
