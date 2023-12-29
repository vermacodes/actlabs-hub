package service

import (
	"errors"
	"strings"

	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"

	"golang.org/x/exp/slog"
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

func (a *assignmentService) GetAllAssignments() ([]entity.Assignment, error) {
	slog.Info("getting all assignments")

	assignments, err := a.assignmentRepository.GetAllAssignments()
	if err != nil {
		slog.Error("not able to get assignments",
			slog.String("error", err.Error()),
		)
		return assignments, errors.New("not able to get assignments")
	}
	return assignments, nil
}

func (a *assignmentService) GetAssignmentsByLabId(labId string) ([]entity.Assignment, error) {
	slog.Info("getting assignments for lab",
		slog.String("labId", labId),
	)

	assignments, err := a.assignmentRepository.GetAssignmentsByLabId(labId)
	if err != nil {
		slog.Error("not able to get assignments for lab",
			slog.String("labId", labId),
			slog.String("error", err.Error()),
		)
		return assignments, errors.New("not able to get assignments for lab")
	}
	return assignments, nil
}

func (a *assignmentService) GetAssignmentsByUserId(userId string) ([]entity.Assignment, error) {
	slog.Info("getting assignments for user",
		slog.String("userId", userId),
	)

	assignments, err := a.assignmentRepository.GetAssignmentsByUserId(userId)
	if err != nil {
		slog.Error("not able to get assignments for user ",
			slog.String("userId", userId),
			slog.String("error", err.Error()),
		)
		return assignments, errors.New("not able to get assignments for user")
	}
	return assignments, nil
}

func (a *assignmentService) GetAllLabsRedacted() ([]entity.LabType, error) {
	slog.Info("getting all labs redacted")
	readinessLabRedacted := []entity.LabType{}

	labs, err := a.labService.GetProtectedLabs("readinesslab")
	if err != nil {
		slog.Error("not able to get readiness labs",
			slog.String("error", err.Error()),
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

	return readinessLabRedacted, nil
}

func (a *assignmentService) GetAssignedLabsRedactedByUserId(userId string) ([]entity.LabType, error) {
	slog.Info("getting all labs redacted by user",
		slog.String("userId", userId),
	)

	assignedLabs := []entity.LabType{}

	assignments, err := a.GetAssignmentsByUserId(userId)
	if err != nil {
		slog.Error("not able to get assignments for user",
			slog.String("userId", userId),
			slog.String("error", err.Error()),
		)
		return assignedLabs, err
	}

	labs, err := a.labService.GetProtectedLabs("readinesslab")
	if err != nil {
		slog.Error("not able to get readiness labs",
			slog.String("userId", userId),
			slog.String("error", err.Error()),
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

	return assignedLabs, nil
}

func (a *assignmentService) CreateAssignments(userIds []string, labIds []string, createdBy string) error {

	for _, userId := range userIds {

		if !strings.Contains(userId, "@microsoft.com") {
			userId = userId + "@microsoft.com"
		}

		valid, err := a.assignmentRepository.ValidateUser(userId)
		if err != nil {
			slog.Error("not able to validate user id",
				slog.String("userId", userId),
				slog.String("error", err.Error()),
			)
			continue
		}
		if !valid {
			err := errors.New("user id is not valid")
			slog.Error("user id is not valid",
				slog.String("userId", userId),
				slog.String("error", err.Error()),
			)
			continue
		}

		for _, labId := range labIds {
			slog.Info("creating assignment",
				slog.String("userId", userId),
				slog.String("labId", labId),
			)

			assignment := entity.Assignment{
				PartitionKey: userId,
				RowKey:       labId,
				AssignmentId: userId + "+" + labId,
				UserId:       userId,
				LabId:        labId,
				CreatedBy:    createdBy,
				CreatedOn:    helper.GetTodaysDateTimeString(),
				Status:       "assigned",
			}

			if err := a.assignmentRepository.UpsertAssignment(assignment); err != nil {
				slog.Error("not able to create assignment",
					slog.String("userId", userId),
					slog.String("labId", labId),
					slog.String("error", err.Error()),
				)
				return err
			}
		}
	}
	return nil
}

func (a *assignmentService) DeleteAssignments(assignmentIds []string) error {
	slog.Info("deleting assignments",
		slog.String("assignmentIds", strings.Join(assignmentIds, ",")),
	)
	for _, assignmentId := range assignmentIds {
		if err := a.assignmentRepository.DeleteAssignment(assignmentId); err != nil {
			slog.Error("not able to delete assignment",
				slog.String("assignmentId", assignmentId),
				slog.String("error", err.Error()),
			)
			continue
		}
	}
	return nil
}
