package memorystorage

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageCRUD(t *testing.T) {
	ctx := context.Background()
	store := New()
	start := time.Date(2026, time.November, 15, 12, 0, 0, 0, time.UTC)
	event := testEvent("event-1", "user-1", start, time.Hour)

	require.NoError(t, store.CreateEvent(ctx, event))
	require.ErrorIs(t, store.CreateEvent(ctx, event), storage.ErrEventExists)
	err := store.CreateEvent(ctx, testEvent("event-2", "user-1", start.Add(time.Minute), time.Hour))
	require.ErrorIs(t, err, storage.ErrDateBusy)
	require.NoError(t, store.CreateEvent(ctx, testEvent("other-user", "user-2", start, time.Hour)))

	adjacent := testEvent("event-2", "user-1", start.Add(time.Hour), 30*time.Minute)
	require.NoError(t, store.CreateEvent(ctx, adjacent))

	event.Title = "updated"
	require.NoError(t, store.UpdateEvent(ctx, event))
	missing := testEvent("missing", "user-1", start.AddDate(0, 0, 1), time.Hour)
	require.ErrorIs(t, store.UpdateEvent(ctx, missing), storage.ErrEventNotFound)

	events, err := store.ListEvents(ctx, start.Add(30*time.Minute), start.Add(90*time.Minute))
	require.NoError(t, err)
	require.Len(t, events, 3)
	require.Equal(t, []string{"event-1", "other-user", "event-2"}, []string{
		events[0].ID, events[1].ID, events[2].ID,
	})
	require.Equal(t, "updated", events[0].Title)

	require.NoError(t, store.DeleteEvent(ctx, event.ID))
	require.ErrorIs(t, store.DeleteEvent(ctx, event.ID), storage.ErrEventNotFound)
}

func TestStorageConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	store := New()
	start := time.Date(2026, time.November, 15, 12, 0, 0, 0, time.UTC)

	const eventCount = 100
	var wg sync.WaitGroup
	for i := 0; i < eventCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			event := testEvent(fmt.Sprintf("event-%d", i), fmt.Sprintf("user-%d", i), start, time.Hour)
			assert.NoError(t, store.CreateEvent(ctx, event))
			_, err := store.ListEvents(ctx, start, start.Add(time.Hour))
			assert.NoError(t, err)
		}(i)
	}
	wg.Wait()

	events, err := store.ListEvents(ctx, start, start.Add(time.Hour))
	require.NoError(t, err)
	require.Len(t, events, eventCount)
}

func testEvent(id, userID string, start time.Time, duration time.Duration) storage.Event {
	return storage.Event{
		ID:           id,
		Title:        "title",
		StartsAt:     start,
		Duration:     duration,
		Description:  "description",
		UserID:       userID,
		NotifyBefore: 15 * time.Minute,
	}
}
