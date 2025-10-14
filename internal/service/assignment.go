package service

import (
	"context"
	"errors"
	"strings"

	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
	"actlabs-hub/internal/logger"
)

type assignmentService struct {
	assignmentRepository entity.AssignmentRepository
	labService           entity.LabService
}

func NewAssignmentService(assignmentRepository entity.AssignmentRepository, labService entity.LabService) entity.AssignmentService {
	return &assignmentService{
		assignmentRepository: assignmentRepository,
		labService:           labService,
	}
}

func (a *assignmentService) GetAllAssignments(ctx context.Context) ([]entity.Assignment, error) {
	logger.LogInfo(ctx, "Starting get all assignments operation",
		"operation", "get_all_assignments",
	)

	assignments, err := a.assignmentRepository.GetAllAssignments(ctx)
	if err != nil {
		logger.LogError(ctx, "Failed to get assignments from repository",
			"operation", "get_all_assignments",
			"error", err,
		)
		return assignments, errors.New("not able to get assignments")
	}

	logger.LogInfo(ctx, "Successfully retrieved all assignments",
		"operation", "get_all_assignments",
		"count", len(assignments),
	)
	return assignments, nil
}

func (a *assignmentService) GetAssignmentsByLabId(ctx context.Context, labId string) ([]entity.Assignment, error) {
	logger.LogInfo(ctx, "Starting get assignments by lab ID operation",
		"operation", "get_assignments_by_lab_id",
		"lab_id", labId,
	)

	assignments, err := a.assignmentRepository.GetAssignmentsByLabId(ctx, labId)
	if err != nil {
		logger.LogError(ctx, "Failed to get assignments by lab ID from repository",
			"operation", "get_assignments_by_lab_id",
			"lab_id", labId,
			"error", err,
		)
		return assignments, errors.New("not able to get assignments for lab")
	}

	logger.LogInfo(ctx, "Successfully retrieved assignments by lab ID",
		"operation", "get_assignments_by_lab_id",
		"lab_id", labId,
		"count", len(assignments),
	)
	return assignments, nil
}

func (a *assignmentService) GetAssignmentsByUserId(ctx context.Context, userId string) ([]entity.Assignment, error) {
	logger.LogInfo(ctx, "Starting get assignments by user ID operation",
		"operation", "get_assignments_by_user_id",
		"user_id", userId,
	)

	assignments, err := a.assignmentRepository.GetAssignmentsByUserId(ctx, userId)
	if err != nil {
		logger.LogError(ctx, "Failed to get assignments by user ID from repository",
			"operation", "get_assignments_by_user_id",
			"user_id", userId,
			"error", err,
		)
		return assignments, errors.New("not able to get assignments for user")
	}

	// remove deleted assignments
	assignments = RemoveDeletedAssignments(assignments)

	logger.LogInfo(ctx, "Successfully retrieved assignments by user ID",
		"operation", "get_assignments_by_user_id",
		"user_id", userId,
		"count", len(assignments),
	)

	return assignments, nil
}

func (a *assignmentService) GetAllLabsRedacted(ctx context.Context, userId string) ([]entity.LabType, error) {
	logger.LogInfo(ctx, "Starting get all labs redacted operation",
		"operation", "get_all_labs_redacted",
		"user_id", userId,
	)
	readinessLabRedacted := []entity.LabType{}

	labs, err := a.labService.GetProtectedLabs(ctx, "readinesslab", userId, false)
	if err != nil {
		logger.LogError(ctx, "Failed to get readiness labs from lab service",
			"operation", "get_all_labs_redacted",
			"user_id", userId,
			"error", err,
		)
		return readinessLabRedacted, err
	}

	for _, lab := range labs {
		lab.ExtendScript = "redacted"
		lab.Description = "<p>" + lab.Name + "</p>" // keep in p tags for UI to render correctly
		lab.Type = "assignment"
		lab.Tags = []string{"assignment"}
		readinessLabRedacted = append(readinessLabRedacted, lab)
	}

	logger.LogInfo(ctx, "Successfully retrieved and redacted labs",
		"operation", "get_all_labs_redacted",
		"user_id", userId,
		"count", len(readinessLabRedacted),
	)

	return readinessLabRedacted, nil
}

