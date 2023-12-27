package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"

	"github.com/google/uuid"
	"golang.org/x/exp/slog"
)

type labService struct {
	labRepository entity.LabRepository
}

func NewLabService(repo entity.LabRepository) entity.LabService {
	return &labService{
		labRepository: repo,
	}
}

func (l *labService) GetAllPrivateLabs(typeOfLab string) ([]entity.LabType, error) {

	if typeOfLab != "challengelab" {
		slog.Error("Only challenge labs are allowed via this function", nil)
		return []entity.LabType{}, errors.New("only challenge labs are allowed via this function")
	}

	labs, err := l.GetLabs(typeOfLab)
	if err != nil {
		slog.Error("Not able to get private labs", err)
	}

	return labs, nil
}

func (l *labService) GetPrivateLabs(typeOfLab string, userId string) ([]entity.LabType, error) {
	// Public labs must only be shown to users who own them.
	labs, err := l.GetLabs(typeOfLab)
	if err != nil {
		slog.Error("Not able to get private labs", err)
	}

	var filteredLabs []entity.LabType

	for _, lab := range labs {
		if helper.Contains(lab.Owners, userId) || helper.Contains(lab.Editors, userId) || helper.Contains(lab.Viewers, userId) {
			filteredLabs = append(filteredLabs, lab)
		}
	}

	return filteredLabs, nil
}

func (l *labService) GetPublicLabs(typeOfLab string) ([]entity.LabType, error) {
	// Public labs must only be shown to users who own them.
	return l.GetLabs(typeOfLab)
}

func (l *labService) GetProtectedLab(typeOfLab string, labId string) (entity.LabType, error) {
	labs, err := l.GetProtectedLabs(typeOfLab)
	if err != nil {
		slog.Error("Not able to get protected labs", err)
	}

	for _, lab := range labs {
		if lab.Id == labId {
			return lab, nil
		}
	}
	return entity.LabType{}, errors.New("lab not found")
}

func (l *labService) GetProtectedLabs(typeOfLab string) ([]entity.LabType, error) {
	return l.GetLabs(typeOfLab)
}

func (l *labService) GetLabs(typeOfLab string) ([]entity.LabType, error) {
	labs := []entity.LabType{}

	blobs, err := l.labRepository.ListBlobs(context.TODO(), typeOfLab)
	if err != nil {
		slog.Error("Not able to get list of blobs", err)
		return labs, err
	}

	for _, element := range blobs {
		if element.IsCurrentVersion {
			lab, err := l.labRepository.GetLab(context.TODO(), typeOfLab, element.Name) //element.Name is labId
			if err != nil {
				slog.Error("not able to get blob from given url",
					slog.String("labId", element.Name),
					slog.String("typeOfLab", typeOfLab),
				)
				continue
			}
			AddCategoryToLabIfMissing(&lab)
			slog.Debug("For " + lab.Type + " type, adding lab " + lab.Name + " to list of labs")
			labs = append(labs, lab)
		}
	}

	return labs, nil
}

func (l *labService) UpsertPrivateLab(lab entity.LabType) error {
	ok, err := l.IsUpsertAllowed(lab)
	if err != nil {
		slog.Error("Not able to verify if upsert is allowed or not.", err)
	}

	if !ok {
		slog.Error(lab.UpdatedBy+" is not either owner or editor which is required to edit the private lab.", nil)
		return errors.New("user is not either owner or editor which is required to edit the private lab")
	}
	return l.UpsertLab(lab)
}

func (l *labService) UpsertPublicLab(lab entity.LabType) error {
	ok, err := l.IsUpsertAllowed(lab)
	if err != nil {
		slog.Error("Not able to verify if upsert is allowed or not.", err)
	}

	if !ok {
		slog.Error(lab.UpdatedBy+" is not either owner or editor which is required to edit the public lab.", nil)
		return errors.New("user is not either owner or editor which is required to edit the public lab")
	}
	return l.UpsertLab(lab)
}

func (l *labService) UpsertProtectedLab(lab entity.LabType) error {
	return l.UpsertLab(lab)
}

