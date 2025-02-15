package main

import (
	"log"
	"net/http"
	"os"
	"strings"
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

	// Get allowed origins from environment variable
	allowedOrigins := []string{"https://zebra.pacerclub.cn"}
	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		allowedOrigins = strings.Split(origins, ",")
	}
	log.Printf("Allowed origins: %v", allowedOrigins)

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Requested-With", "Origin"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Global middleware to ensure CORS headers are always set
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				origin = "*"
			}
			for _, allowed := range allowedOrigins {
				if allowed == "*" || allowed == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token, X-Requested-With, Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	})

	// Public routes
	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.HandleFunc("/register", handlers.Register).Methods("POST", "OPTIONS")
			r.HandleFunc("/login", handlers.Login).Methods("POST", "OPTIONS")
			r.HandleFunc("/preferences", handlers.UpdatePreferences).Methods("POST", "OPTIONS")
		})

		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware)

			// Protected routes
			r.HandleFunc("/sync", handlers.SyncData).Methods("GET", "POST", "OPTIONS")
			r.HandleFunc("/sync/status", handlers.SyncStatus).Methods("GET", "OPTIONS")
		})
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
