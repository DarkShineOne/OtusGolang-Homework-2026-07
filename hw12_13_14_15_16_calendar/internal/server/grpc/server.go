// Package internalgrpc exposes the calendar business logic over gRPC.
package internalgrpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	eventpb "github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/server/grpc/pb"
	"github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/storage"
	"google.golang.org/grpc"
)

const stopTimeout = 5 * time.Second

// Logger is the minimal logging dependency of the gRPC server.
type Logger interface {
	Info(msg string)
}

// Application is the business-logic facade used by the gRPC server.
type Application interface {
	CreateEvent(ctx context.Context, event storage.Event) error
	UpdateEvent(ctx context.Context, event storage.Event) error
	DeleteEvent(ctx context.Context, id string) error
	ListEventsForDay(ctx context.Context, day time.Time) ([]storage.Event, error)
	ListEventsForWeek(ctx context.Context, weekStart time.Time) ([]storage.Event, error)
	ListEventsForMonth(ctx context.Context, monthStart time.Time) ([]storage.Event, error)
}

// Server wraps the gRPC server and its lifecycle.
type Server struct {
	logger  Logger
	addr    string
	service *service
	server  *grpc.Server
}

// NewServer builds a gRPC server bound to host:port and wired to the application.
func NewServer(logger Logger, application Application, host string, port int) *Server {
	s := &Server{
		logger:  logger,
		addr:    net.JoinHostPort(host, strconv.Itoa(port)),
		service: &service{app: application},
	}
	s.server = grpc.NewServer(
		grpc.UnaryInterceptor(loggingInterceptor(logger)),
	)
	eventpb.RegisterEventServiceServer(s.server, s.service)
	return s
}

// Start serves until ctx is cancelled or Serve fails.
func (s *Server) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}
	return s.startListener(ctx, listener)
}

func (s *Server) startListener(ctx context.Context, listener net.Listener) error {
	s.logger.Info("grpc server started on " + listener.Addr().String())

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		s.Stop()
		return nil
	case err := <-errCh:
		if err == nil || errors.Is(err, grpc.ErrServerStopped) {
			return nil
		}
		return fmt.Errorf("serve grpc: %w", err)
	}
}

// Stop gracefully stops the server, forcing shutdown after a timeout.
func (s *Server) Stop() {
	stopped := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(stopTimeout):
		s.server.Stop()
	}
	s.logger.Info("grpc server stopped")
}
