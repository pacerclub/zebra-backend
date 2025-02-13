package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func CreateSession(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement session creation
	w.WriteHeader(http.StatusNotImplemented)
}

func ListSessions(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement session listing
	w.WriteHeader(http.StatusNotImplemented)
}

func UpdateSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}
	// TODO: Implement session update
	w.WriteHeader(http.StatusNotImplemented)
}

func DeleteSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}
	// TODO: Implement session deletion
	w.WriteHeader(http.StatusNotImplemented)
}
