package sqlstorage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/storage"
	"github.com/stretchr/testify/require"
)

func TestPostgresIntegration(t *testing.T) {
	dsn := os.Getenv("CALENDAR_TEST_DSN")
	if dsn == "" {
		t.Skip("CALENDAR_TEST_DSN is not set")
	}

	ctx := context.Background()
	store, err := New(dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = store.Close(context.Background())
	})
	require.NoError(t, store.Connect(ctx))
	applySchema(ctx, t, store.db)

	suffix := time.Now().UnixNano()
	start := time.Now().UTC().Truncate(time.Second).Add(24 * time.Hour)
	first := storage.Event{
		ID:           fmt.Sprintf("integration-%d-1", suffix),
		Title:        "first",
		StartsAt:     start,
		Duration:     time.Hour,
		Description:  "integration test",
		UserID:       fmt.Sprintf("integration-user-%d", suffix),
		NotifyBefore: 15 * time.Minute,
	}
	second := first
	second.ID = fmt.Sprintf("integration-%d-2", suffix)
	second.Title = "second"
	second.StartsAt = start.Add(time.Hour)
	t.Cleanup(func() {
		_ = store.DeleteEvent(context.Background(), first.ID)
		_ = store.DeleteEvent(context.Background(), second.ID)
	})

	require.NoError(t, store.CreateEvent(ctx, first))
	require.NoError(t, store.CreateEvent(ctx, second))
	conflicting := first
	conflicting.ID = fmt.Sprintf("integration-%d-3", suffix)
	conflicting.StartsAt = start.Add(30 * time.Minute)
	require.ErrorIs(t, store.CreateEvent(ctx, conflicting), storage.ErrDateBusy)

	first.Title = "updated"
	require.NoError(t, store.UpdateEvent(ctx, first))
	events, err := store.ListEvents(ctx, start, start.AddDate(0, 0, 1))
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, "updated", events[0].Title)
	require.Equal(t, second.ID, events[1].ID)

	require.NoError(t, store.DeleteEvent(ctx, first.ID))
	require.ErrorIs(t, store.DeleteEvent(ctx, first.ID), storage.ErrEventNotFound)
}

// applySchema выполняет Up-секцию первого файла миграций,
// подменяя в тесте применение миграций из CLI.
func applySchema(ctx context.Context, t *testing.T, db *sql.DB) {
	t.Helper()
	path := filepath.Join("..", "..", "..", "migrations", "000001_create_events_table.sql")
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	upSection := strings.TrimPrefix(strings.Split(string(data), "-- +goose Down")[0], "-- +goose Up")
	for _, statement := range strings.Split(upSection, ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		_, err := db.ExecContext(ctx, statement)
		require.NoError(t, err)
	}
}
