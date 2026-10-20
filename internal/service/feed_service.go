package service

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/mikaelstaldal/go-server-common/httputil"
	"github.com/mikaelstaldal/mycal/internal/api"
	"github.com/mikaelstaldal/mycal/internal/ical"
	"github.com/mikaelstaldal/mycal/internal/model"
	"github.com/mikaelstaldal/mycal/internal/repository"
)

const maxFeedImportSize = 10 * 1024 * 1024 // 10 MiB
const maxFeedEvents = 10000

var errFeedLimit = errors.New("feed exceeds import limit")

type FeedService struct {
	feedRepo  repository.FeedRepository
	eventRepo repository.EventRepository
	calRepo   repository.CalendarRepository
}

func NewFeedService(feedRepo repository.FeedRepository, eventRepo repository.EventRepository, calRepo repository.CalendarRepository) *FeedService {
	return &FeedService{feedRepo: feedRepo, eventRepo: eventRepo, calRepo: calRepo}
}

func (s *FeedService) Create(req *api.CreateFeedRequest) (*model.Feed, error) {
	refreshInterval, err := ValidateCreateFeedRequest(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	rawURL := req.URL.String()
	if _, err := feedFetchURL(rawURL); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}

	calendarName := req.CalendarName.Or("")
	calendarColor := req.CalendarColor.Or("")
	calendarID, err := s.resolveCalendarName(calendarName, calendarColor)
	if err != nil {
		return nil, err
	}

	feed := &model.Feed{
		URL:                    rawURL,
		CalendarID:             calendarID,
		RefreshIntervalMinutes: refreshInterval,
		Enabled:                true,
	}
	if err := s.feedRepo.CreateFeed(feed); err != nil {
		return nil, err
	}
	return feed, nil
}

func (s *FeedService) GetByID(id int64) (*model.Feed, error) {
	f, err := s.feedRepo.GetFeedByID(id)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, ErrNotFound
	}
	return f, nil
}

func (s *FeedService) List() ([]model.Feed, error) {
	feeds, err := s.feedRepo.ListFeeds()
	if err != nil {
		return nil, err
	}
	if feeds == nil {
		feeds = []model.Feed{}
	}
	return feeds, nil
}

func (s *FeedService) Update(id int64, req *api.UpdateFeedRequest) (*model.Feed, error) {
	if err := ValidateUpdateFeedRequest(req); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	existing, err := s.feedRepo.GetFeedByID(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrNotFound
	}
	if req.URL.Set {
		rawURL := req.URL.Value.String()
		if _, err := feedFetchURL(rawURL); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
		}
		existing.URL = rawURL
	}
	if req.CalendarName.Set {
		calID, err := s.resolveCalendarName(req.CalendarName.Value, "")
		if err != nil {
			return nil, err
		}
		existing.CalendarID = calID
	}
	if req.RefreshIntervalMinutes.Set {
		existing.RefreshIntervalMinutes = req.RefreshIntervalMinutes.Value
	}
	if req.Enabled.Set {
		existing.Enabled = req.Enabled.Value
	}
	if err := s.feedRepo.UpdateFeed(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *FeedService) Delete(id int64) error {
	feed, err := s.feedRepo.GetFeedByID(id)
	if err != nil {
		return err
	}
	if feed == nil {
		return ErrNotFound
	}
	calendarID := feed.CalendarID
	if err := s.feedRepo.DeleteFeed(id); err != nil {
		return err
	}
	if calendarID != 0 {
		_ = s.calRepo.DeleteCalendarIfUnused(calendarID)
	}
	return nil
}

func (s *FeedService) RefreshFeed(id int64) (*model.Feed, error) {
	feed, err := s.feedRepo.GetFeedByID(id)
	if err != nil {
		return nil, err
	}
	if feed == nil {
		return nil, ErrNotFound
	}
	s.doRefresh(feed)
	return feed, nil
}

func (s *FeedService) resolveCalendarName(name, color string) (int64, error) {
	if name == "" {
		return 0, nil
	}
	cal, err := s.calRepo.GetCalendarByName(name)
	if err != nil {
		return 0, err
	}
	if cal != nil {
		return cal.ID, nil
	}
	if color == "" {
		color = "dodgerblue"
	}
	newCal := &model.Calendar{Name: name, Color: color}
	if err := s.calRepo.CreateCalendar(newCal); err != nil {
		return 0, err
	}
	return newCal.ID, nil
}

