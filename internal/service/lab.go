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
		slog.Error("only challenge labs are allowed via this function",
			slog.String("typeOfLab", typeOfLab),
		)
		return []entity.LabType{}, errors.New("only challenge labs are allowed via this function")
	}

	labs, err := l.GetLabs(typeOfLab)
	if err != nil {
		return []entity.LabType{}, err
	}

	return labs, nil
}

func (l *labService) GetPrivateLabs(typeOfLab string, userId string) ([]entity.LabType, error) {
	labs, err := l.GetLabs(typeOfLab)
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

func (l *labService) GetPrivateLab(typeOfLab string, labId string) (entity.LabType, error) {
	lab, err := l.labRepository.GetLab(context.TODO(), typeOfLab, labId)
	if err != nil {
		slog.Error("not able to get lab",
			slog.String("labId", labId),
			slog.String("typeOfLab", typeOfLab),
			slog.String("error", err.Error()),
		)

		return entity.LabType{}, fmt.Errorf("not able to get lab")
	}

	return lab, nil
}

func (l *labService) GetPublicLabs(typeOfLab string) ([]entity.LabType, error) {
	// Public labs must only be shown to users who own them.
	return l.GetLabs(typeOfLab)
}

func (l *labService) GetProtectedLab(typeOfLab string, labId string, userId string, requestIsWithSecret bool) (entity.LabType, error) {
	labs, err := l.GetProtectedLabs(typeOfLab, userId, requestIsWithSecret)
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

func (l *labService) GetProtectedLabs(typeOfLab string, userId string, requestIsWithSecret bool) ([]entity.LabType, error) {

	// If RbacEnforcedProtectedLab is enabled then we need to redact description, supporting document and script.
	labs, err := l.GetLabs(typeOfLab)
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

func (l *labService) GetLabs(typeOfLab string) ([]entity.LabType, error) {
	slog.Info("getting labs",
		slog.String("typeOfLab", typeOfLab),
	)

	labs := []entity.LabType{}

	blobs, err := l.labRepository.ListBlobs(context.TODO(), typeOfLab)
	if err != nil {
		slog.Error("not able to get list of blobs",
			slog.String("typeOfLab", typeOfLab),
			slog.String("error", err.Error()),
		)
		return labs, err
	}

	for _, element := range blobs {
		if element.IsCurrentVersion {
			lab, err := l.labRepository.GetLab(context.TODO(), typeOfLab, element.Name) //element.Name is labId
			if err != nil {
				slog.Error("not able to get blob from given url",
					slog.String("labId", element.Name),
					slog.String("typeOfLab", typeOfLab),
					slog.String("error", err.Error()),
				)
				continue
			}
			AddCategoryToLabIfMissing(l, &lab)
			labs = append(labs, lab)
		}
	}

	return labs, nil
}

func (l *labService) UpsertPrivateLab(lab entity.LabType) (entity.LabType, error) {
	ok, err := l.IsUpsertAllowed(lab)
	if err != nil {
		return lab, err
	}

	if !ok {
		return lab, fmt.Errorf("user is not either owner or editor which is required to edit the private lab")
	}

	return l.UpsertLab(lab)
}

func (l *labService) UpsertPublicLab(lab entity.LabType) (entity.LabType, error) {
	ok, err := l.IsUpsertAllowed(lab)
	if err != nil {
		return lab, err
	}

	if !ok {
		return lab, fmt.Errorf("user is not either owner or editor which is required to edit the public lab")
	}
	return l.UpsertLab(lab)
}

func (l *labService) UpsertProtectedLab(lab entity.LabType, userId string) (entity.LabType, error) {
	// if lab Id is empty and there are no owners, let it go.
	if lab.Id == "" && len(lab.Owners) == 0 {
		return l.UpsertLab(lab)
	}

	if lab.RbacEnforcedProtectedLab && !helper.Contains(lab.Owners, userId) && !helper.Contains(lab.Editors, userId) {
		return lab, errors.New("only the owner or an editor can modify this RBAC-enforced lab, please contact the owner or editor for access or further details")
	}
	return l.UpsertLab(lab)
}

func (l *labService) UpsertLab(lab entity.LabType) (entity.LabType, error) {

	ok, err := l.ValidateAddingEditorsOrViewers(lab)
	if err != nil {
		return lab, &json.MarshalerError{}
	}

	if !ok {
		return lab, errors.New("user is not an owner and there are changes in owners, editors, or viewers")
	}

	l.NewLabThings(&lab)
	l.AssignOwnerToOrphanLab(&lab)

	// Ensure supporting document exists if its part of the lab.
	if lab.SupportingDocumentId != "" {
		if !l.DoesSupportingDocumentExist(context.TODO(), lab.SupportingDocumentId) {
			slog.Error("Supporting document doesn't exist",
				slog.String("SupportingDocumentId", lab.SupportingDocumentId),
			)
			return lab, errors.New("supporting document doesn't exist")
		}
	}

	val, err := json.Marshal(lab)
	if err != nil {
		slog.Error("not able to convert object to string",
			slog.String("labName", lab.Name),
			slog.String("labId", lab.Id),
			slog.String("typeOfLab", lab.Type),
			slog.String("owners", strings.Join(lab.Owners, ", ")),
			slog.String("editors", strings.Join(lab.Editors, ", ")),
			slog.String("error", err.Error()),
		)
		return lab, fmt.Errorf("not able to convert object to string")
	}

	if err := l.labRepository.UpsertLab(context.TODO(), lab.Id, string(val), lab.Type); err != nil {
		slog.Error("not able to save lab",
			slog.String("labName", lab.Name),
			slog.String("labId", lab.Id),
			slog.String("typeOfLab", lab.Type),
			slog.String("owners", strings.Join(lab.Owners, ", ")),
			slog.String("editors", strings.Join(lab.Editors, ", ")),
			slog.String("error", err.Error()),
		)
		return lab, fmt.Errorf("not able to save lab")
	}

	return lab, nil
}

func (l *labService) DeletePrivateLab(typeOfLab string, labId string, userId string) error {
	ok, err := l.IsDeleteAllowed(typeOfLab, labId, userId)
	if err != nil {
		return err
	}

	if !ok {
		return errors.New("only owner can delete the lab")
	}

	if err := l.DeleteLab(typeOfLab, labId); err != nil {
		slog.Error("not able to delete lab",
			slog.String("userId", userId),
			slog.String("labId", labId),
			slog.String("typeOfLab", typeOfLab),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("not able to delete lab")
	}

	return nil
}

func (l *labService) DeletePublicLab(typeOfLab string, labId string, userId string) error {
	ok, err := l.IsDeleteAllowed(typeOfLab, labId, userId)
	if err != nil {
		slog.Error("Not able to verify if delete should be allowed or not",
			slog.String("error", err.Error()),
		)
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
	// Delete supporting document if any.
	lab, err := l.labRepository.GetLab(context.TODO(), typeOfLab, labId)
	if err != nil {
		slog.Error("not able to get lab",
			slog.String("labId", labId),
		)
		return err
	}

	if lab.SupportingDocumentId != "" {
		if l.DoesSupportingDocumentExist(context.TODO(), lab.SupportingDocumentId) {
			if err := l.DeleteSupportingDocument(context.TODO(), lab.SupportingDocumentId); err != nil {
				slog.Error("not able to delete supporting document",
					slog.String("error", err.Error()),
				)
				return err
			}
		}
	}

	if err := l.labRepository.DeleteLab(context.TODO(), typeOfLab, labId); err != nil {
		slog.Error("not able to delete lab",
			slog.String("error", err.Error()),
		)
		return err
	}
	return nil
}

func (l *labService) GetPrivateLabVersions(typeOfLab string, labId string, userId string) ([]entity.LabType, error) {
	existingLab, err := l.labRepository.GetLab(context.TODO(), typeOfLab, labId)
	if err != nil {
		slog.Error("Not able to get the current version of lab.",
			slog.String("error", err.Error()),
		)
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
		slog.Error("Not able to get list of blobs",
			slog.String("error", err.Error()),
		)
		return []entity.LabType{}, err
	}

	return labs, nil
}

// Supporting Documents
func (l *labService) UpsertSupportingDocument(ctx context.Context, supportingDocument multipart.File) (string, error) {
	supportingDocumentId, err := l.labRepository.UpsertSupportingDocument(ctx, supportingDocument)
	if err != nil {
		slog.Error("not able to save supporting document",
			slog.String("error", err.Error()),
		)
		return "", err
	}

	return supportingDocumentId, nil
}

func (l *labService) DeleteSupportingDocument(ctx context.Context, supportingDocumentId string) error {

	if !l.DoesSupportingDocumentExist(ctx, supportingDocumentId) {
		slog.Error("Supporting document doesn't exist",
			slog.String("SupportingDocumentId", supportingDocumentId),
		)

		return nil
	}

	if err := l.labRepository.DeleteSupportingDocument(ctx, supportingDocumentId); err != nil {
		slog.Error("not able to delete supporting document",
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}

func (l *labService) GetSupportingDocument(ctx context.Context, supportingDocumentId string) (io.ReadCloser, error) {
	supportingDocument, err := l.labRepository.GetSupportingDocument(ctx, supportingDocumentId)
	if err != nil {
		slog.Error("not able to get supporting document",
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	return supportingDocument, nil
}

func (l *labService) DoesSupportingDocumentExist(ctx context.Context, supportingDocumentId string) bool {
	exists := l.labRepository.DoesSupportingDocumentExist(ctx, supportingDocumentId)
	slog.Debug("Supporting document exists: " + supportingDocumentId + " " + fmt.Sprint(exists))
	return exists
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
		slog.Debug("new lab, all good",
			slog.String("labName", lab.Name),
			slog.String("owners", strings.Join(lab.Owners, ", ")),
		)
		return true, nil // New lab all good.
	}

	existingLab, err := l.labRepository.GetLab(context.TODO(), lab.Type, lab.Id)
	if err != nil {
		slog.Error("error getting current version of lab",
			slog.String("labName", lab.Name),
			slog.String("labId", lab.Id),
			slog.String("typeOfLab", lab.Type),
			slog.String("owners", strings.Join(lab.Owners, ", ")),
			slog.String("editors", strings.Join(lab.Editors, ", ")),
			slog.String("error", err.Error()),
		)
		return false, fmt.Errorf("error getting current version of lab")
	}

	if len(existingLab.Owners) == 0 || (len(existingLab.Owners) == 1 && existingLab.Owners[0] == "") {
		slog.Debug("no owners specified allowing changes",
			slog.String("labName", lab.Name),
			slog.String("labId", lab.Id),
			slog.String("typeOfLab", lab.Type),
			slog.String("owners", strings.Join(lab.Owners, ", ")),
			slog.String("editors", strings.Join(lab.Editors, ", ")),
		)
		return true, nil
	}

	if !helper.Contains(existingLab.Owners, lab.UpdatedBy) {
		if !helper.SlicesAreEqual(existingLab.Owners, lab.Owners) || !helper.SlicesAreEqual(existingLab.Editors, lab.Editors) || !helper.SlicesAreEqual(existingLab.Viewers, lab.Viewers) {
			slog.Error("user is not owner and there are changes in either owners, editors or viewers which is not allowed",
				slog.String("labName", lab.Name),
				slog.String("labId", lab.Id),
				slog.String("typeOfLab", lab.Type),
				slog.String("updatedBy", lab.UpdatedBy),
				slog.String("owners", strings.Join(lab.Owners, ", ")),
				slog.String("editors", strings.Join(lab.Editors, ", ")),
				slog.String("viewers", strings.Join(lab.Viewers, ", ")),
			)
			return false, fmt.Errorf("user is not owner and there are changes in owners, editors, or viewers")
		}
	}

	return true, nil
}

func (l *labService) IsUpsertAllowed(lab entity.LabType) (bool, error) {
	if lab.Id == "" {
		slog.Debug("New lab, all good",
			slog.String("labName", lab.Name),
			slog.String("owners", strings.Join(lab.Owners, ", ")),
		)
		return true, nil // New lab all good.
	}

	// if lab is not new, we must find it in db.
	existingLab, err := l.labRepository.GetLab(context.TODO(), lab.Type, lab.Id)
	if err != nil {
		slog.Error("error getting current version of lab",
			slog.String("labName", lab.Name),
			slog.String("labId", lab.Id),
			slog.String("typeOfLab", lab.Type),
			slog.String("owners", strings.Join(lab.Owners, ", ")),
			slog.String("error", err.Error()),
		)
		return false, err
	}

	if len(existingLab.Owners) == 0 || (len(existingLab.Owners) == 1 && existingLab.Owners[0] == "") {
		slog.Debug("no owners specified. allowing changes",
			slog.String("labName", lab.Name),
			slog.String("labId", lab.Id),
			slog.String("typeOfLab", lab.Type),
			slog.String("owners", strings.Join(lab.Owners, ", ")),
		)
		return true, nil
	}

	if !helper.Contains(existingLab.Owners, lab.UpdatedBy) && !helper.Contains(existingLab.Editors, lab.UpdatedBy) {
		slog.Error("user is not either owner or editor which is required to edit the lab",
			slog.String("labName", lab.Name),
			slog.String("labId", lab.Id),
			slog.String("typeOfLab", lab.Type),
			slog.String("owners", strings.Join(lab.Owners, ", ")),
			slog.String("editors", strings.Join(lab.Editors, ", ")),
		)
		return false, fmt.Errorf("user is not owner or editor") // user not either owner or editor. edit not allowed.
	}

	return true, nil
}

func (l *labService) IsDeleteAllowed(typeOfLab string, labId string, userId string) (bool, error) {
	existingLab, err := l.labRepository.GetLab(context.TODO(), typeOfLab, labId)
	if err != nil {
		slog.Error("error getting current version of lab",
			slog.String("useId", userId),
			slog.String("labId", labId),
			slog.String("typeOfLab", typeOfLab),
			slog.String("error", err.Error()),
		)
		return false, fmt.Errorf("error getting current version of lab")
	}

	if !helper.Contains(existingLab.Owners, userId) {
		return false, nil
	}

	return true, nil
}

func AddCategoryToLabIfMissing(l *labService, lab *entity.LabType) {
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

		// Update lab so that this change is saved.
		l.UpsertLab(*lab)
	}
}
