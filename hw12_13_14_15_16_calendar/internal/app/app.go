package app

import (
	"context"
	"time"

	"github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/storage"
)

type App struct {
	logger  Logger
	storage storage.Storage
}

type Logger interface {
	Info(msg string)
}

func New(logger Logger, eventStorage storage.Storage) *App {
	return &App{logger: logger, storage: eventStorage}
}

func (a *App) CreateEvent(ctx context.Context, event storage.Event) error {
	if err := a.storage.CreateEvent(ctx, event); err != nil {
		return err
	}
	a.logger.Info("event created: " + event.ID)
	return nil
}

func (a *App) UpdateEvent(ctx context.Context, event storage.Event) error {
	if err := a.storage.UpdateEvent(ctx, event); err != nil {
		return err
	}
	a.logger.Info("event updated: " + event.ID)
	return nil
}

func (a *App) DeleteEvent(ctx context.Context, id string) error {
	if err := a.storage.DeleteEvent(ctx, id); err != nil {
		return err
	}
	a.logger.Info("event deleted: " + id)
	return nil
}

func (a *App) ListEventsForDay(ctx context.Context, day time.Time) ([]storage.Event, error) {
	return a.storage.ListEvents(ctx, day, day.AddDate(0, 0, 1))
}

func (a *App) ListEventsForWeek(ctx context.Context, weekStart time.Time) ([]storage.Event, error) {
	return a.storage.ListEvents(ctx, weekStart, weekStart.AddDate(0, 0, 7))
}

func (a *App) ListEventsForMonth(ctx context.Context, monthStart time.Time) ([]storage.Event, error) {
	return a.storage.ListEvents(ctx, monthStart, monthStart.AddDate(0, 1, 0))
}
