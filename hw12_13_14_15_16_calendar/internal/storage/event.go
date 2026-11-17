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

type Storage interface {
	CreateEvent(ctx context.Context, event Event) error
	UpdateEvent(ctx context.Context, event Event) error
	DeleteEvent(ctx context.Context, id string) error
	ListEvents(ctx context.Context, start, end time.Time) ([]Event, error)
}
