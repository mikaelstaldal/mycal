package handler

import (
	"errors"
	"net/http"

	"github.com/mikaelstaldal/mycal/internal/model"
	"github.com/mikaelstaldal/mycal/internal/service"
)

func registerFeedRoutes(mux *http.ServeMux, feedSvc *service.FeedService) {
	mux.HandleFunc("GET /api/v1/feeds", listFeeds(feedSvc))
	mux.HandleFunc("POST /api/v1/feeds", createFeed(feedSvc))
	mux.HandleFunc("GET /api/v1/feeds/{id}", getFeed(feedSvc))
	mux.HandleFunc("PUT /api/v1/feeds/{id}", updateFeed(feedSvc))
	mux.HandleFunc("DELETE /api/v1/feeds/{id}", deleteFeed(feedSvc))
	mux.HandleFunc("POST /api/v1/feeds/{id}/refresh", refreshFeed(feedSvc))
}

func setFeedStringID(f *model.Feed) {
	f.SetStringID()
}

func setFeedStringIDs(feeds []model.Feed) {
	for i := range feeds {
		feeds[i].SetStringID()
	}
}

func listFeeds(feedSvc *service.FeedService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		feeds, err := feedSvc.List()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list feeds")
			return
		}
		setFeedStringIDs(feeds)
		writeJSON(w, http.StatusOK, feeds)
	}
}

func createFeed(feedSvc *service.FeedService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req model.CreateFeedRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		feed, err := feedSvc.Create(&req)
		if err != nil {
			if errors.Is(err, service.ErrValidation) {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to create feed")
			return
		}
		setFeedStringID(feed)
		writeJSON(w, http.StatusCreated, feed)
	}
}

func getFeed(feedSvc *service.FeedService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := model.ParseFeedID(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		feed, err := feedSvc.GetByID(id)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				writeError(w, http.StatusNotFound, "feed not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to get feed")
			return
		}
		setFeedStringID(feed)
		writeJSON(w, http.StatusOK, feed)
	}
}

func updateFeed(feedSvc *service.FeedService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := model.ParseFeedID(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		var req model.UpdateFeedRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		feed, err := feedSvc.Update(id, &req)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				writeError(w, http.StatusNotFound, "feed not found")
				return
			}
			if errors.Is(err, service.ErrValidation) {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to update feed")
			return
		}
		setFeedStringID(feed)
		writeJSON(w, http.StatusOK, feed)
	}
}

func deleteFeed(feedSvc *service.FeedService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := model.ParseFeedID(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		if err := feedSvc.Delete(id); err != nil {
			if errors.Is(err, service.ErrNotFound) {
				writeError(w, http.StatusNotFound, "feed not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to delete feed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func refreshFeed(feedSvc *service.FeedService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := model.ParseFeedID(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid id")
			return
		}
		feed, err := feedSvc.RefreshFeed(id)
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				writeError(w, http.StatusNotFound, "feed not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to refresh feed")
			return
		}
		setFeedStringID(feed)
		writeJSON(w, http.StatusOK, feed)
	}
}
