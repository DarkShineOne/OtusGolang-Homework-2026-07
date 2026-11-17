package sqlstorage

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/storage"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestConnectPings(t *testing.T) {
	store, mock := newMockStorage(t)
	mock.ExpectPing()

	require.NoError(t, store.Connect(context.Background()))
	assertExpectations(t, mock)
}

func TestCreateEvent(t *testing.T) {
	store, mock := newMockStorage(t)
	event := sqlTestEvent()
	mock.ExpectBegin()
	expectDateBusy(mock, event, false)
	mock.ExpectExec("INSERT INTO events").WithArgs(
		event.ID, event.Title, event.StartsAt, event.Duration,
		event.Description, event.UserID, event.NotifyBefore,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	require.NoError(t, store.CreateEvent(context.Background(), event))
	assertExpectations(t, mock)
}

func TestCreateEventSerializationRetry(t *testing.T) {
	store, mock := newMockStorage(t)
	event := sqlTestEvent()
	mock.ExpectBegin()
	expectDateBusy(mock, event, false)
	mock.ExpectExec("INSERT INTO events").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(&pgconn.PgError{Code: "40001"})
	mock.ExpectBegin()
	expectDateBusy(mock, event, false)
	mock.ExpectExec("INSERT INTO events").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	require.NoError(t, store.CreateEvent(context.Background(), event))
	assertExpectations(t, mock)
}

func TestCreateEventDateBusy(t *testing.T) {
	store, mock := newMockStorage(t)
	event := sqlTestEvent()
	mock.ExpectBegin()
	expectDateBusy(mock, event, true)
	mock.ExpectRollback()

	require.ErrorIs(t, store.CreateEvent(context.Background(), event), storage.ErrDateBusy)
	assertExpectations(t, mock)
}

func TestCreateEventAlreadyExists(t *testing.T) {
	store, mock := newMockStorage(t)
	event := sqlTestEvent()
	mock.ExpectBegin()
	expectDateBusy(mock, event, false)
	mock.ExpectExec("INSERT INTO events").WillReturnError(&pgconn.PgError{Code: "23505"})
	mock.ExpectRollback()

	require.ErrorIs(t, store.CreateEvent(context.Background(), event), storage.ErrEventExists)
	assertExpectations(t, mock)
}

func TestUpdateEventNotFound(t *testing.T) {
	store, mock := newMockStorage(t)
	event := sqlTestEvent()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT EXISTS \\(SELECT 1 FROM events WHERE id = \\$1\\)").
		WithArgs(event.ID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectRollback()

	require.ErrorIs(t, store.UpdateEvent(context.Background(), event), storage.ErrEventNotFound)
	assertExpectations(t, mock)
}

func TestDeleteEventNotFound(t *testing.T) {
	store, mock := newMockStorage(t)
	mock.ExpectExec("DELETE FROM events WHERE id = \\$1").
		WithArgs("missing").
		WillReturnResult(sqlmock.NewResult(0, 0))

	require.ErrorIs(t, store.DeleteEvent(context.Background(), "missing"), storage.ErrEventNotFound)
	assertExpectations(t, mock)
}

func TestListEvents(t *testing.T) {
	store, mock := newMockStorage(t)
	event := sqlTestEvent()
	mock.ExpectQuery("SELECT id, title, starts_at").
		WithArgs(event.StartsAt, event.StartsAt.Add(24*time.Hour)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "starts_at", "duration", "description", "user_id", "notify_before",
		}).AddRow(
			event.ID, event.Title, event.StartsAt, int64(event.Duration),
			event.Description, event.UserID, int64(event.NotifyBefore),
		))

	events, err := store.ListEvents(context.Background(), event.StartsAt, event.StartsAt.Add(24*time.Hour))
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, event.ID, events[0].ID)
	require.Equal(t, event.Duration, events[0].Duration)
	assertExpectations(t, mock)
}

func newMockStorage(t *testing.T) (*Storage, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})
	return &Storage{db: db}, mock
}

func expectDateBusy(mock sqlmock.Sqlmock, event storage.Event, busy bool) {
	mock.ExpectQuery(regexp.QuoteMeta(dateBusyQuery)).
		WithArgs(event.UserID, event.ID, event.StartsAt, event.EndsAt()).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(busy))
}

func assertExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	require.NoError(t, mock.ExpectationsWereMet())
}

func sqlTestEvent() storage.Event {
	return storage.Event{
		ID:           "event-1",
		Title:        "title",
		StartsAt:     time.Date(2026, time.November, 15, 12, 0, 0, 0, time.UTC),
		Duration:     time.Hour,
		Description:  "description",
		UserID:       "user-1",
		NotifyBefore: 15 * time.Minute,
	}
}
