# Zebra Backend

A Go backend service for the Zebra time tracking application, supporting user accounts and multi-device synchronization.

## Features

- User authentication with JWT
- Timer session management
- Project management
- Multi-device synchronization
- PostgreSQL database for persistent storage

## Prerequisites

- Go 1.21 or later
- PostgreSQL 12 or later
- Docker (optional, for containerized development)

## Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/zigaowang/zebra-backend.git
   cd zebra-backend
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Create the database:
   ```bash
   createdb zebra
   ```

4. Set up the environment:
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

5. Initialize the database schema:
   ```bash
   psql -d zebra -f internal/db/schema.sql
   ```

6. Run the server:
   ```bash
   go run cmd/api/main.go
   ```

## API Endpoints

### Authentication
- `POST /api/register` - Register a new user
- `POST /api/login` - Login and get JWT token

### Timer Sessions
- `POST /api/sessions` - Create a new timer session
- `GET /api/sessions` - List user's timer sessions
- `PUT /api/sessions/{id}` - Update a timer session
- `DELETE /api/sessions/{id}` - Delete a timer session

### Projects
- `POST /api/projects` - Create a new project
- `GET /api/projects` - List user's projects
- `PUT /api/projects/{id}` - Update a project
- `DELETE /api/projects/{id}` - Delete a project

### Sync
- `POST /api/sync` - Sync data between devices
- `GET /api/sync/status` - Get sync status

## Development

### Database Migrations

The schema is currently managed through a single SQL file. For production, consider using a migration tool like `golang-migrate`.

### Testing

Run the tests:
```bash
go test ./...
```

### Docker

A Dockerfile and docker-compose configuration will be added soon for containerized deployment.

## License

MIT
