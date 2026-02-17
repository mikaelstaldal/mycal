package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/mikaelstaldal/mycal/internal/service"
)

func NewRouter(svc *service.EventService) http.Handler {
	mux := http.NewServeMux()
	registerEventRoutes(mux, svc)
	return withMiddleware(mux)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("error encoding JSON: %v", err)
	}
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
