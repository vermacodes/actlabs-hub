package service

import (
	"actlabs-hub/internal/entity"
	"context"
	"log/slog"
)

type eventService struct {
	eventRepository entity.EventRepository
}

func NewEventService(eventRepository entity.EventRepository) entity.EventService {
	return &eventService{
		eventRepository: eventRepository,
	}
}

func (es *eventService) GetEvents(ctx context.Context) ([]entity.Event, error) {
	slog.Debug("getting events")

	events, err := es.eventRepository.GetEvents(ctx)
	if err != nil {
		slog.Debug("failed to get events")
		return nil, err
	}

	return events, nil
}

func (es *eventService) CreateEvent(ctx context.Context, event entity.Event) error {
	slog.Debug("creating event",
		slog.String("type", event.Type),
		slog.String("reason", event.Reason),
		slog.String("message", event.Message),
		slog.String("reporter", event.Reporter),
		slog.String("object", event.Object),
		slog.String("timeStamp", event.TimeStamp),
	)

	if err := es.eventRepository.CreateEvent(ctx, event); err != nil {
		slog.Debug("failed to create event",
			slog.String("type", event.Type),
			slog.String("reason", event.Reason),
			slog.String("message", event.Message),
			slog.String("reporter", event.Reporter),
			slog.String("object", event.Object),
			slog.String("timeStamp", event.TimeStamp),
		)
		return err
	}

	return nil
}