func (l *labService) UpsertLab(lab entity.LabType) error {

	ok, err := l.ValidateAddingEditorsOrViewers(lab)
	if err != nil {
		slog.Error("Error validating lab", err)
	}

	if !ok {
		slog.Error("User is not an owner and there are changes in either owners, editors or viewers which is no allowed", nil)
		return errors.New("user is not an owner and there are changes in owners, editors, or viewers")
	}

	l.NewLabThings(&lab)
	l.AssignOwnerToOrphanLab(&lab)

	val, err := json.Marshal(lab)
	if err != nil {
		slog.Error("not able to convert object to string", err)
		return err
	}

	if err := l.labRepository.UpsertLab(context.TODO(), lab.Id, string(val), lab.Type); err != nil {
		slog.Error("not able to save lab", err)
		return err
	}

	return nil
}

func (l *labService) DeletePrivateLab(typeOfLab string, labId string, userId string) error {
	ok, err := l.IsDeleteAllowed(typeOfLab, labId, userId)
	if err != nil {
		slog.Error("Not able to verify if delete should be allowed or not", err)
	}

	if !ok {
		slog.Error("Only owner can delete the lab", err)
		return errors.New("only owner can delete the lab")
	}

	return l.DeleteLab(typeOfLab, labId)
}

func (l *labService) DeletePublicLab(typeOfLab string, labId string, userId string) error {
	ok, err := l.IsDeleteAllowed(typeOfLab, labId, userId)
	if err != nil {
		slog.Error("Not able to verify if delete should be allowed or not", err)
	}

	if !ok {
		slog.Error("Only owner can delete the lab", nil)
		return errors.New("only owner can delete the lab")
	}

	return l.DeleteLab(typeOfLab, labId)
}

func (l *labService) DeleteProtectedLab(typeOfLab string, labId string) error {
	return l.DeleteLab(typeOfLab, labId)
}

func (l *labService) DeleteLab(typeOfLab string, labId string) error {
	if err := l.labRepository.DeleteLab(context.TODO(), typeOfLab, labId); err != nil {
		slog.Error("not able to delete lab", err)
		return err
	}
	return nil
}

func (l *labService) GetPrivateLabVersions(typeOfLab string, labId string, userId string) ([]entity.LabType, error) {
	existingLab, err := l.labRepository.GetLab(context.TODO(), typeOfLab, labId)
	if err != nil {
		slog.Error("Not able to get the current version of lab.", err)
		return []entity.LabType{}, err
	}
	if !helper.Contains(existingLab.Owners, userId) && !helper.Contains(existingLab.Editors, userId) && !helper.Contains(existingLab.Viewers, userId) {
		slog.Error("User doesn't have access to view Lab "+existingLab.Name, err)
		return []entity.LabType{}, errors.New("user does not have access to view lab")
	}
	return l.GetLabVersions(typeOfLab, labId)
}

func (l *labService) GetPublicLabVersions(typeOfLab string, labId string) ([]entity.LabType, error) {
	return l.GetLabVersions(typeOfLab, labId)
}

func (l *labService) GetProtectedLabVersions(typeOfLab string, labId string) ([]entity.LabType, error) {
	return l.GetLabVersions(typeOfLab, labId)
}

func (l *labService) GetLabVersions(typeOfLab string, labId string) ([]entity.LabType, error) {
	labs, err := l.labRepository.GetLabWithVersions(context.TODO(), typeOfLab, labId)
	if err != nil {
		slog.Error("Not able to get list of blobs", err)
		return []entity.LabType{}, err
	}

	return labs, nil
}

// Helper functions.

// NewLabThings modifies the given lab.
func (l *labService) NewLabThings(lab *entity.LabType) {
	if lab.Id == "" {
		lab.Id = uuid.NewString()
		lab.Owners = append(lab.Owners, lab.CreatedBy)
	}
}

