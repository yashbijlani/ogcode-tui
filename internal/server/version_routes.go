package server

import (
	"encoding/json"
	"net/http"
)

// handleVersion returns current version and update information.
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	response, err := s.versionManager.GetResponse()
	if err != nil {
		// Return basic info even if update check fails
		response = s.versionManager.GetResponseFallback()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleVersionCheck forces a fresh update check (ignoring cache).
func (s *Server) handleVersionCheck(w http.ResponseWriter, r *http.Request) {
	// Create a new manager to bypass cache
	s.versionManager.ClearCache()
	response, err := s.versionManager.GetResponse()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
