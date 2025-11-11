package repository

import (
	"context"
	"encoding/json"
	"strings"

	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/logger"

	"github.com/redis/go-redis/v9"
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

func (a *assignmentRepository) GetAllAssignments(ctx context.Context) ([]entity.Assignment, error) {
	assignment := entity.Assignment{}
	assignments := []entity.Assignment{}

	pager := a.auth.ActlabsReadinessTableClient.NewListEntitiesPager(nil)
	for pager.More() {
		response, err := pager.NextPage(ctx)
		if err != nil {
			logger.LogError(ctx, "Table storage query failed for all assignments",
				"operation", "get_all_assignments",
				"table", "actlabs_readiness",
				"error_type", "database",
				"error", err.Error(),
			)
			return assignments, err
		}

		for _, element := range response.Entities {
			//var myEntity aztables.EDMEntity
			if err := json.Unmarshal(element, &assignment); err != nil {
				logger.LogError(ctx, "JSON unmarshal failed for assignment entity",
					"operation", "get_all_assignments",
					"table", "actlabs_readiness",
					"error_type", "serialization",
					"error", err.Error(),
				)
				return assignments, err
			}
			assignments = append(assignments, assignment)
		}
	}

	return assignments, nil
}

func (a *assignmentRepository) GetAssignmentsByLabId(ctx context.Context, labId string) ([]entity.Assignment, error) {
	assignment := entity.Assignment{}
	assignments := []entity.Assignment{}

	pager := a.auth.ActlabsReadinessTableClient.NewListEntitiesPager(nil)
	for pager.More() {
		response, err := pager.NextPage(ctx)
		if err != nil {
			logger.LogError(ctx, "Table storage query failed for assignments by lab ID",
				"operation", "get_assignments_by_lab_id",
				"table", "actlabs_readiness",
				"lab_id", labId,
				"error_type", "database",
				"error", err.Error(),
			)
			return assignments, err
		}

		for _, element := range response.Entities {
			//var myEntity aztables.EDMEntity
			if err := json.Unmarshal(element, &assignment); err != nil {
				logger.LogError(ctx, "JSON unmarshal failed for assignment entity",
					"operation", "get_assignments_by_lab_id",
					"table", "actlabs_readiness",
					"lab_id", labId,
					"error_type", "serialization",
					"error", err.Error(),
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

func (a *assignmentRepository) GetAssignmentsByUserId(ctx context.Context, userId string) ([]entity.Assignment, error) {
	assignment := entity.Assignment{}
	assignments := []entity.Assignment{}

	pager := a.auth.ActlabsReadinessTableClient.NewListEntitiesPager(nil)
	for pager.More() {
		response, err := pager.NextPage(ctx)
		if err != nil {
			logger.LogError(ctx, "Table storage query failed for assignments by user ID",
				"operation", "get_assignments_by_user_id",
				"table", "actlabs_readiness",
				"user_id", userId,
				"error_type", "database",
				"error", err.Error(),
			)
			return assignments, err
		}

		for _, element := range response.Entities {
			//var myEntity aztables.EDMEntity
			if err := json.Unmarshal(element, &assignment); err != nil {
				logger.LogError(ctx, "JSON unmarshal failed for assignment entity",
					"operation", "get_assignments_by_user_id",
					"table", "actlabs_readiness",
					"user_id", userId,
					"error_type", "serialization",
					"error", err.Error(),
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

func (a *assignmentRepository) DeleteAssignment(ctx context.Context, assignmentId string) error {
	userId := assignmentId[:strings.Index(assignmentId, "+")]

	_, err := a.auth.ActlabsReadinessTableClient.DeleteEntity(ctx, userId, assignmentId, nil)
	if err != nil {
		logger.LogError(ctx, "Table storage delete operation failed",
			"operation", "delete_assignment",
			"table", "actlabs_readiness",
			"assignment_id", assignmentId,
			"user_id", userId,
			"error_type", "database",
			"error", err.Error(),
		)
		return err
	}

	return nil
}

func (a *assignmentRepository) UpsertAssignment(ctx context.Context, assignment entity.Assignment) error {
	assignment.PartitionKey = assignment.UserId
	assignment.RowKey = assignment.AssignmentId

	val, err := json.Marshal(assignment)
	if err != nil {
		logger.LogError(ctx, "JSON marshal failed for assignment entity",
			"operation", "upsert_assignment",
			"table", "actlabs_readiness",
			"assignment_id", assignment.AssignmentId,
			"user_id", assignment.UserId,
			"lab_id", assignment.LabId,
			"error_type", "serialization",
			"error", err.Error(),
		)
		return err
	}

	_, err = a.auth.ActlabsReadinessTableClient.UpsertEntity(ctx, val, nil)

	if err != nil {
		logger.LogError(ctx, "Table storage upsert operation failed",
			"operation", "upsert_assignment",
			"table", "actlabs_readiness",
			"assignment_id", assignment.AssignmentId,
			"user_id", assignment.UserId,
			"lab_id", assignment.LabId,
			"error_type", "database",
			"error", err.Error(),
		)
		return err
	}

	return nil
}

func (a *assignmentRepository) ValidateUser(ctx context.Context, userId string) (bool, error) {
	return true, nil
}
