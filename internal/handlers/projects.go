package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func CreateProject(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement project creation
	w.WriteHeader(http.StatusNotImplemented)
}

func ListProjects(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement project listing
	w.WriteHeader(http.StatusNotImplemented)
}

func UpdateProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing project ID", http.StatusBadRequest)
		return
	}
	// TODO: Implement project update
	w.WriteHeader(http.StatusNotImplemented)
}

func DeleteProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Missing project ID", http.StatusBadRequest)
		return
	}
	// TODO: Implement project deletion
	w.WriteHeader(http.StatusNotImplemented)
}
