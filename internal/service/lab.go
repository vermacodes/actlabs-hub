package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"strings"

	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
	"actlabs-hub/internal/logger"

	"github.com/google/uuid"
)

type labService struct {
	labRepository entity.LabRepository
}

func NewLabService(repo entity.LabRepository) entity.LabService {
	return &labService{
		labRepository: repo,
	}
}

func (l *labService) GetAllPrivateLabs(ctx context.Context, typeOfLab string) ([]entity.LabType, error) {
	if typeOfLab != "challengelab" {
		logger.LogError(ctx, "only challenge labs are allowed via this function", "typeOfLab", typeOfLab)
		return []entity.LabType{}, errors.New("only challenge labs are allowed via this function")
	}

	labs, err := l.GetLabs(ctx, typeOfLab)
	if err != nil {
		return []entity.LabType{}, err
	}

	return labs, nil
}

func (l *labService) GetPrivateLabs(ctx context.Context, typeOfLab string, userId string) ([]entity.LabType, error) {
	labs, err := l.GetLabs(ctx, typeOfLab)
	if err != nil {
		return labs, err
	}

	var filteredLabs []entity.LabType

	for _, lab := range labs {
		if helper.Contains(lab.Owners, userId) || helper.Contains(lab.Editors, userId) || helper.Contains(lab.Viewers, userId) {
			filteredLabs = append(filteredLabs, lab)
		}
	}

	return filteredLabs, nil
}

func (l *labService) GetPrivateLab(ctx context.Context, typeOfLab string, labId string) (entity.LabType, error) {
	lab, err := l.labRepository.GetLab(ctx, typeOfLab, labId)
	if err != nil {
		logger.LogError(ctx, "not able to get lab", "labId", labId, "typeOfLab", typeOfLab, "error", err.Error())
		return entity.LabType{}, fmt.Errorf("not able to get lab")
	}

	return lab, nil
}

func (l *labService) GetPublicLabs(ctx context.Context, typeOfLab string) ([]entity.LabType, error) {
	// Public labs must only be shown to users who own them.
	return l.GetLabs(ctx, typeOfLab)
}

func (l *labService) GetProtectedLab(ctx context.Context, typeOfLab string, labId string, userId string, requestIsWithSecret bool) (entity.LabType, error) {
	labs, err := l.GetProtectedLabs(ctx, typeOfLab, userId, requestIsWithSecret)
	if err != nil {
		return entity.LabType{}, err
	}

	for _, lab := range labs {
		if lab.Id == labId {
			return lab, nil
		}
	}
	return entity.LabType{}, errors.New("lab not found")
}

func (l *labService) GetProtectedLabs(ctx context.Context, typeOfLab string, userId string, requestIsWithSecret bool) ([]entity.LabType, error) {

	// If RbacEnforcedProtectedLab is enabled then we need to redact description, supporting document and script.
	labs, err := l.GetLabs(ctx, typeOfLab)
	if err != nil {
		return []entity.LabType{}, err
	}

	if requestIsWithSecret {
		return labs, nil
	}

	for i := range labs {

		if labs[i].RbacEnforcedProtectedLab && !helper.Contains(labs[i].Owners, userId) && !helper.Contains(labs[i].Editors, userId) && !helper.Contains(labs[i].Viewers, userId) {
			labs[i].Description = base64.StdEncoding.EncodeToString([]byte("Access to this lab is restricted by its owners. Please contact them for access or further information."))
			labs[i].SupportingDocumentId = ""
			labs[i].ExtendScript = ""
		}
	}

	return labs, nil
}

func (l *labService) GetLabs(ctx context.Context, typeOfLab string) ([]entity.LabType, error) {
	logger.LogInfo(ctx, "getting labs", "typeOfLab", typeOfLab)

	labs := []entity.LabType{}

	blobs, err := l.labRepository.ListBlobs(ctx, typeOfLab)
	if err != nil {
		logger.LogError(ctx, "not able to get list of blobs", "typeOfLab", typeOfLab, "error", err.Error())
		return labs, err
	}

	logger.LogDebug(ctx, "listed labs", "type", typeOfLab, "count", len(blobs))

	for _, element := range blobs {
		if element.IsCurrentVersion {
			lab, err := l.labRepository.GetLab(ctx, typeOfLab, element.Name) //element.Name is labId
			if err != nil {
				logger.LogError(ctx, "not able to get blob from given url", "labId", element.Name, "typeOfLab", typeOfLab, "error", err.Error())
				continue
			}
			AddCategoryToLabIfMissing(ctx, l, &lab)
			labs = append(labs, lab)
		}
	}

	logger.LogDebug(ctx, "current versions of labs", "type", typeOfLab, "count", len(labs))

	return labs, nil
}

