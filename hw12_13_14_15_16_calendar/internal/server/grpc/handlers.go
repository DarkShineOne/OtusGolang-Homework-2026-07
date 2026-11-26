package internalgrpc

import (
	"context"
	"time"

	eventpb "github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/server/grpc/pb"
	"github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type service struct {
	eventpb.UnimplementedEventServiceServer
	app Application
}

func (s *service) CreateEvent(
	ctx context.Context,
	req *eventpb.CreateEventRequest,
) (*eventpb.CreateEventResponse, error) {
	event, err := eventFromProto(req.GetEvent())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if event.ID == "" {
		return nil, status.Error(codes.InvalidArgument, "event id is required")
	}
	if err := s.app.CreateEvent(ctx, event); err != nil {
		return nil, statusFromError(err)
	}
	return &eventpb.CreateEventResponse{Event: eventToProto(event)}, nil
}

func (s *service) UpdateEvent(
	ctx context.Context,
	req *eventpb.UpdateEventRequest,
) (*eventpb.UpdateEventResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "event id is required")
	}
	event, err := eventFromProto(req.GetEvent())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	event.ID = req.GetId()
	if err := s.app.UpdateEvent(ctx, event); err != nil {
		return nil, statusFromError(err)
	}
	return &eventpb.UpdateEventResponse{Event: eventToProto(event)}, nil
}

func (s *service) DeleteEvent(
	ctx context.Context,
	req *eventpb.DeleteEventRequest,
) (*eventpb.DeleteEventResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "event id is required")
	}
	if err := s.app.DeleteEvent(ctx, req.GetId()); err != nil {
		return nil, statusFromError(err)
	}
	return &eventpb.DeleteEventResponse{}, nil
}

func (s *service) ListEventsForDay(
	ctx context.Context,
	req *eventpb.ListEventsRequest,
) (*eventpb.ListEventsResponse, error) {
	return s.listEvents(ctx, req, s.app.ListEventsForDay)
}

func (s *service) ListEventsForWeek(
	ctx context.Context,
	req *eventpb.ListEventsRequest,
) (*eventpb.ListEventsResponse, error) {
	return s.listEvents(ctx, req, s.app.ListEventsForWeek)
}

func (s *service) ListEventsForMonth(
	ctx context.Context,
	req *eventpb.ListEventsRequest,
) (*eventpb.ListEventsResponse, error) {
	return s.listEvents(ctx, req, s.app.ListEventsForMonth)
}

type listFunc func(context.Context, time.Time) ([]storage.Event, error)

func (s *service) listEvents(
	ctx context.Context,
	req *eventpb.ListEventsRequest,
	list listFunc,
) (*eventpb.ListEventsResponse, error) {
	date := req.GetDate()
	if date == nil || !date.IsValid() {
		return nil, status.Error(codes.InvalidArgument, "date is required")
	}
	events, err := list(ctx, date.AsTime().UTC())
	if err != nil {
		return nil, statusFromError(err)
	}

	response := &eventpb.ListEventsResponse{Events: make([]*eventpb.Event, 0, len(events))}
	for i := range events {
		response.Events = append(response.Events, eventToProto(events[i]))
	}
	return response, nil
}
