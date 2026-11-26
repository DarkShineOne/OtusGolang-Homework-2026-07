package internalhttp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/storage"
)

const maxRequestSize = 1 << 20

type eventRequest struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	StartsAt     string `json:"starts_at"`
	Duration     string `json:"duration"`
	Description  string `json:"description"`
	UserID       string `json:"user_id"`
	NotifyBefore string `json:"notify_before"`
}

type eventResponse struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	StartsAt     string `json:"starts_at"`
	Duration     string `json:"duration"`
	Description  string `json:"description"`
	UserID       string `json:"user_id"`
	NotifyBefore string `json:"notify_before"`
}

type listEventsResponse struct {
	Events []eventResponse `json:"events"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (r eventRequest) toEvent() (storage.Event, error) {
	startsAt, err := parseTimestamp(r.StartsAt)
	if err != nil {
		return storage.Event{}, err
	}
	duration, err := parseDuration(r.Duration, true)
	if err != nil {
		return storage.Event{}, err
	}
	notifyBefore, err := parseDuration(r.NotifyBefore, false)
	if err != nil {
		return storage.Event{}, err
	}

	event := storage.Event{
		ID:           r.ID,
		Title:        r.Title,
		StartsAt:     startsAt,
		Duration:     duration,
		Description:  r.Description,
		UserID:       r.UserID,
		NotifyBefore: notifyBefore,
	}
	return event, event.Validate()
}

func eventToResponse(e storage.Event) eventResponse {
	return eventResponse{
		ID:           e.ID,
		Title:        e.Title,
		StartsAt:     e.StartsAt.Format(time.RFC3339),
		Duration:     e.Duration.String(),
		Description:  e.Description,
		UserID:       e.UserID,
		NotifyBefore: e.NotifyBefore.String(),
	}
}

func (s *Server) createEvent(w http.ResponseWriter, r *http.Request) {
	req, err := decodeEventRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.ID == "" {
		writeError(w, http.StatusBadRequest, errors.New("event id is required"))
		return
	}
	event, err := req.toEvent()
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.app.CreateEvent(r.Context(), event); err != nil {
		code, message := statusFromError(err)
		writeError(w, code, errors.New(message))
		return
	}
	writeJSON(w, http.StatusCreated, eventToResponse(event))
}

func (s *Server) updateEvent(w http.ResponseWriter, r *http.Request) {
	req, err := decodeEventRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	event, err := req.toEvent()
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	event.ID = r.PathValue("id")
	if err := s.app.UpdateEvent(r.Context(), event); err != nil {
		code, message := statusFromError(err)
		writeError(w, code, errors.New(message))
		return
	}
	writeJSON(w, http.StatusOK, eventToResponse(event))
}

func (s *Server) deleteEvent(w http.ResponseWriter, r *http.Request) {
	if err := s.app.DeleteEvent(r.Context(), r.PathValue("id")); err != nil {
		code, message := statusFromError(err)
		writeError(w, code, errors.New(message))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listDayEvents(w http.ResponseWriter, r *http.Request) {
	s.listEvents(w, r, s.app.ListEventsForDay)
}

func (s *Server) listWeekEvents(w http.ResponseWriter, r *http.Request) {
	s.listEvents(w, r, s.app.ListEventsForWeek)
}

func (s *Server) listMonthEvents(w http.ResponseWriter, r *http.Request) {
	s.listEvents(w, r, s.app.ListEventsForMonth)
}

type listFunc func(ctx context.Context, start time.Time) ([]storage.Event, error)

func (s *Server) listEvents(w http.ResponseWriter, r *http.Request, list listFunc) {
	start, err := parseDateParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	events, err := list(r.Context(), start)
	if err != nil {
		code, message := statusFromError(err)
		writeError(w, code, errors.New(message))
		return
	}

	response := listEventsResponse{Events: make([]eventResponse, 0, len(events))}
	for _, event := range events {
		response.Events = append(response.Events, eventToResponse(event))
	}
	writeJSON(w, http.StatusOK, response)
}

func decodeEventRequest(r *http.Request) (eventRequest, error) {
	var req eventRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, maxRequestSize))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return eventRequest{}, fmt.Errorf("decode request: %w", err)
	}
	return req, nil
}

func parseDateParam(r *http.Request) (time.Time, error) {
	value := r.URL.Query().Get("date")
	if value == "" {
		return time.Time{}, errors.New("date query parameter is required")
	}
	return parseTimestamp(value)
}

func parseTimestamp(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date %q: expected RFC3339 or YYYY-MM-DD", value)
}

func parseDuration(value string, required bool) (time.Duration, error) {
	if value == "" {
		if required {
			return 0, errors.New("duration is required")
		}
		return 0, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", value, err)
	}
	return parsed, nil
}

func statusFromError(err error) (int, string) {
	switch {
	case errors.Is(err, storage.ErrEventNotFound):
		return http.StatusNotFound, err.Error()
	case errors.Is(err, storage.ErrEventExists):
		return http.StatusConflict, err.Error()
	case errors.Is(err, storage.ErrDateBusy):
		return http.StatusConflict, err.Error()
	default:
		return http.StatusInternalServerError, "internal error"
	}
}

func writeError(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, errorResponse{Error: err.Error()})
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	data, err := json.Marshal(body)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_, _ = w.Write(data)
}