func (l *labService) UpsertPrivateLab(ctx context.Context, lab entity.LabType) (entity.LabType, error) {
	ok, err := l.IsUpsertAllowed(ctx, lab)
	if err != nil {
		return lab, err
	}

	if !ok {
		return lab, fmt.Errorf("user is not either owner or editor which is required to edit the private lab")
	}

	return l.UpsertLab(ctx, lab)
}

func (l *labService) UpsertPublicLab(ctx context.Context, lab entity.LabType) (entity.LabType, error) {
	ok, err := l.IsUpsertAllowed(ctx, lab)
	if err != nil {
		return lab, err
	}

	if !ok {
		return lab, fmt.Errorf("user is not either owner or editor which is required to edit the public lab")
	}
	return l.UpsertLab(ctx, lab)
}

func (l *labService) UpsertProtectedLab(ctx context.Context, lab entity.LabType, userId string) (entity.LabType, error) {
	// if lab Id is empty and there are no owners, let it go.
	if lab.Id == "" && len(lab.Owners) == 0 {
		return l.UpsertLab(ctx, lab)
	}

	if lab.RbacEnforcedProtectedLab && !helper.Contains(lab.Owners, userId) && !helper.Contains(lab.Editors, userId) {
		return lab, errors.New("only the owner or an editor can modify this RBAC-enforced lab, please contact the owner or editor for access or further details")
	}
	return l.UpsertLab(ctx, lab)
}

func (l *labService) UpsertLab(ctx context.Context, lab entity.LabType) (entity.LabType, error) {

	ok, err := l.ValidateAddingEditorsOrViewers(ctx, lab)
	if err != nil {
		return lab, &json.MarshalerError{}
	}

	if !ok {
		return lab, errors.New("user is not an owner and there are changes in owners, editors, or viewers")
	}

	l.NewLabThings(ctx, &lab)
	l.AssignOwnerToOrphanLab(ctx, &lab)

	// Ensure supporting document exists if its part of the lab.
	if lab.SupportingDocumentId != "" {
		if !l.DoesSupportingDocumentExist(ctx, lab.SupportingDocumentId) {
			logger.LogError(ctx, "Supporting document doesn't exist", "SupportingDocumentId", lab.SupportingDocumentId)
			return lab, errors.New("supporting document doesn't exist")
		}
	}

	val, err := json.Marshal(lab)
	if err != nil {
		logger.LogError(ctx, "not able to convert object to string", "labName", lab.Name, "labId", lab.Id, "typeOfLab", lab.Type, "owners", strings.Join(lab.Owners, ", "), "editors", strings.Join(lab.Editors, ", "), "error", err.Error())
		return lab, fmt.Errorf("not able to convert object to string")
	}

	if err := l.labRepository.UpsertLab(ctx, lab.Id, string(val), lab.Type); err != nil {
		logger.LogError(ctx, "not able to save lab", "labName", lab.Name, "labId", lab.Id, "typeOfLab", lab.Type, "owners", strings.Join(lab.Owners, ", "), "editors", strings.Join(lab.Editors, ", "), "error", err.Error())
		return lab, fmt.Errorf("not able to save lab")
	}

	return lab, nil
}

func (l *labService) DeletePrivateLab(ctx context.Context, typeOfLab string, labId string, userId string) error {
	ok, err := l.IsDeleteAllowed(ctx, typeOfLab, labId, userId)
	if err != nil {
		return err
	}

	if !ok {
		return errors.New("only owner can delete the lab")
	}

	if err := l.DeleteLab(ctx, typeOfLab, labId); err != nil {
		logger.LogError(ctx, "not able to delete lab", "labId", labId, "typeOfLab", typeOfLab, "error", err.Error())
		return fmt.Errorf("not able to delete lab")
	}

	return nil
}

func (l *labService) DeletePublicLab(ctx context.Context, typeOfLab string, labId string, userId string) error {
	ok, err := l.IsDeleteAllowed(ctx, typeOfLab, labId, userId)
	if err != nil {
		logger.LogError(ctx, "Not able to verify if delete should be allowed or not", "error", err.Error())
	}

	if !ok {
		logger.LogError(ctx, "Only owner can delete the lab", "labId", labId)
		return errors.New("only owner can delete the lab")
	}

	return l.DeleteLab(ctx, typeOfLab, labId)
}

func (l *labService) DeleteProtectedLab(ctx context.Context, typeOfLab string, labId string) error {
	return l.DeleteLab(ctx, typeOfLab, labId)
}

