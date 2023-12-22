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
	assignment := entity.Assignment{}
	assignments := []entity.Assignment{}

	pager := a.auth.ActlabsReadinessTableClient.NewListEntitiesPager(nil)
	for pager.More() {
		response, err := pager.NextPage(context.Background())
		if err != nil {
			slog.Error("Error getting entities: ", err)
			return assignments, err
		}

		for _, element := range response.Entities {
			//var myEntity aztables.EDMEntity
			if err := json.Unmarshal(element, &assignment); err != nil {
				slog.Error("Error unmarshal entity: ", err)
				return assignments, err
			}
			assignments = append(assignments, assignment)
		}
	}

	return assignments, nil
}

func (a *assignmentRepository) GetAssignmentsByLabId(labId string) ([]entity.Assignment, error) {
	assignment := entity.Assignment{}
	assignments := []entity.Assignment{}

	pager := a.auth.ActlabsReadinessTableClient.NewListEntitiesPager(nil)
	for pager.More() {
		response, err := pager.NextPage(context.Background())
		if err != nil {
			slog.Error("Error getting entities: ", err)
			return assignments, err
		}

		for _, element := range response.Entities {
			//var myEntity aztables.EDMEntity
			if err := json.Unmarshal(element, &assignment); err != nil {
				slog.Error("Error unmarshal entity: ", err)
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
	assignment := entity.Assignment{}
	assignments := []entity.Assignment{}

	pager := a.auth.ActlabsReadinessTableClient.NewListEntitiesPager(nil)
	for pager.More() {
		response, err := pager.NextPage(context.Background())
		if err != nil {
			slog.Error("Error getting entities: ", err)
			return assignments, err
		}

		for _, element := range response.Entities {
			//var myEntity aztables.EDMEntity
			if err := json.Unmarshal(element, &assignment); err != nil {
				slog.Error("Error unmarshal entity: ", err)
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

	slog.Debug("Deleting assignment: ", assignmentId)

	userId := assignmentId[:strings.Index(assignmentId, "+")]

	_, err := a.auth.ActlabsReadinessTableClient.DeleteEntity(context.Background(), userId, assignmentId, nil)
	if err != nil {
		slog.Error("Error deleting assignment record: ", err)
		return err
	}
	slog.Debug("Assignment record deleted successfully")
	return nil
}

func (a *assignmentRepository) UpsertAssignment(assignment entity.Assignment) error {
	assignment.PartitionKey = assignment.UserId
	assignment.RowKey = assignment.AssignmentId

	val, err := json.Marshal(assignment)
	if err != nil {
		slog.Error("Error marshalling assignment record: ", err)
		return err
	}

	slog.Debug("Assignment record: ", string(val))

	_, err = a.auth.ActlabsReadinessTableClient.UpsertEntity(context.TODO(), val, nil)

	if err != nil {
		slog.Error("Error creating assignment record: ", err)
		return err
	}

	slog.Debug("Assignment record created successfully")

	return nil
}

func (a *assignmentRepository) ValidateUser(userId string) (bool, error) {
	return true, nil
}
