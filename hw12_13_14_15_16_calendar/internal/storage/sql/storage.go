package sqlstorage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/storage"
	"github.com/jackc/pgx/v5/pgconn"
)

const maxTxAttempts = 3

type Storage struct {
	db *sql.DB
}

func New(dsn string) (*Storage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	return &Storage{db: db}, nil
}

// Connect проверяет доступность PostgreSQL.
func (s *Storage) Connect(ctx context.Context) error {
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return nil
}

func (s *Storage) Close(_ context.Context) error {
	return s.db.Close()
}

func (s *Storage) CreateEvent(ctx context.Context, event storage.Event) error {
	for attempt := 1; ; attempt++ {
		err := s.createEvent(ctx, event)
		if err == nil || attempt >= maxTxAttempts || !isRetryableTxError(err) {
			return err
		}
	}
}

func (s *Storage) createEvent(ctx context.Context, event storage.Event) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("begin create event: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	busy, err := dateBusy(ctx, tx, event)
	if err != nil {
		return err
	}
	if busy {
		return storage.ErrDateBusy
	}

	_, err = tx.ExecContext(ctx, insertEventQuery,
		event.ID, event.Title, event.StartsAt, event.Duration,
		event.Description, event.UserID, event.NotifyBefore)
	if err != nil {
		return translateError("create event", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create event: %w", err)
	}
	return nil
}

func (s *Storage) UpdateEvent(ctx context.Context, event storage.Event) error {
	for attempt := 1; ; attempt++ {
		err := s.updateEvent(ctx, event)
		if err == nil || attempt >= maxTxAttempts || !isRetryableTxError(err) {
			return err
		}
	}
}

func (s *Storage) updateEvent(ctx context.Context, event storage.Event) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("begin update event: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	exists, err := eventExists(ctx, tx, event.ID)
	if err != nil {
		return err
	}

	if !exists {
		return storage.ErrEventNotFound
	}

	busy, err := dateBusy(ctx, tx, event)
	if err != nil {
		return err
	}

	if busy {
		return storage.ErrDateBusy
	}

	_, err = tx.ExecContext(ctx, updateEventQuery,
		event.ID, event.Title, event.StartsAt, event.Duration,
		event.Description, event.UserID, event.NotifyBefore)
	if err != nil {
		return translateError("update event", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update event: %w", err)
	}

	return nil
}

func (s *Storage) DeleteEvent(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, deleteEventQuery, id)
	if err != nil {
		return fmt.Errorf("delete event: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete event rows affected: %w", err)
	}

	if affected == 0 {
		return storage.ErrEventNotFound
	}

	return nil
}

func (s *Storage) ListEvents(
	ctx context.Context,
	start, end time.Time,
) ([]storage.Event, error) {
	rows, err := s.db.QueryContext(ctx, listEventsQuery, start, end)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	events := make([]storage.Event, 0)
	for rows.Next() {
		var event storage.Event
		var duration, notifyBefore int64
		if err := rows.Scan(
			&event.ID, &event.Title, &event.StartsAt, &duration,
			&event.Description, &event.UserID, &notifyBefore,
		); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}

		event.Duration = time.Duration(duration)
		event.NotifyBefore = time.Duration(notifyBefore)
		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}

	return events, nil
}

func dateBusy(ctx context.Context, tx *sql.Tx, event storage.Event) (bool, error) {
	var busy bool

	err := tx.QueryRowContext(
		ctx, dateBusyQuery,
		event.UserID, event.ID, event.StartsAt, event.EndsAt(),
	).Scan(&busy)
	if err != nil {
		return false, fmt.Errorf("check date busy: %w", err)
	}

	return busy, nil
}

func eventExists(ctx context.Context, tx *sql.Tx, id string) (bool, error) {
	var exists bool
	if err := tx.QueryRowContext(ctx, eventExistsQuery, id).Scan(&exists); err != nil {
		return false, fmt.Errorf("check event exists: %w", err)
	}

	return exists, nil
}

func translateError(operation string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("%s: %w", operation, storage.ErrEventExists)
	}

	return fmt.Errorf("%s: %w", operation, err)
}

func isRetryableTxError(err error) bool {
	var pgErr *pgconn.PgError

	return errors.As(err, &pgErr) && (pgErr.Code == "40001" || pgErr.Code == "40P01")
}
