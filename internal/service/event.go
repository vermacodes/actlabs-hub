package service

import (
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
	"actlabs-hub/internal/logger"
	"context"
	"errors"
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
	userID := logger.GetUserID(ctx)
	if userID == "" {
		logger.LogError(ctx, "user_id not found in context for event creation")
		return errors.New("unauthorized: user_id not found in context")
	}

	event.PartitionKey = userID
	event.RowKey = helper.GenerateUUID()
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