// Orphan Lab needs owner
func (l *labService) AssignOwnerToOrphanLab(lab *entity.LabType) {
	if len(lab.Owners) == 0 {
		if lab.CreatedBy != "" {
			lab.Owners = append(lab.Owners, lab.CreatedBy)
		}
		if lab.UpdatedBy != "" {
			lab.Owners = append(lab.Owners, lab.UpdatedBy)
		}
		if lab.CreatedBy == "" && lab.UpdatedBy == "" {
			lab.Owners = append(lab.Owners, "ashisverma@microsoft.com")
			lab.Owners = append(lab.Owners, "ericlucier@microsoft.com")
		}
		slog.Debug("Updating Owners: " + strings.Join(lab.Owners, ", "))
	}

	if len(lab.Owners) == 1 && lab.Owners[0] == "" {
		lab.Owners = []string{}

		if lab.CreatedBy != "" {
			lab.Owners = append(lab.Owners, lab.CreatedBy)
		}
		if lab.UpdatedBy != "" {
			lab.Owners = append(lab.Owners, lab.UpdatedBy)
		}
		if lab.CreatedBy == "" && lab.UpdatedBy == "" {
			lab.Owners = append(lab.Owners, "ashisverma@microsoft.com")
			lab.Owners = append(lab.Owners, "ericlucier@microsoft.com")
		}
		slog.Debug("Updating Owners: " + strings.Join(lab.Owners, ", "))
	}
}

func (l *labService) ValidateAddingEditorsOrViewers(lab entity.LabType) (bool, error) {
	if lab.Id == "" {
		slog.Debug("New lab, all good")
		return true, nil // New lab all good.
	}

	existingLab, err := l.labRepository.GetLab(context.TODO(), lab.Type, lab.Id)
	if err != nil {
		slog.Error("Error getting current version of lab "+lab.Name+" : ", err)
		return false, err
	}

	if len(existingLab.Owners) == 0 || (len(existingLab.Owners) == 1 && existingLab.Owners[0] == "") {
		slog.Info("No owners specified. Allowing changes.")
		return true, nil
	}

	if !helper.Contains(existingLab.Owners, lab.UpdatedBy) {
		if !helper.SlicesAreEqual(existingLab.Owners, lab.Owners) || !helper.SlicesAreEqual(existingLab.Editors, lab.Editors) || !helper.SlicesAreEqual(existingLab.Viewers, lab.Viewers) {
			slog.Debug("Existing Lab: ", existingLab)
			slog.Debug("New Lab: ", lab)
			return false, nil
		}
	}

	return true, nil
}

func (l *labService) IsUpsertAllowed(lab entity.LabType) (bool, error) {
	if lab.Id == "" {
		slog.Debug("New lab, all good")
		return true, nil // New lab all good.
	}

	existingLab, err := l.labRepository.GetLab(context.TODO(), lab.Type, lab.Id)
	if err != nil {
		slog.Error("Error getting current version of lab "+lab.Name+" : ", err)
		return false, err
	}

	if len(existingLab.Owners) == 0 || (len(existingLab.Owners) == 1 && existingLab.Owners[0] == "") {
		slog.Info("No owners specified. Allowing changes.")
		return true, nil
	}

	if !helper.Contains(existingLab.Owners, lab.UpdatedBy) && !helper.Contains(existingLab.Editors, lab.UpdatedBy) {
		return false, nil // user not either owner or editor. edit not allowed.
	}

	return true, nil
}

func (l *labService) IsDeleteAllowed(typeOfLab string, labId string, userId string) (bool, error) {
	slog.Debug("Validating if user is owner of lab or not",
		slog.String("typeOfLab", typeOfLab),
		slog.String("labId", labId),
		slog.String("userId", userId),
	)
	existingLab, err := l.labRepository.GetLab(context.TODO(), typeOfLab, labId)
	if err != nil {
		slog.Error("Error getting current version of lab", err)
		return false, err
	}

	if !helper.Contains(existingLab.Owners, userId) {
		return false, nil
	}

	return true, nil
}

func AddCategoryToLabIfMissing(lab *entity.LabType) {
	if lab.Category == "" {
		slog.Debug("Updating Category for " + lab.Name)
		if lab.Type == "readinesslab" || lab.Type == "mockcase" {
			lab.Category = "protected"
		}
		if lab.Type == "privatelab" || lab.Type == "challengelab" {
			lab.Category = "private"
		}
		if lab.Type == "publiclab" {
			lab.Category = "public"
		}
	}
}
