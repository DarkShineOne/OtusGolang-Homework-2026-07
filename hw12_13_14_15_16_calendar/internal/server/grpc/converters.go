package internalgrpc

import (
	"errors"

	eventpb "github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/server/grpc/pb"
	"github.com/darkshineone/otusgolang-homework/hw12_13_14_15_calendar/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func eventFromProto(e *eventpb.Event) (storage.Event, error) {
	if e == nil {
		return storage.Event{}, errors.New("event is required")
	}
	startsAt := e.GetStartsAt()
	if startsAt == nil || !startsAt.IsValid() {
		return storage.Event{}, errors.New("event starts_at is required")
	}
	duration := e.GetDuration()
	if duration == nil || !duration.IsValid() {
		return storage.Event{}, errors.New("event duration is required")
	}
	notifyBefore := e.GetNotifyBefore()
	if notifyBefore != nil && !notifyBefore.IsValid() {
		return storage.Event{}, errors.New("event notify_before is invalid")
	}

	event := storage.Event{
		ID:          e.GetId(),
		Title:       e.GetTitle(),
		StartsAt:    startsAt.AsTime().UTC(),
		Duration:    duration.AsDuration(),
		Description: e.GetDescription(),
		UserID:      e.GetUserId(),
	}
	if notifyBefore != nil {
		event.NotifyBefore = notifyBefore.AsDuration()
	}
	return event, event.Validate()
}

func eventToProto(e storage.Event) *eventpb.Event {
	return &eventpb.Event{
		Id:           e.ID,
		Title:        e.Title,
		StartsAt:     timestamppb.New(e.StartsAt),
		Duration:     durationpb.New(e.Duration),
		Description:  e.Description,
		UserId:       e.UserID,
		NotifyBefore: durationpb.New(e.NotifyBefore),
	}
}

func statusFromError(err error) error {
	switch {
	case errors.Is(err, storage.ErrEventNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, storage.ErrEventExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, storage.ErrDateBusy):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