func (a *assignmentService) GetAssignedLabsRedactedByUserId(ctx context.Context, userId string) ([]entity.LabType, error) {
	logger.LogInfo(ctx, "Starting get assigned labs redacted by user ID operation",
		"operation", "get_assigned_labs_redacted_by_user_id",
		"user_id", userId,
	)

	assignedLabs := []entity.LabType{}

	assignments, err := a.GetAssignmentsByUserId(ctx, userId)
	if err != nil {
		logger.LogError(ctx, "Failed to get assignments for user",
			"operation", "get_assigned_labs_redacted_by_user_id",
			"user_id", userId,
			"error", err,
		)
		return assignedLabs, err
	}

	labs, err := a.labService.GetProtectedLabs(ctx, "readinesslab", userId, false)
	if err != nil {
		logger.LogError(ctx, "Failed to get readiness labs from lab service",
			"operation", "get_assigned_labs_redacted_by_user_id",
			"user_id", userId,
			"error", err,
		)
		return assignedLabs, err
	}

	for _, assignment := range assignments {
		for _, lab := range labs {
			if assignment.LabId == lab.Id {
				if assignment.UserId == userId {
					lab.ExtendScript = "redacted"
					lab.Description = lab.Message // Replace description with message fro redacted labs
					lab.Type = "assignment"
					lab.Tags = []string{"assignment"}
					assignedLabs = append(assignedLabs, lab)
					break
				}
			}
		}
	}

	logger.LogInfo(ctx, "Successfully retrieved assigned labs redacted by user ID",
		"operation", "get_assigned_labs_redacted_by_user_id",
		"user_id", userId,
		"assigned_labs_count", len(assignedLabs),
		"total_assignments", len(assignments),
	)

	return assignedLabs, nil
}

func (a *assignmentService) CreateAssignments(ctx context.Context, userIds []string, labIds []string, createdBy string) error {
	logger.LogInfo(ctx, "Starting create assignments operation",
		"operation", "create_assignments",
		"user_count", len(userIds),
		"lab_count", len(labIds),
		"created_by", createdBy,
	)

	for _, userId := range userIds {

		if !strings.Contains(userId, "@microsoft.com") {
			userId = userId + "@microsoft.com"
		}

		valid, err := a.assignmentRepository.ValidateUser(ctx, userId)
		if err != nil {
			logger.LogError(ctx, "Failed to validate user ID",
				"operation", "create_assignments",
				"user_id", userId,
				"error", err,
			)
			continue
		}
		if !valid {
			logger.LogError(ctx, "User ID validation failed",
				"operation", "create_assignments",
				"user_id", userId,
				"validation_rule", "user_exists",
			)
			continue
		}

		for _, labId := range labIds {
			logger.LogInfo(ctx, "Creating individual assignment",
				"operation", "create_assignments",
				"user_id", userId,
				"lab_id", labId,
			)

			assignment := entity.Assignment{
				PartitionKey: userId,
				RowKey:       labId,
				AssignmentId: userId + "+" + labId,
				UserId:       userId,
				LabId:        labId,
				CreatedBy:    createdBy,
				CreatedAt:    helper.GetTodaysDateTimeString(),
				Status:       entity.AssignmentStatusCreated,
			}

			if err := a.assignmentRepository.UpsertAssignment(ctx, assignment); err != nil {
				logger.LogError(ctx, "Failed to create assignment in repository",
					"operation", "create_assignments",
					"user_id", userId,
					"lab_id", labId,
					"error", err,
				)
				return err
			}
		}
	}

	logger.LogInfo(ctx, "Successfully created assignments",
		"operation", "create_assignments",
		"user_count", len(userIds),
		"lab_count", len(labIds),
		"created_by", createdBy,
	)
	return nil
}

func (a *assignmentService) UpdateAssignment(ctx context.Context, userId string, labId string, status string) error {
	logger.LogInfo(ctx, "Starting update assignment operation",
		"operation", "update_assignment",
		"user_id", userId,
		"lab_id", labId,
		"new_status", status,
	)

	assignment, err := getAssignmentByUserIdAndLabId(ctx, userId, labId, a.assignmentRepository)
	if err != nil {
		logger.LogError(ctx, "Failed to get assignment for update",
			"operation", "update_assignment",
			"user_id", userId,
			"lab_id", labId,
			"error", err,
		)
		return err
	}

	if status == "InProgress" {
		assignment.StartedAt = helper.GetTodaysDateTimeString()
	} else if status == "Completed" {
		assignment.CompletedAt = helper.GetTodaysDateTimeString()
	} else if status == "Deleted" {
		assignment.DeletedAt = helper.GetTodaysDateTimeString()
	} else {
		logger.LogError(ctx, "Invalid assignment status provided",
			"operation", "update_assignment",
			"user_id", userId,
			"lab_id", labId,
			"invalid_status", status,
			"valid_statuses", "InProgress,Completed,Deleted",
		)
		return errors.New("invalid status")
	}

	assignment.Status = status

	if err := a.assignmentRepository.UpsertAssignment(ctx, assignment); err != nil {
		logger.LogError(ctx, "Failed to update assignment in repository",
			"operation", "update_assignment",
			"user_id", userId,
			"lab_id", labId,
			"status", status,
			"error", err,
		)
		return err
	}

	logger.LogInfo(ctx, "Successfully updated assignment",
		"operation", "update_assignment",
		"user_id", userId,
		"lab_id", labId,
		"new_status", status,
	)

	return nil
}

