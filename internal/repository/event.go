package repository

import (
	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/entity"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/google/uuid"
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
			slog.Debug("failed to get next page of events")
			return events, err
		}

		for _, e := range resp.Entities {
			var tableEntity aztables.EDMEntity
			var event entity.Event
			if err := json.Unmarshal(e, &tableEntity); err != nil {
				slog.Debug("failed to unmarshal table entity", slog.String("error", err.Error()))
				return events, err
			}

			propertiesBytes, err := json.Marshal(tableEntity.Properties)
			if err != nil {
				slog.Debug("failed to marshal properties", slog.String("error", err.Error()))
				return events, err
			}

			if err := json.Unmarshal(propertiesBytes, &event); err != nil {
				slog.Debug("failed to unmarshal properties", slog.String("error", err.Error()))
				return events, err
			}

			events = append(events, event)
		}

	}

	return events, nil
}

func (er *eventRepository) CreateEvent(ctx context.Context, event entity.Event) error {
	event.PartitionKey = event.Object
	event.RowKey = uuid.New().String()

	marshalledEvent, err := json.Marshal(event)
	if err != nil {
		slog.Debug("failed to marshal event",
			slog.String("type", event.Type),
			slog.String("reason", event.Reason),
			slog.String("message", event.Message),
			slog.String("reporter", event.Reporter),
			slog.String("object", event.Object),
			slog.String("timeStamp", event.TimeStamp),
		)
		return err
	}

	_, err = er.auth.ActlabsEventsTableClient.AddEntity(ctx, marshalledEvent, nil)
	if err != nil {
		slog.Debug("failed to add event to table",
			slog.String("type", event.Type),
			slog.String("reason", event.Reason),
			slog.String("message", event.Message),
			slog.String("reporter", event.Reporter),
			slog.String("object", event.Object),
			slog.String("timeStamp", event.TimeStamp),
		)

		return err
	}

	slog.Debug("event added to table")

	return nil
}
