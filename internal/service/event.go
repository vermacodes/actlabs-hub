package service

import (
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/logger"
	"context"
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
	events, err := es.eventRepository.GetEvents(ctx)
	if err != nil {
		logger.LogError(ctx, "failed to get events from repository",
			"error", err,
		)
		return nil, err
	}

	return events, nil
}

func (es *eventService) CreateEvent(ctx context.Context, event entity.Event) error {
	if err := es.eventRepository.CreateEvent(ctx, event); err != nil {
		logger.LogError(ctx, "failed to create event in repository",
			"event_type", event.Type,
			"event_reason", event.Reason,
			"event_object", event.Object,
			"error", err,
		)
		return err
	}

	return nil
}
