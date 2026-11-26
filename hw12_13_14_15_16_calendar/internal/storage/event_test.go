package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEventValidate(t *testing.T) {
	base := func() Event {
		return Event{
			ID:           "ev1",
			Title:        "Standup",
			StartsAt:     time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC),
			Duration:     time.Hour,
			NotifyBefore: 30 * time.Minute,
		}
	}

	require.NoError(t, base().Validate())

	noTitle := base()
	noTitle.Title = ""
	require.ErrorContains(t, noTitle.Validate(), "title")

	noStartsAt := base()
	noStartsAt.StartsAt = time.Time{}
	require.ErrorContains(t, noStartsAt.Validate(), "starts_at")

	negativeDuration := base()
	negativeDuration.Duration = -time.Minute
	require.ErrorContains(t, negativeDuration.Validate(), "duration")

	negativeNotify := base()
	negativeNotify.NotifyBefore = -time.Minute
	require.ErrorContains(t, negativeNotify.Validate(), "notify_before")

	zeroOptional := base()
	zeroOptional.NotifyBefore = 0
	require.NoError(t, zeroOptional.Validate())
}
