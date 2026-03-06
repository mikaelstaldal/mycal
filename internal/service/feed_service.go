package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mikaelstaldal/mycal/internal/ical"
	"github.com/mikaelstaldal/mycal/internal/model"
	"github.com/mikaelstaldal/mycal/internal/repository"

)

const maxFeedImportSize = 10 * 1024 * 1024 // 10 MiB

type FeedService struct {
	feedRepo  repository.FeedRepository
	eventRepo repository.EventRepository
	calRepo   repository.CalendarRepository
}

func NewFeedService(feedRepo repository.FeedRepository, eventRepo repository.EventRepository, calRepo repository.CalendarRepository) *FeedService {
	return &FeedService{feedRepo: feedRepo, eventRepo: eventRepo, calRepo: calRepo}
}

func (s *FeedService) Create(req *model.CreateFeedRequest) (*model.Feed, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	if err := ValidateExternalURL(req.URL); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}

	calendarID, err := s.resolveCalendarName(req.CalendarName, req.CalendarColor)
	if err != nil {
		return nil, err
	}

	feed := &model.Feed{
		URL:                    req.URL,
		CalendarID:             calendarID,
		RefreshIntervalMinutes: req.RefreshIntervalMinutes,
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

func (s *FeedService) Update(id int64, req *model.UpdateFeedRequest) (*model.Feed, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	existing, err := s.feedRepo.GetFeedByID(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrNotFound
	}
	if req.URL != nil {
		if err := ValidateExternalURL(*req.URL); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
		}
		existing.URL = *req.URL
	}
	if req.CalendarName != nil {
		calID, err := s.resolveCalendarName(*req.CalendarName, "")
		if err != nil {
			return nil, err
		}
		existing.CalendarID = calID
	}
	if req.RefreshIntervalMinutes != nil {
		existing.RefreshIntervalMinutes = *req.RefreshIntervalMinutes
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if err := s.feedRepo.UpdateFeed(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *FeedService) Delete(id int64) error {
	err := s.feedRepo.DeleteFeed(id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
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
	imported, err := s.fetchAndImport(feed.URL, feed.CalendarID, eventColor)
	now := time.Now().UTC().Format(time.RFC3339)
	feed.LastRefreshedAt = now
	if err != nil {
		feed.LastError = err.Error()
		log.Printf("feed %d refresh error: %v", feed.ID, err)
	} else {
		feed.LastError = ""
		if imported > 0 {
			log.Printf("feed %d: imported %d new events", feed.ID, imported)
		}
	}
	if updateErr := s.feedRepo.UpdateFeed(feed); updateErr != nil {
		log.Printf("feed %d: failed to update after refresh: %v", feed.ID, updateErr)
	}
}

func (s *FeedService) fetchAndImport(feedURL string, calendarID int64, eventColor string) (int, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(feedURL)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch URL: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("URL returned status %d", resp.StatusCode)
	}

	events, err := ical.Decode(io.LimitReader(resp.Body, maxFeedImportSize))
	if err != nil {
		return 0, fmt.Errorf("failed to parse iCalendar data: %v", err)
	}

	imported := 0
	for _, e := range events {
		if e.RecurrenceOriginalStart != "" {
			continue // skip overrides for now
		}
		if e.ImportUID != "" {
			exists, err := s.eventRepo.ExistsByIcsUID(e.ImportUID)
			if err != nil {
				continue
			}
			if exists {
				continue
			}
		}
		ev, err := buildEventForImport(e)
		if err != nil {
			continue
		}
		ev.CalendarID = calendarID
		if eventColor != "" && ev.Color == "" {
			ev.Color = eventColor
		}
		if err := s.eventRepo.Create(ev); err != nil {
			continue
		}
		imported++
	}
	return imported, nil
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

// ValidateExternalURL checks that the URL is safe to fetch (not localhost or private IPs).
func ValidateExternalURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https")
	}
	hostname := u.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL must have a hostname")
	}
	lower := strings.ToLower(hostname)
	if lower == "localhost" || strings.HasSuffix(lower, ".localhost") {
		return fmt.Errorf("URL must not point to localhost")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, hostname)
	if err != nil {
		return fmt.Errorf("DNS lookup failed for %s: %v", hostname, err)
	}

	for _, ip := range ips {
		if ip.IP.IsLoopback() || ip.IP.IsPrivate() || ip.IP.IsLinkLocalUnicast() || ip.IP.IsLinkLocalMulticast() || ip.IP.IsUnspecified() {
			return fmt.Errorf("URL must not point to a private or local address")
		}
	}
	return nil
}
