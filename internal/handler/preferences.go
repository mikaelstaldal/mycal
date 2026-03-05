package handler

import (
	"errors"
	"net/http"

	"github.com/mikaelstaldal/mycal/internal/service"
)

func registerPreferencesRoutes(mux *http.ServeMux, svc *service.PreferencesService) {
	mux.HandleFunc("GET /api/v1/preferences", getPreferences(svc))
	mux.HandleFunc("PATCH /api/v1/preferences", updatePreferences(svc))
}

func getPreferences(svc *service.PreferencesService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		prefs, err := svc.GetAll()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to get preferences")
			return
		}
		writeJSON(w, http.StatusOK, prefs)
	}
}

func updatePreferences(svc *service.PreferencesService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var prefs map[string]string
		if err := readJSON(r, &prefs); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		result, err := svc.Update(prefs)
		if err != nil {
			if errors.Is(err, service.ErrValidation) {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to update preferences")
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}
