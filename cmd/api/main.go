package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	"github.com/pacerclub/zebra-backend/internal/auth"
	"github.com/pacerclub/zebra-backend/internal/db"
	"github.com/pacerclub/zebra-backend/internal/handlers"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize database
	if err := db.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.CloseDB()

	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "https://zebra.pacerclub.cn"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Public routes
	r.Group(func(r chi.Router) {
		r.Post("/api/register", handlers.Register)
		r.Post("/api/login", handlers.Login)
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware)

		// Timer sessions
		r.Route("/api/sessions", func(r chi.Router) {
			r.Post("/", handlers.CreateSession)
			r.Get("/", handlers.ListSessions)
			r.Put("/{id}", handlers.UpdateSession)
			r.Delete("/{id}", handlers.DeleteSession)
		})

		// Projects
		r.Route("/api/projects", func(r chi.Router) {
			r.Post("/", handlers.CreateProject)
			r.Get("/", handlers.ListProjects)
			r.Put("/{id}", handlers.UpdateProject)
			r.Delete("/{id}", handlers.DeleteProject)
		})

		// Sync
		r.Route("/api/sync", func(r chi.Router) {
			r.Post("/", handlers.SyncData)
			r.Get("/status", handlers.SyncStatus)
		})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
