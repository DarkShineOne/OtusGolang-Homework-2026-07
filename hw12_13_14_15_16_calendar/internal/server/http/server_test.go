package internalhttp

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/logger"
	"github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/storage"
	"github.com/stretchr/testify/require"
)

type testApplication struct{}

func (testApplication) CreateEvent(context.Context, storage.Event) error { return nil }
func (testApplication) UpdateEvent(context.Context, storage.Event) error { return nil }
func (testApplication) DeleteEvent(context.Context, string) error        { return nil }
func (testApplication) ListEventsForDay(context.Context, time.Time) ([]storage.Event, error) {
	return nil, nil
}

func (testApplication) ListEventsForWeek(context.Context, time.Time) ([]storage.Event, error) {
	return nil, nil
}

func (testApplication) ListEventsForMonth(context.Context, time.Time) ([]storage.Event, error) {
	return nil, nil
}

func TestHelloAndLoggingMiddleware(t *testing.T) {
	var logs bytes.Buffer
	logg, err := logger.New("info", &logs)
	require.NoError(t, err)
	server := NewServer(logg, testApplication{}, "127.0.0.1", 8080)

	request := httptest.NewRequest(http.MethodGet, "/hello?q=1", nil)
	request.RemoteAddr = "192.0.2.1:12345"
	request.Header.Set("User-Agent", "calendar-test")
	response := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "Hello, world!\n", response.Body.String())
	logLine := logs.String()
	for _, expected := range []string{
		"192.0.2.1 [", "GET /hello?q=1 HTTP/1.1 200", `"calendar-test"`,
	} {
		require.Contains(t, logLine, expected)
	}
}

func TestLoggingMiddlewareStatus(t *testing.T) {
	var logs bytes.Buffer
	logg, err := logger.New("info", &logs)
	require.NoError(t, err)
	handler := loggingMiddleware(logg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/events", nil))

	require.Contains(t, logs.String(), "POST /events HTTP/1.1 418")
}

func TestServerGracefulShutdown(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	address := listener.Addr().String()
	_ = listener.Close()

	var logs bytes.Buffer
	logg, err := logger.New("info", &logs)
	require.NoError(t, err)
	server := NewServer(logg, testApplication{}, "127.0.0.1", port)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()

	require.Eventually(t, func() bool {
		conn, dialErr := net.DialTimeout("tcp", address, 20*time.Millisecond)
		if dialErr != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, time.Second, 20*time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop after context cancellation")
	}
}
