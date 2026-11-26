package internalgrpc

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	"github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/app"
	"github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/logger"
	eventpb "github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/server/grpc/pb"
	memorystorage "github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/storage/memory"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func startTestServer(t *testing.T) (eventpb.EventServiceClient, *bytes.Buffer) {
	t.Helper()

	var logs bytes.Buffer
	logg, err := logger.New("info", &logs)
	require.NoError(t, err)
	application := app.New(logg, memorystorage.New())
	server := NewServer(logg, application, "127.0.0.1", 0)

	listener := bufconn.Listen(1024 * 512)
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.startListener(context.Background(), listener)
	}()
	t.Cleanup(func() {
		server.Stop()
		require.NoError(t, <-errCh)
	})

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return eventpb.NewEventServiceClient(conn), &logs
}

func testEvent() *eventpb.Event {
	return &eventpb.Event{
		Id:           "ev1",
		Title:        "Standup",
		StartsAt:     timestamppb.New(time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC)),
		Duration:     durationpb.New(time.Hour),
		Description:  "daily sync",
		UserId:       "u1",
		NotifyBefore: durationpb.New(30 * time.Minute),
	}
}

func TestGRPCEventFlow(t *testing.T) {
	client, logs := startTestServer(t)
	ctx := context.Background()

	created, err := client.CreateEvent(ctx, &eventpb.CreateEventRequest{Event: testEvent()})
	require.NoError(t, err)
	require.Equal(t, "ev1", created.GetEvent().GetId())
	require.Equal(t, "Standup", created.GetEvent().GetTitle())

	listed, err := client.ListEventsForDay(ctx, &eventpb.ListEventsRequest{
		Date: timestamppb.New(time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)
	require.Len(t, listed.GetEvents(), 1)

	week, err := client.ListEventsForWeek(ctx, &eventpb.ListEventsRequest{
		Date: timestamppb.New(time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)
	require.Len(t, week.GetEvents(), 1)

	month, err := client.ListEventsForMonth(ctx, &eventpb.ListEventsRequest{
		Date: timestamppb.New(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)
	require.Len(t, month.GetEvents(), 1)

	otherDay, err := client.ListEventsForDay(ctx, &eventpb.ListEventsRequest{
		Date: timestamppb.New(time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)
	require.Empty(t, otherDay.GetEvents())

	updated, err := client.UpdateEvent(ctx, &eventpb.UpdateEventRequest{
		Id: "ev1",
		Event: &eventpb.Event{
			Title:    "Standup v2",
			StartsAt: timestamppb.New(time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)),
			Duration: durationpb.New(30 * time.Minute),
			UserId:   "u1",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "Standup v2", updated.GetEvent().GetTitle())

	_, err = client.DeleteEvent(ctx, &eventpb.DeleteEventRequest{Id: "ev1"})
	require.NoError(t, err)

	afterDelete, err := client.ListEventsForDay(ctx, &eventpb.ListEventsRequest{
		Date: timestamppb.New(time.Date(2026, 11, 25, 0, 0, 0, 0, time.UTC)),
	})
	require.NoError(t, err)
	require.Empty(t, afterDelete.GetEvents())

	require.Contains(t, logs.String(), "/event.EventService/CreateEvent")
	require.Contains(t, logs.String(), "OK")
}

func TestGRPCErrorCodes(t *testing.T) {
	client, logs := startTestServer(t)
	ctx := context.Background()

	_, err := client.CreateEvent(ctx, &eventpb.CreateEventRequest{Event: testEvent()})
	require.NoError(t, err)

	_, err = client.CreateEvent(ctx, &eventpb.CreateEventRequest{Event: testEvent()})
	require.Equal(t, codes.AlreadyExists, status.Code(err))

	busy := testEvent()
	busy.Id = "ev2"
	busy.StartsAt = timestamppb.New(time.Date(2026, 11, 25, 10, 30, 0, 0, time.UTC))
	_, err = client.CreateEvent(ctx, &eventpb.CreateEventRequest{Event: busy})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))

	_, err = client.DeleteEvent(ctx, &eventpb.DeleteEventRequest{Id: "missing"})
	require.Equal(t, codes.NotFound, status.Code(err))

	_, err = client.UpdateEvent(ctx, &eventpb.UpdateEventRequest{Id: "missing", Event: testEvent()})
	require.Equal(t, codes.NotFound, status.Code(err))

	require.Contains(t, logs.String(), "AlreadyExists")
	require.Contains(t, logs.String(), "NotFound")
}

func TestGRPCValidation(t *testing.T) {
	client, _ := startTestServer(t)
	ctx := context.Background()

	_, err := client.CreateEvent(ctx, &eventpb.CreateEventRequest{})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	noID := testEvent()
	noID.Id = ""
	_, err = client.CreateEvent(ctx, &eventpb.CreateEventRequest{Event: noID})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	noTitle := testEvent()
	noTitle.Title = ""
	_, err = client.CreateEvent(ctx, &eventpb.CreateEventRequest{Event: noTitle})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	negative := testEvent()
	negative.Duration = durationpb.New(-time.Hour)
	_, err = client.CreateEvent(ctx, &eventpb.CreateEventRequest{Event: negative})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = client.CreateEvent(ctx, &eventpb.CreateEventRequest{Event: testEvent()})
	require.NoError(t, err)

	_, err = client.UpdateEvent(ctx, &eventpb.UpdateEventRequest{Event: testEvent()})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = client.DeleteEvent(ctx, &eventpb.DeleteEventRequest{})
	require.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = client.ListEventsForDay(ctx, &eventpb.ListEventsRequest{})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGRPCServerStartStop(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	address := listener.Addr().String()
	require.NoError(t, listener.Close())

	var logs bytes.Buffer
	logg, err := logger.New("info", &logs)
	require.NoError(t, err)
	server := NewServer(logg, app.New(logg, memorystorage.New()), "127.0.0.1", port)

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
		return conn.Close() == nil
	}, time.Second, 20*time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("grpc server did not stop after context cancellation")
	}
	require.Contains(t, logs.String(), "grpc server stopped")
}
