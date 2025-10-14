package repository

import (
	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/logger"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"golang.org/x/net/context"
)

type eventRepository struct {
	auth *auth.Auth
}

func NewEventRepository(auth *auth.Auth) (entity.EventRepository, error) {
	return &eventRepository{
		auth: auth,
	}, nil
}

func (er *eventRepository) GetEvents(ctx context.Context) ([]entity.Event, error) {
	events := []entity.Event{}

	timeStamp := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)

	pager := er.auth.ActlabsEventsTableClient.NewListEntitiesPager(&aztables.ListEntitiesOptions{
		Filter: to.Ptr(fmt.Sprintf("timeStamp ge datetime'%s'", timeStamp)),
	})
	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			logger.LogError(ctx, "failed to get next page of events from table storage",
				"error", err,
			)
			return events, err
		}

		for _, e := range resp.Entities {
			var tableEntity aztables.EDMEntity
			var event entity.Event
			if err := json.Unmarshal(e, &tableEntity); err != nil {
				logger.LogError(ctx, "failed to unmarshal event table entity from storage",
					"error", err,
				)
				return events, err
			}

			propertiesBytes, err := json.Marshal(tableEntity.Properties)
			if err != nil {
				logger.LogError(ctx, "failed to marshal event properties from storage",
					"error", err,
				)
				return events, err
			}

			if err := json.Unmarshal(propertiesBytes, &event); err != nil {
				logger.LogError(ctx, "failed to unmarshal event properties from storage",
					"error", err,
				)
				return events, err
			}

			events = append(events, event)
		}

	}

	return events, nil
}

func (er *eventRepository) CreateEvent(ctx context.Context, event entity.Event) error {
	eventBinary, err := json.Marshal(event)
	if err != nil {
		logger.LogError(ctx, "failed to marshal event for storage",
			"error", err,
		)
		return err
	}

	_, err = er.auth.ActlabsEventsTableClient.AddEntity(ctx, eventBinary, nil)
	if err != nil {
		logger.LogError(ctx, "failed to add event to table storage",
			"error", err,
		)
		return err
	}

	return nil
}