// func (a *assignmentService) DeleteAssignments(assignmentIds []string) error {
// 	slog.Info("deleting assignments",
// 		slog.String("assignmentIds", strings.Join(assignmentIds, ",")),
// 	)
// 	for _, assignmentId := range assignmentIds {
// 		if err := a.assignmentRepository.DeleteAssignment(assignmentId); err != nil {
// 			slog.Error("not able to delete assignment",
// 				slog.String("assignmentId", assignmentId),
// 				slog.String("error", err),
// 			)
// 			continue
// 		}
// 	}
// 	return nil
// }

func (a *assignmentService) DeleteAssignments(ctx context.Context, assignmentIds []string, userPrincipal string) error {
	logger.LogInfo(ctx, "Starting delete assignments operation",
		"operation", "delete_assignments",
		"assignment_count", len(assignmentIds),
		"deleted_by", userPrincipal,
	)
	for _, assignmentId := range assignmentIds {

		assignment, err := getAssignmentByUserIdAndLabId(ctx, assignmentId[:strings.Index(assignmentId, "+")], assignmentId[strings.Index(assignmentId, "+")+1:], a.assignmentRepository)
		if err != nil {
			logger.LogError(ctx, "Failed to get assignment for deletion",
				"operation", "delete_assignments",
				"assignment_id", assignmentId,
				"error", err,
			)
			continue
		}

		assignment.DeletedAt = helper.GetTodaysDateTimeString()
		assignment.DeletedBy = userPrincipal
		assignment.Status = entity.AssignmentStatusDeleted

		if err := a.assignmentRepository.UpsertAssignment(ctx, assignment); err != nil {
			logger.LogError(ctx, "Failed to mark assignment as deleted in repository",
				"operation", "delete_assignments",
				"assignment_id", assignmentId,
				"error", err,
			)
			continue
		}

		logger.LogInfo(ctx, "Successfully marked assignment as deleted",
			"operation", "delete_assignments",
			"assignment_id", assignmentId,
			"deleted_by", userPrincipal,
		)
	}

	logger.LogInfo(ctx, "Completed delete assignments operation",
		"operation", "delete_assignments",
		"assignment_count", len(assignmentIds),
		"deleted_by", userPrincipal,
	)
	return nil
}

func getAssignmentByUserIdAndLabId(ctx context.Context, userId string, labId string, assignmentRepository entity.AssignmentRepository) (entity.Assignment, error) {
	assignments, err := assignmentRepository.GetAssignmentsByUserId(ctx, userId)
	if err != nil {
		logger.LogError(ctx, "Failed to get assignments for user from repository",
			"operation", "get_assignment_by_user_and_lab",
			"user_id", userId,
			"lab_id", labId,
			"error", err,
		)
		return entity.Assignment{}, err
	}

	var assignment entity.Assignment
	for _, a := range assignments {
		if a.LabId == labId {
			assignment = a
			break
		}
	}

	if assignment.UserId == "" {
		logger.LogError(ctx, "Assignment not found for user and lab combination",
			"operation", "get_assignment_by_user_and_lab",
			"user_id", userId,
			"lab_id", labId,
		)
		return entity.Assignment{}, errors.New("not able to find assignment")
	}

	return assignment, nil
}

func RemoveDeletedAssignments(assignments []entity.Assignment) []entity.Assignment {
	updatedAssignments := []entity.Assignment{}
	for _, assignment := range assignments {
		if assignment.Status != entity.AssignmentStatusDeleted {
			updatedAssignments = append(updatedAssignments, assignment)
		}
	}
	return updatedAssignments
}
