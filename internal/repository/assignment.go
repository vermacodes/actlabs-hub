package repository

import (
	"context"
	"encoding/json"
	"strings"

	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"

	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

type assignmentRepository struct {
	auth      *auth.Auth
	appConfig *config.Config
	rdb       *redis.Client
}

func NewAssignmentRepository(
	auth *auth.Auth,
	appConfig *config.Config,
	rdb *redis.Client,
) (entity.AssignmentRepository, error) {
	return &assignmentRepository{
		auth:      auth,
		appConfig: appConfig,
		rdb:       rdb,
	}, nil
}

func (a *assignmentRepository) GetAllAssignments() ([]entity.Assignment, error) {
	slog.Debug("getting all assignments")
	assignment := entity.Assignment{}
	assignments := []entity.Assignment{}

	pager := a.auth.ActlabsReadinessTableClient.NewListEntitiesPager(nil)
	for pager.More() {
		response, err := pager.NextPage(context.Background())
		if err != nil {
			slog.Debug("error getting assignments entities",
				slog.String("error", err.Error()),
			)
			return assignments, err
		}

		for _, element := range response.Entities {
			//var myEntity aztables.EDMEntity
			if err := json.Unmarshal(element, &assignment); err != nil {
				slog.Debug("error unmarshal entity",
					slog.String("error", err.Error()),
				)
				return assignments, err
			}
			assignments = append(assignments, assignment)
		}
	}

	return assignments, nil
}

func (a *assignmentRepository) GetAssignmentsByLabId(labId string) ([]entity.Assignment, error) {
	slog.Debug("getting assignments by lab id",
		slog.String("labId", labId),
	)
	assignment := entity.Assignment{}
	assignments := []entity.Assignment{}

	pager := a.auth.ActlabsReadinessTableClient.NewListEntitiesPager(nil)
	for pager.More() {
		response, err := pager.NextPage(context.Background())
		if err != nil {
			slog.Debug("error getting entities",
				slog.String("labId", labId),
				slog.String("error", err.Error()),
			)
			return assignments, err
		}

		for _, element := range response.Entities {
			//var myEntity aztables.EDMEntity
			if err := json.Unmarshal(element, &assignment); err != nil {
				slog.Debug("error unmarshal entity",
					slog.String("labId", labId),
					slog.String("error", err.Error()),
				)
				return assignments, err
			}

			if assignment.LabId == labId {
				assignments = append(assignments, assignment)
			}
		}
	}

	return assignments, nil
}

func (a *assignmentRepository) GetAssignmentsByUserId(userId string) ([]entity.Assignment, error) {
	slog.Debug("getting assignments by user id",
		slog.String("userId", userId),
	)

	assignment := entity.Assignment{}
	assignments := []entity.Assignment{}

	pager := a.auth.ActlabsReadinessTableClient.NewListEntitiesPager(nil)
	for pager.More() {
		response, err := pager.NextPage(context.Background())
		if err != nil {
			slog.Debug("error getting entities",
				slog.String("userId", userId),
				slog.String("error", err.Error()),
			)
			return assignments, err
		}

		for _, element := range response.Entities {
			//var myEntity aztables.EDMEntity
			if err := json.Unmarshal(element, &assignment); err != nil {
				slog.Debug("error unmarshal entity",
					slog.String("userId", userId),
					slog.String("error", err.Error()),
				)
				return assignments, err
			}

			if assignment.UserId == userId {
				assignments = append(assignments, assignment)
			}
		}
	}

	return assignments, nil
}

func (a *assignmentRepository) DeleteAssignment(assignmentId string) error {

	slog.Debug("deleting assignment",
		slog.String("assignmentId", assignmentId),
	)

	userId := assignmentId[:strings.Index(assignmentId, "+")]

	_, err := a.auth.ActlabsReadinessTableClient.DeleteEntity(context.Background(), userId, assignmentId, nil)
	if err != nil {
		slog.Debug("error deleting assignment record: ",
			slog.String("assignmentId", assignmentId),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}

func (a *assignmentRepository) UpsertAssignment(assignment entity.Assignment) error {
	slog.Debug("upserting assignment",
		slog.String("assignmentId", assignment.AssignmentId),
		slog.String("userId", assignment.UserId),
		slog.String("labId", assignment.LabId),
	)

	assignment.PartitionKey = assignment.UserId
	assignment.RowKey = assignment.AssignmentId

	val, err := json.Marshal(assignment)
	if err != nil {
		slog.Error("error marshalling assignment record",
			slog.String("assignmentId", assignment.AssignmentId),
			slog.String("userId", assignment.UserId),
			slog.String("labId", assignment.LabId),
			slog.String("error", err.Error()),
		)
		return err
	}

	_, err = a.auth.ActlabsReadinessTableClient.UpsertEntity(context.TODO(), val, nil)

	if err != nil {
		slog.Error("error creating assignment record: ",
			slog.String("assignmentId", assignment.AssignmentId),
			slog.String("userId", assignment.UserId),
			slog.String("labId", assignment.LabId),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}

func (a *assignmentRepository) ValidateUser(userId string) (bool, error) {
	return true, nil
}
