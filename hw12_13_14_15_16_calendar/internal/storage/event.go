package storage

import (
	"context"
	"errors"
	"time"
)

var (
	ErrDateBusy      = errors.New("date is busy")
	ErrEventNotFound = errors.New("event not found")
	ErrEventExists   = errors.New("event already exists")
)

type Event struct {
	ID           string
	Title        string
	StartsAt     time.Time
	Duration     time.Duration
	Description  string
	UserID       string
	NotifyBefore time.Duration
}

func (e Event) EndsAt() time.Time {
	return e.StartsAt.Add(e.Duration)
}

// Validate checks the domain invariants of the event.
func (e Event) Validate() error {
	switch {
	case e.Title == "":
		return errors.New("event title is required")
	case e.StartsAt.IsZero():
		return errors.New("event starts_at is required")
	case e.Duration < 0:
		return errors.New("event duration must not be negative")
	case e.NotifyBefore < 0:
		return errors.New("event notify_before must not be negative")
	}
	return nil
}

type Storage interface {
	CreateEvent(ctx context.Context, event Event) error
	UpdateEvent(ctx context.Context, event Event) error
	DeleteEvent(ctx context.Context, id string) error
	ListEvents(ctx context.Context, start, end time.Time) ([]Event, error)
}