func (s *FeedService) doRefresh(feed *model.Feed) {
	eventColor := ""
	if feed.CalendarID != 0 {
		if cal, err := s.calRepo.GetCalendarByID(feed.CalendarID); err == nil && cal != nil {
			eventColor = cal.Color
		}
	}
	imported, err := s.fetchAndImport(feed, eventColor)
	now := time.Now().UTC().Format(time.RFC3339)
	feed.LastRefreshedAt = now
	if err != nil {
		log.Printf("feed %d refresh error: %v", feed.ID, err)
		feed.LastError = "feed refresh failed"
		if errors.Is(err, errFeedLimit) {
			feed.LastError = errFeedLimit.Error()
		}
	} else {
		feed.LastError = ""
		if imported > 0 {
			log.Printf("feed %d: synchronized %d events", feed.ID, imported)
		}
	}
	if updateErr := s.feedRepo.UpdateFeedRefreshStatus(feed); updateErr != nil {
		log.Printf("feed %d: failed to update after refresh: %v", feed.ID, updateErr)
	}
}

func (s *FeedService) fetchAndImport(feed *model.Feed, eventColor string) (int, error) {
	// Re-validate stored feed URLs on every refresh, not just at create time.
	fetchURL, err := feedFetchURL(feed.URL)
	if err != nil {
		return 0, err
	}
	client := httputil.NewSafeHTTPClient(30 * time.Second)
	// This client serves one fetch; do not retain idle connections.
	client.Transport.(*http.Transport).DisableKeepAlives = true
	resp, err := client.Get(fetchURL)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch URL: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("URL returned status %d", resp.StatusCode)
	}

	return s.importFeedSnapshot(feed, eventColor, resp.Body)
}

// importFeedSnapshot validates the entire response before replacing stored events.
func (s *FeedService) importFeedSnapshot(feed *model.Feed, eventColor string, body io.Reader) (int, error) {
	data, err := io.ReadAll(io.LimitReader(body, maxFeedImportSize+1))
	if err != nil {
		return 0, err
	}
	if len(data) > maxFeedImportSize {
		return 0, errFeedLimit
	}
	events, err := ical.DecodeSnapshot(bytes.NewReader(data))
	if err != nil {
		return 0, fmt.Errorf("failed to parse iCalendar data: %w", err)
	}
	if len(events) > maxFeedEvents {
		return 0, errFeedLimit
	}
	snapshot := make([]model.Event, 0, len(events))
	for _, e := range events {
		if e.RecurrenceOriginalStart != "" {
			continue
		}
		ev, err := buildEventForImport(e)
		if err != nil {
			return 0, fmt.Errorf("invalid feed event: %w", err)
		}
		ev.CalendarID = feed.CalendarID
		if eventColor != "" && ev.Color == "" {
			ev.Color = eventColor
		}
		snapshot = append(snapshot, *ev)
	}
	if err := s.feedRepo.ReplaceFeedEvents(feed, snapshot); err != nil {
		return 0, err
	}
	return len(snapshot), nil
}

// feedFetchURL keeps the subscription URI as entered, but fetches webcal(s) over
// HTTPS. The shared safe client accepts only HTTP(S) and checks each connection
// and redirect for private addresses.
func feedFetchURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL")
	}
	if u.Scheme == "webcal" || u.Scheme == "webcals" {
		u.Scheme = "https"
	}
	fetchURL := u.String()
	if err := httputil.ValidateExternalURL(fetchURL); err != nil {
		return "", err
	}
	return fetchURL, nil
}

func (s *FeedService) RefreshAllDue() {
	feeds, err := s.feedRepo.ListFeeds()
	if err != nil {
		log.Printf("feed refresh: failed to list feeds: %v", err)
		return
	}
	now := time.Now().UTC()
	for i := range feeds {
		f := &feeds[i]
		if !f.Enabled {
			continue
		}
		if f.LastRefreshedAt != "" {
			lastRefreshed, err := time.Parse(time.RFC3339, f.LastRefreshedAt)
			if err == nil {
				nextRefresh := lastRefreshed.Add(time.Duration(f.RefreshIntervalMinutes) * time.Minute)
				if now.Before(nextRefresh) {
					continue
				}
			}
		}
		s.doRefresh(f)
	}
}
