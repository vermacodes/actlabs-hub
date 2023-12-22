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
	assignments, err := a.assignmentRepository.GetAllAssignments()
	if err != nil {
		slog.Error("not able to get assignments", err)
		return assignments, err
	}
	return assignments, nil
}

func (a *assignmentService) GetAssignmentsByLabId(labId string) ([]entity.Assignment, error) {
	assignments, err := a.assignmentRepository.GetAssignmentsByLabId(labId)
	if err != nil {
		slog.Error("not able to get assignments for lab "+labId, err)
		return assignments, err
	}
	return assignments, nil
}

func (a *assignmentService) GetAssignmentsByUserId(userId string) ([]entity.Assignment, error) {
	assignments, err := a.assignmentRepository.GetAssignmentsByUserId(userId)
	if err != nil {
		slog.Error("not able to get assignments for user "+userId, err)
		return assignments, err
	}
	return assignments, nil
}

func (a *assignmentService) GetAllLabsRedacted() ([]entity.LabType, error) {
	readinessLabRedacted := []entity.LabType{}

	labs, err := a.labService.GetProtectedLabs("readinesslab")
	if err != nil {
		slog.Error("not able to get readiness labs", err)
		return readinessLabRedacted, err
	}

	for _, lab := range labs {
		slog.Debug("Lab ID : " + lab.Name)
		lab.ExtendScript = "redacted"
		lab.Description = "<p>" + lab.Name + "</p>"
		lab.Type = "assignment"
		lab.Tags = []string{"assignment"}
		readinessLabRedacted = append(readinessLabRedacted, lab)
	}

	return readinessLabRedacted, nil
}

func (a *assignmentService) GetAssignedLabsRedactedByUserId(userId string) ([]entity.LabType, error) {
	assignedLabs := []entity.LabType{}

	assignments, err := a.GetAssignmentsByUserId(userId)
	if err != nil {
		slog.Error("not able to get assignments for user"+userId, err)
		return assignedLabs, err
	}

	labs, err := a.labService.GetProtectedLabs("readinesslab")
	if err != nil {
		slog.Error("not able to get readiness labs", err)
		return assignedLabs, err
	}

	for _, assignment := range assignments {
		slog.Debug("Lab ID : " + assignment.LabId)
		for _, lab := range labs {
			slog.Debug("Checking Lab Name : " + lab.Name)
			if assignment.LabId == lab.Id {
				slog.Debug("Assignment ID : " + assignment.AssignmentId + " is for lab " + lab.Name)
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
			slog.Error("not able to validate user id"+userId, err)
			continue
		}
		if !valid {
			err := errors.New("user id is not valid")
			slog.Error("user id is not valid"+userId, err)
			continue
		}

		for _, labId := range labIds {

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
				slog.Error("not able to create assignment", err)
				return err
			}

			slog.Debug("Assigned lab " + labId + " to user " + userId)
		}
	}
	return nil
}

func (a *assignmentService) DeleteAssignments(assignmentIds []string) error {
	for _, assignmentId := range assignmentIds {
		if err := a.assignmentRepository.DeleteAssignment(assignmentId); err != nil {
			slog.Error("not able to delete assignment with id "+assignmentId, err)
			continue
		}
	}
	return nil
}