func (l *labService) DeleteLab(ctx context.Context, typeOfLab string, labId string) error {
	// Delete supporting document if any.
	lab, err := l.labRepository.GetLab(ctx, typeOfLab, labId)
	if err != nil {
		logger.LogError(ctx, "not able to get lab", "labId", labId, "error", err.Error())
		return err
	}

	if lab.SupportingDocumentId != "" {
		if l.DoesSupportingDocumentExist(ctx, lab.SupportingDocumentId) {
			if err := l.DeleteSupportingDocument(ctx, lab.SupportingDocumentId); err != nil {
				logger.LogError(ctx, "not able to delete supporting document", "error", err.Error())
				return err
			}
		}
	}

	if err := l.labRepository.DeleteLab(ctx, typeOfLab, labId); err != nil {
		logger.LogError(ctx, "not able to delete lab", "error", err.Error())
		return err
	}
	return nil
}

func (l *labService) GetPrivateLabVersions(ctx context.Context, typeOfLab string, labId string, userId string) ([]entity.LabType, error) {
	existingLab, err := l.labRepository.GetLab(ctx, typeOfLab, labId)
	if err != nil {
		logger.LogError(ctx, "Not able to get the current version of lab.", "error", err.Error())
		return []entity.LabType{}, err
	}
	if !helper.Contains(existingLab.Owners, userId) && !helper.Contains(existingLab.Editors, userId) && !helper.Contains(existingLab.Viewers, userId) {
		logger.LogError(ctx, "User doesn't have access to view Lab", "labName", existingLab.Name)
		return []entity.LabType{}, errors.New("user does not have access to view lab")
	}
	return l.GetLabVersions(ctx, typeOfLab, labId)
}

func (l *labService) GetPublicLabVersions(ctx context.Context, typeOfLab string, labId string) ([]entity.LabType, error) {
	return l.GetLabVersions(ctx, typeOfLab, labId)
}

func (l *labService) GetProtectedLabVersions(ctx context.Context, typeOfLab string, labId string) ([]entity.LabType, error) {
	return l.GetLabVersions(ctx, typeOfLab, labId)
}

func (l *labService) GetLabVersions(ctx context.Context, typeOfLab string, labId string) ([]entity.LabType, error) {
	labs, err := l.labRepository.GetLabWithVersions(ctx, typeOfLab, labId)
	if err != nil {
		logger.LogError(ctx, "Not able to get list of blobs", "error", err.Error())
		return []entity.LabType{}, err
	}

	return labs, nil
}

// Supporting Documents
func (l *labService) UpsertSupportingDocument(ctx context.Context, supportingDocument multipart.File) (string, error) {
	supportingDocumentId, err := l.labRepository.UpsertSupportingDocument(ctx, supportingDocument)
	if err != nil {
		logger.LogError(ctx, "not able to save supporting document", "error", err.Error())
		return "", err
	}

	return supportingDocumentId, nil
}

func (l *labService) DeleteSupportingDocument(ctx context.Context, supportingDocumentId string) error {

	if !l.DoesSupportingDocumentExist(ctx, supportingDocumentId) {
		logger.LogError(ctx, "Supporting document doesn't exist", "SupportingDocumentId", supportingDocumentId)

		return nil
	}

	if err := l.labRepository.DeleteSupportingDocument(ctx, supportingDocumentId); err != nil {
		logger.LogError(ctx, "not able to delete supporting document", "error", err.Error())
		return err
	}

	return nil
}

func (l *labService) GetSupportingDocument(ctx context.Context, supportingDocumentId string) (io.ReadCloser, error) {
	supportingDocument, err := l.labRepository.GetSupportingDocument(ctx, supportingDocumentId)
	if err != nil {
		logger.LogError(ctx, "not able to get supporting document", "error", err.Error())
		return nil, err
	}

	return supportingDocument, nil
}

func (l *labService) DoesSupportingDocumentExist(ctx context.Context, supportingDocumentId string) bool {
	exists := l.labRepository.DoesSupportingDocumentExist(ctx, supportingDocumentId)
	logger.LogDebug(ctx, "Supporting document exists", "supportingDocumentId", supportingDocumentId, "exists", exists)
	return exists
}

// Helper functions.

// NewLabThings modifies the given lab.
func (l *labService) NewLabThings(ctx context.Context, lab *entity.LabType) {
	if lab.Id == "" {
		lab.Id = uuid.NewString()
		lab.Owners = append(lab.Owners, lab.CreatedBy)
	}
}

// Orphan Lab needs owner
func (l *labService) AssignOwnerToOrphanLab(ctx context.Context, lab *entity.LabType) {
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
		logger.LogDebug(ctx, "Updating Owners", "owners", strings.Join(lab.Owners, ", "))
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
		logger.LogDebug(ctx, "Updating Owners", "owners", strings.Join(lab.Owners, ", "))
	}
}

