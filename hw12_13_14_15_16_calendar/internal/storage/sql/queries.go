package sqlstorage

const dateBusyQuery = `SELECT EXISTS (
    SELECT 1 FROM events
    WHERE user_id = $1
      AND id != $2
      AND starts_at < $4
      AND $3 < starts_at + (duration / 1000.0) * INTERVAL '1 microsecond'
)`

const eventExistsQuery = `SELECT EXISTS (SELECT 1 FROM events WHERE id = $1)`

const insertEventQuery = `
    INSERT INTO events (id, title, starts_at, duration, description, user_id, notify_before)
    VALUES ($1, $2, $3, $4, $5, $6, $7)`

const updateEventQuery = `
    UPDATE events
    SET title = $2, starts_at = $3, duration = $4,
        description = $5, user_id = $6, notify_before = $7
    WHERE id = $1`

const deleteEventQuery = `DELETE FROM events WHERE id = $1`

const listEventsQuery = `
    SELECT id, title, starts_at, duration, description, user_id, notify_before
    FROM events
    WHERE starts_at < $2
      AND $1 < starts_at + (duration / 1000.0) * INTERVAL '1 microsecond'
    ORDER BY starts_at, id`
