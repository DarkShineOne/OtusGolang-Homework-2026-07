package internalhttp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/app"
	"github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/logger"
	memorystorage "github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/storage/memory"
	"github.com/stretchr/testify/require"
)

func newRealAppServer(t *testing.T, logs *bytes.Buffer) *Server {
	t.Helper()
	logg, err := logger.New("info", logs)
	require.NoError(t, err)
	application := app.New(logg, memorystorage.New())
	return NewServer(logg, application, "127.0.0.1", 0)
}

func doRequest(s *Server, method, target string, body io.Reader) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, body)
	request.RemoteAddr = "192.0.2.1:12345"
	response := httptest.NewRecorder()
	s.server.Handler.ServeHTTP(response, request)
	return response
}

func decodeEventResponse(t *testing.T, body *bytes.Buffer) eventResponse {
	t.Helper()
	var response eventResponse
	require.NoError(t, json.NewDecoder(body).Decode(&response))
	return response
}

func decodeListResponse(t *testing.T, body *bytes.Buffer) listEventsResponse {
	t.Helper()
	var response listEventsResponse
	require.NoError(t, json.NewDecoder(body).Decode(&response))
	return response
}

const createEventBody = `{"id":"ev1","title":"Standup","starts_at":"2026-11-25T10:00:00Z",` +
	`"duration":"1h","description":"daily sync","user_id":"u1","notify_before":"30m"}`

func TestCreateEvent(t *testing.T) {
	var logs bytes.Buffer
	server := newRealAppServer(t, &logs)

	response := doRequest(server, http.MethodPost, "/events", strings.NewReader(createEventBody))
	require.Equal(t, http.StatusCreated, response.Code)

	created := decodeEventResponse(t, response.Body)
	require.Equal(t, "ev1", created.ID)
	require.Equal(t, "Standup", created.Title)
	require.Equal(t, "2026-11-25T10:00:00Z", created.StartsAt)
	require.Equal(t, "1h0m0s", created.Duration)
	require.Equal(t, "u1", created.UserID)
	require.Equal(t, "30m0s", created.NotifyBefore)
	require.Contains(t, logs.String(), "POST /events HTTP/1.1 201")
}

func TestCreateEventValidation(t *testing.T) {
	var logs bytes.Buffer
	server := newRealAppServer(t, &logs)

	tests := map[string]string{
		"missing id":        `{"title":"t","starts_at":"2026-11-25T10:00:00Z","duration":"1h"}`,
		"missing title":     `{"id":"ev1","starts_at":"2026-11-25T10:00:00Z","duration":"1h"}`,
		"bad starts_at":     `{"id":"ev1","title":"t","starts_at":"25.11.2026","duration":"1h"}`,
		"missing duration":  `{"id":"ev1","title":"t","starts_at":"2026-11-25T10:00:00Z"}`,
		"bad duration":      `{"id":"ev1","title":"t","starts_at":"2026-11-25T10:00:00Z","duration":"soon"}`,
		"negative duration": `{"id":"ev1","title":"t","starts_at":"2026-11-25T10:00:00Z","duration":"-1h"}`,
		"unknown field":     `{"id":"ev1","title":"t","starts_at":"2026-11-25T10:00:00Z","duration":"1h","x":1}`,
		"malformed json":    `{"id":`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			response := doRequest(server, http.MethodPost, "/events", strings.NewReader(body))
			require.Equal(t, http.StatusBadRequest, response.Code)

			var decoded errorResponse
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &decoded))
			require.NotEmpty(t, decoded.Error)
		})
	}
}

func TestEventCRUDFlow(t *testing.T) {
	var logs bytes.Buffer
	server := newRealAppServer(t, &logs)

	require.Equal(t, http.StatusCreated,
		doRequest(server, http.MethodPost, "/events", strings.NewReader(createEventBody)).Code)

	for _, target := range []string{
		"/events/day?date=2026-11-25",
		"/events/week?date=2026-11-23T00:00:00Z",
		"/events/month?date=2026-11-01T00:00:00Z",
	} {
		response := doRequest(server, http.MethodGet, target, nil)
		require.Equal(t, http.StatusOK, response.Code)
		require.Len(t, decodeListResponse(t, response.Body).Events, 1, target)
	}

	empty := doRequest(server, http.MethodGet, "/events/day?date=2026-11-26", nil)
	require.Equal(t, http.StatusOK, empty.Code)
	require.Empty(t, decodeListResponse(t, empty.Body).Events)

	update := `{"title":"Standup v2","starts_at":"2026-11-25T12:00:00Z","duration":"30m","user_id":"u1"}`
	response := doRequest(server, http.MethodPut, "/events/ev1", strings.NewReader(update))
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "Standup v2", decodeEventResponse(t, response.Body).Title)

	require.Equal(t, http.StatusNoContent,
		doRequest(server, http.MethodDelete, "/events/ev1", nil).Code)

	afterDelete := doRequest(server, http.MethodGet, "/events/day?date=2026-11-25", nil)
	require.Empty(t, decodeListResponse(t, afterDelete.Body).Events)
	require.Contains(t, logs.String(), "DELETE /events/ev1 HTTP/1.1 204")
}

func TestEventErrorCodes(t *testing.T) {
	var logs bytes.Buffer
	server := newRealAppServer(t, &logs)

	require.Equal(t, http.StatusCreated,
		doRequest(server, http.MethodPost, "/events", strings.NewReader(createEventBody)).Code)

	duplicate := `{"id":"ev1","title":"Other","starts_at":"2026-12-25T10:00:00Z","duration":"1h"}`
	require.Equal(t, http.StatusConflict,
		doRequest(server, http.MethodPost, "/events", strings.NewReader(duplicate)).Code)

	busy := `{"id":"ev2","title":"Busy","starts_at":"2026-11-25T10:30:00Z","duration":"1h","user_id":"u1"}`
	require.Equal(t, http.StatusConflict,
		doRequest(server, http.MethodPost, "/events", strings.NewReader(busy)).Code)

	missing := `{"title":"t","starts_at":"2026-11-25T10:00:00Z","duration":"1h"}`
	require.Equal(t, http.StatusNotFound,
		doRequest(server, http.MethodPut, "/events/unknown", strings.NewReader(missing)).Code)
	require.Equal(t, http.StatusNotFound,
		doRequest(server, http.MethodDelete, "/events/unknown", nil).Code)
}

func TestListEventsValidation(t *testing.T) {
	var logs bytes.Buffer
	server := newRealAppServer(t, &logs)

	require.Equal(t, http.StatusBadRequest,
		doRequest(server, http.MethodGet, "/events/day", nil).Code)
	require.Equal(t, http.StatusBadRequest,
		doRequest(server, http.MethodGet, "/events/week?date=yesterday", nil).Code)
}
