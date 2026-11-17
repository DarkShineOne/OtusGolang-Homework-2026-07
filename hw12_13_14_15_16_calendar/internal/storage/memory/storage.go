package memorystorage

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/storage"
)

type Storage struct {
	events map[string]storage.Event
	mu     sync.RWMutex
}

func New() *Storage {
	return &Storage{events: make(map[string]storage.Event)}
}

func (s *Storage) CreateEvent(_ context.Context, event storage.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.events[event.ID]; ok {
		return storage.ErrEventExists
	}
	if s.dateBusy(event) {
		return storage.ErrDateBusy
	}

	s.events[event.ID] = event
	return nil
}

func (s *Storage) UpdateEvent(_ context.Context, event storage.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.events[event.ID]; !ok {
		return storage.ErrEventNotFound
	}
	if s.dateBusy(event) {
		return storage.ErrDateBusy
	}

	s.events[event.ID] = event
	return nil
}

func (s *Storage) DeleteEvent(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.events[id]; !ok {
		return storage.ErrEventNotFound
	}
	delete(s.events, id)
	return nil
}

func (s *Storage) ListEvents(
	_ context.Context,
	start, end time.Time,
) ([]storage.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := make([]storage.Event, 0)
	for _, event := range s.events {
		if event.StartsAt.Before(end) && start.Before(event.EndsAt()) {
			events = append(events, event)
		}
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].StartsAt.Equal(events[j].StartsAt) {
			return events[i].ID < events[j].ID
		}
		return events[i].StartsAt.Before(events[j].StartsAt)
	})
	return events, nil
}

func (s *Storage) dateBusy(candidate storage.Event) bool {
	for _, event := range s.events {
		if event.ID == candidate.ID || event.UserID != candidate.UserID {
			continue
		}
		if candidate.StartsAt.Before(event.EndsAt()) && event.StartsAt.Before(candidate.EndsAt()) {
			return true
		}
	}
	return false
}