func (l *labService) ValidateAddingEditorsOrViewers(ctx context.Context, lab entity.LabType) (bool, error) {
	if lab.Id == "" {
		logger.LogDebug(ctx, "new lab, all good", "labName", lab.Name, "owners", strings.Join(lab.Owners, ", "))
		return true, nil // New lab all good.
	}

	existingLab, err := l.labRepository.GetLab(ctx, lab.Type, lab.Id)
	if err != nil {
		logger.LogError(ctx, "error getting current version of lab", "labName", lab.Name, "labId", lab.Id, "typeOfLab", lab.Type, "owners", strings.Join(lab.Owners, ", "), "editors", strings.Join(lab.Editors, ", "), "error", err.Error())
		return false, fmt.Errorf("error getting current version of lab")
	}

	if len(existingLab.Owners) == 0 || (len(existingLab.Owners) == 1 && existingLab.Owners[0] == "") {
		logger.LogDebug(ctx, "no owners specified allowing changes", "labName", lab.Name, "labId", lab.Id, "typeOfLab", lab.Type, "owners", strings.Join(lab.Owners, ", "), "editors", strings.Join(lab.Editors, ", "))
		return true, nil
	}

	if !helper.Contains(existingLab.Owners, lab.UpdatedBy) {
		if !helper.SlicesAreEqual(existingLab.Owners, lab.Owners) || !helper.SlicesAreEqual(existingLab.Editors, lab.Editors) || !helper.SlicesAreEqual(existingLab.Viewers, lab.Viewers) {
			logger.LogError(ctx, "user is not owner and there are changes in either owners, editors or viewers which is not allowed", "labName", lab.Name, "labId", lab.Id, "typeOfLab", lab.Type, "updatedBy", lab.UpdatedBy, "owners", strings.Join(lab.Owners, ", "), "editors", strings.Join(lab.Editors, ", "), "viewers", strings.Join(lab.Viewers, ", "))
			return false, fmt.Errorf("user is not owner and there are changes in owners, editors, or viewers")
		}
	}

	return true, nil
}

func (l *labService) IsUpsertAllowed(ctx context.Context, lab entity.LabType) (bool, error) {
	if lab.Id == "" {
		logger.LogDebug(ctx, "New lab, all good", "labName", lab.Name, "owners", strings.Join(lab.Owners, ", "))
		return true, nil // New lab all good.
	}

	// if lab is not new, we must find it in db.
	existingLab, err := l.labRepository.GetLab(ctx, lab.Type, lab.Id)
	if err != nil {
		logger.LogError(ctx, "error getting current version of lab", "labName", lab.Name, "labId", lab.Id, "typeOfLab", lab.Type, "owners", strings.Join(lab.Owners, ", "), "error", err.Error())
		return false, err
	}

	if len(existingLab.Owners) == 0 || (len(existingLab.Owners) == 1 && existingLab.Owners[0] == "") {
		logger.LogDebug(ctx, "no owners specified. allowing changes", "labName", lab.Name, "labId", lab.Id, "typeOfLab", lab.Type, "owners", strings.Join(lab.Owners, ", "))
		return true, nil
	}

	if !helper.Contains(existingLab.Owners, lab.UpdatedBy) && !helper.Contains(existingLab.Editors, lab.UpdatedBy) {
		logger.LogError(ctx, "user is not either owner or editor which is required to edit the lab", "labName", lab.Name, "labId", lab.Id, "typeOfLab", lab.Type, "owners", strings.Join(lab.Owners, ", "), "editors", strings.Join(lab.Editors, ", "))
		return false, fmt.Errorf("user is not owner or editor") // user not either owner or editor. edit not allowed.
	}

	return true, nil
}

func (l *labService) IsDeleteAllowed(ctx context.Context, typeOfLab string, labId string, userId string) (bool, error) {
	existingLab, err := l.labRepository.GetLab(ctx, typeOfLab, labId)
	if err != nil {
		logger.LogError(ctx, "error getting current version of lab", "labId", labId, "typeOfLab", typeOfLab, "error", err.Error())
		return false, fmt.Errorf("error getting current version of lab")
	}

	if !helper.Contains(existingLab.Owners, userId) {
		return false, nil
	}

	return true, nil
}

func AddCategoryToLabIfMissing(ctx context.Context, l *labService, lab *entity.LabType) {
	if lab.Category == "" {
		logger.LogDebug(ctx, "Updating Category for lab", "labName", lab.Name)
		if lab.Type == "readinesslab" || lab.Type == "mockcase" {
			lab.Category = "protected"
		}
		if lab.Type == "privatelab" || lab.Type == "challengelab" {
			lab.Category = "private"
		}
		if lab.Type == "publiclab" {
			lab.Category = "public"
		}

		// Update lab so that this change is saved.
		l.UpsertLab(ctx, *lab)
	}
}
