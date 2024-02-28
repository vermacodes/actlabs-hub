package entity

import "context"

type Event struct {
	PartitionKey string `json:"PartitionKey"`
	RowKey       string `json:"RowKey"`
	Type         string `json:"type"`
	Reason       string `json:"reason"`
	Message      string `json:"message"`
	Reporter     string `json:"reporter"`
	Object       string `json:"object"`
	TimeStamp    string `json:"timeStamp"`
}

type EventService interface {
	GetEvents(ctx context.Context) ([]Event, error)
	CreateEvent(ctx context.Context, event Event) error
}

type EventRepository interface {
	GetEvents(ctx context.Context) ([]Event, error)
	CreateEvent(ctx context.Context, event Event) error
}
