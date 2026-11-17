package internalhttp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/storage"
)

type Server struct {
	logger Logger
	app    Application
	server *http.Server
}

type Logger interface {
	Info(msg string)
}

type Application interface {
	CreateEvent(ctx context.Context, event storage.Event) error
	UpdateEvent(ctx context.Context, event storage.Event) error
	DeleteEvent(ctx context.Context, id string) error
	ListEventsForDay(ctx context.Context, day time.Time) ([]storage.Event, error)
	ListEventsForWeek(ctx context.Context, weekStart time.Time) ([]storage.Event, error)
	ListEventsForMonth(ctx context.Context, monthStart time.Time) ([]storage.Event, error)
}

func NewServer(logger Logger, application Application, host string, port int) *Server {
	mux := http.NewServeMux()
	mux.Handle("/hello", loggingMiddleware(logger, http.HandlerFunc(hello)))

	return &Server{
		logger: logger,
		app:    application,
		server: &http.Server{
			Addr:              net.JoinHostPort(host, strconv.Itoa(port)),
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

func (s *Server) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return fmt.Errorf("listen http: %w", err)
	}
	s.logger.Info("http server started on " + listener.Addr().String())

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.Stop(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve http: %w", err)
	}
}

func (s *Server) Stop(ctx context.Context) error {
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}
	s.logger.Info("http server stopped")

	return nil
}

func hello(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("Hello, world!\n"))
}
