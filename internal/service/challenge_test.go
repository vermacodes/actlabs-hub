package service

import (
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/logger"
	"actlabs/labentity"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"testing"
)

func TestIsDeleteAllowed(t *testing.T) {
	tests := []struct {
		name          string
		callingUserId string
		challenge     entity.Challenge
		lab           entity.LabType
		wantErr       bool
	}{
		{
			name:          "owner can always delete",
			callingUserId: "owner@microsoft.com",
			challenge: entity.Challenge{
				UserId:    "user@microsoft.com",
				CreatedBy: "creator@microsoft.com",
			},
			lab: entity.LabType{
				Owners: []string{"owner@microsoft.com"},
			},
			wantErr: false,
		},
		{
			name:          "editor can always delete",
			callingUserId: "editor@microsoft.com",
			challenge: entity.Challenge{
				UserId:    "user@microsoft.com",
				CreatedBy: "creator@microsoft.com",
			},
			lab: entity.LabType{
				Editors: []string{"editor@microsoft.com"},
			},
			wantErr: false,
		},
		{
			name:          "challenger can delete when allowed",
			callingUserId: "creator@microsoft.com",
			challenge: entity.Challenge{
				UserId:    "user@microsoft.com",
				CreatedBy: "creator@microsoft.com",
			},
			lab: entity.LabType{
				LabControls: labentity.LabControlsType{
					ChallengeLabAllowChallengerToDeleteChallenge: true,
				},
			},
			wantErr: false,
		},
		{
			name:          "challenger cannot delete when disallowed",
			callingUserId: "creator@microsoft.com",
			challenge: entity.Challenge{
				UserId:    "user@microsoft.com",
				CreatedBy: "creator@microsoft.com",
			},
			lab: entity.LabType{
				LabControls: labentity.LabControlsType{
					ChallengeLabAllowChallengerToDeleteChallenge: false,
				},
			},
			wantErr: true,
		},
		{
			name:          "challenged user can delete when allowed",
			callingUserId: "user@microsoft.com",
			challenge: entity.Challenge{
				UserId:    "user@microsoft.com",
				CreatedBy: "creator@microsoft.com",
			},
			lab: entity.LabType{
				LabControls: labentity.LabControlsType{
					ChallengeLabAllowUserToDeleteChallenge: true,
				},
			},
			wantErr: false,
		},
		{
			name:          "challenged user cannot delete when disallowed",
			callingUserId: "user@microsoft.com",
			challenge: entity.Challenge{
				UserId:    "user@microsoft.com",
				CreatedBy: "creator@microsoft.com",
			},
			lab: entity.LabType{
				LabControls: labentity.LabControlsType{
					ChallengeLabAllowUserToDeleteChallenge: false,
				},
			},
			wantErr: true,
		},
		{
			name:          "unrelated user cannot delete",
			callingUserId: "random@microsoft.com",
			challenge: entity.Challenge{
				UserId:    "user@microsoft.com",
				CreatedBy: "creator@microsoft.com",
			},
			lab: entity.LabType{
				Owners:  []string{"owner@microsoft.com"},
				Editors: []string{"editor@microsoft.com"},
				LabControls: labentity.LabControlsType{
					ChallengeLabAllowChallengerToDeleteChallenge: true,
					ChallengeLabAllowUserToDeleteChallenge:       true,
				},
			},
			wantErr: true,
		},
		{
			name:          "owner takes priority even when lab controls are false",
			callingUserId: "owner@microsoft.com",
			challenge: entity.Challenge{
				UserId:    "user@microsoft.com",
				CreatedBy: "creator@microsoft.com",
			},
			lab: entity.LabType{
				Owners: []string{"owner@microsoft.com"},
				LabControls: labentity.LabControlsType{
					ChallengeLabAllowChallengerToDeleteChallenge: false,
					ChallengeLabAllowUserToDeleteChallenge:       false,
				},
			},
			wantErr: false,
		},
		{
			name:          "user who is both challenger and challenged can delete when challenger flag is set",
			callingUserId: "self@microsoft.com",
			challenge: entity.Challenge{
				UserId:    "self@microsoft.com",
				CreatedBy: "self@microsoft.com",
			},
			lab: entity.LabType{
				LabControls: labentity.LabControlsType{
					ChallengeLabAllowChallengerToDeleteChallenge: true,
					ChallengeLabAllowUserToDeleteChallenge:       false,
				},
			},
			wantErr: false,
		},
		{
			name:          "user who is both challenger and challenged can delete when user flag is set",
			callingUserId: "self@microsoft.com",
			challenge: entity.Challenge{
				UserId:    "self@microsoft.com",
				CreatedBy: "self@microsoft.com",
			},
			lab: entity.LabType{
				LabControls: labentity.LabControlsType{
					ChallengeLabAllowChallengerToDeleteChallenge: false,
					ChallengeLabAllowUserToDeleteChallenge:       true,
				},
			},
			wantErr: false,
		},
		{
			name:          "no owners or editors and all controls false",
			callingUserId: "user@microsoft.com",
			challenge: entity.Challenge{
				UserId:    "user@microsoft.com",
				CreatedBy: "creator@microsoft.com",
			},
			lab: entity.LabType{
				LabControls: labentity.LabControlsType{
					ChallengeLabAllowChallengerToDeleteChallenge: false,
					ChallengeLabAllowUserToDeleteChallenge:       false,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := isDeleteAllowed(tt.callingUserId, tt.challenge, tt.lab)
			if (err != nil) != tt.wantErr {
				t.Errorf("isDeleteAllowed() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// --- Mocks ---

type mockChallengeRepository struct {
	challenge    entity.Challenge
	challenges   []entity.Challenge
	err          error
	upsertErr    error
	validateUser bool
}

func (m *mockChallengeRepository) GetAllChallenges(ctx context.Context) ([]entity.Challenge, error) {
	return m.challenges, m.err
}
func (m *mockChallengeRepository) GetChallengeByUserIdAndLabId(ctx context.Context, userId string, labId string) (entity.Challenge, error) {
	return m.challenge, m.err
}
func (m *mockChallengeRepository) GetChallengesByLabId(ctx context.Context, labId string) ([]entity.Challenge, error) {
	return m.challenges, m.err
}
func (m *mockChallengeRepository) GetChallengesByUserId(ctx context.Context, userId string) ([]entity.Challenge, error) {
	return m.challenges, m.err
}
func (m *mockChallengeRepository) DeleteChallenge(ctx context.Context, challengeId string) error {
	return m.err
}
func (m *mockChallengeRepository) UpsertChallenge(ctx context.Context, challenge entity.Challenge) error {
	if m.upsertErr != nil {
		return m.upsertErr
	}
	return m.err
}
func (m *mockChallengeRepository) ValidateUser(ctx context.Context, userId string) (bool, error) {
	return m.validateUser, m.err
}

type mockLabService struct {
	lab  entity.LabType
	labs []entity.LabType
	err  error
}

func (m *mockLabService) GetAllPrivateLabs(ctx context.Context, typeOfLab string) ([]entity.LabType, error) {
	return m.labs, m.err
}
func (m *mockLabService) GetPrivateLabs(ctx context.Context, typeOfLab string, userId string) ([]entity.LabType, error) {
	return m.labs, m.err
}
func (m *mockLabService) GetPrivateLab(ctx context.Context, typeOfLab string, labId string) (entity.LabType, error) {
	return m.lab, m.err
}
func (m *mockLabService) GetPrivateLabVersions(ctx context.Context, typeOfLab string, labId string, userId string) ([]entity.LabType, error) {
	return m.labs, m.err
}
func (m *mockLabService) UpsertPrivateLab(ctx context.Context, lab entity.LabType) (entity.LabType, error) {
	return m.lab, m.err
}
func (m *mockLabService) DeletePrivateLab(ctx context.Context, typeOfLab string, labId string, userId string) error {
	return m.err
}
func (m *mockLabService) GetPublicLabs(ctx context.Context, typeOfLab string) ([]entity.LabType, error) {
	return m.labs, m.err
}
func (m *mockLabService) GetPublicLabVersions(ctx context.Context, typeOfLab string, labId string) ([]entity.LabType, error) {
	return m.labs, m.err
}
func (m *mockLabService) UpsertPublicLab(ctx context.Context, lab entity.LabType) (entity.LabType, error) {
	return m.lab, m.err
}
func (m *mockLabService) DeletePublicLab(ctx context.Context, typeOfLab string, labId string, userId string) error {
	return m.err
}
func (m *mockLabService) GetProtectedLabs(ctx context.Context, typeOfLab string, userId string, requestIsWithSecret bool) ([]entity.LabType, error) {
	return m.labs, m.err
}
func (m *mockLabService) GetProtectedLab(ctx context.Context, typeOfLab string, labId string, userId string, requestIsWithSecret bool) (entity.LabType, error) {
	return m.lab, m.err
}
func (m *mockLabService) GetProtectedLabVersions(ctx context.Context, typeOfLab string, labId string) ([]entity.LabType, error) {
	return m.labs, m.err
}
func (m *mockLabService) UpsertProtectedLab(ctx context.Context, lab entity.LabType, userId string) (entity.LabType, error) {
	return m.lab, m.err
}
func (m *mockLabService) DeleteProtectedLab(ctx context.Context, typeOfLab string, labId string) error {
	return m.err
}
func (m *mockLabService) GetLabByIdAndType(ctx context.Context, typeOfLab string, labId string) (entity.LabType, error) {
	return m.lab, m.err
}
func (m *mockLabService) GetLabs(ctx context.Context, typeOfLab string) ([]entity.LabType, error) {
	return m.labs, m.err
}
func (m *mockLabService) GetLabVersions(ctx context.Context, typeOfLab string, labId string) ([]entity.LabType, error) {
	return m.labs, m.err
}
func (m *mockLabService) UpsertLab(ctx context.Context, lab entity.LabType) (entity.LabType, error) {
	return m.lab, m.err
}
func (m *mockLabService) DeleteLab(ctx context.Context, typeOfLab string, labId string) error {
	return m.err
}
func (m *mockLabService) UpsertSupportingDocument(ctx context.Context, supportingDocument multipart.File) (string, error) {
	return "", m.err
}
func (m *mockLabService) DeleteSupportingDocument(ctx context.Context, supportingDocumentId string) error {
	return m.err
}
func (m *mockLabService) GetSupportingDocument(ctx context.Context, supportingDocumentId string) (io.ReadCloser, error) {
	return nil, m.err
}
func (m *mockLabService) DoesSupportingDocumentExist(ctx context.Context, supportingDocumentId string) bool {
	return false
}

// --- Orchestrator Tests ---

func TestIsDeleteAllowedOrchestrator(t *testing.T) {
	t.Run("returns error when challenge not found", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				err: errors.New("not found"),
			},
		}
		err := svc.IsDeleteAllowed(context.Background(), "user@microsoft.com", "lab1")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("returns error when lab not found", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				challenge: entity.Challenge{LabId: "lab1"},
			},
			labService: &mockLabService{
				err: errors.New("not found"),
			},
		}
		err := svc.IsDeleteAllowed(context.Background(), "user@microsoft.com", "lab1")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("allows delete when calling user is owner", func(t *testing.T) {
		ctx := logger.WithUserID(context.Background(), "owner@microsoft.com")
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				challenge: entity.Challenge{
					LabId:     "lab1",
					UserId:    "user@microsoft.com",
					CreatedBy: "creator@microsoft.com",
				},
			},
			labService: &mockLabService{
				lab: entity.LabType{
					Owners: []string{"owner@microsoft.com"},
				},
			},
		}
		err := svc.IsDeleteAllowed(ctx, "user@microsoft.com", "lab1")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("denies delete when calling user has no permissions", func(t *testing.T) {
		ctx := logger.WithUserID(context.Background(), "random@microsoft.com")
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				challenge: entity.Challenge{
					LabId:     "lab1",
					UserId:    "user@microsoft.com",
					CreatedBy: "creator@microsoft.com",
				},
			},
			labService: &mockLabService{
				lab: entity.LabType{
					Owners: []string{"owner@microsoft.com"},
					LabControls: labentity.LabControlsType{
						ChallengeLabAllowChallengerToDeleteChallenge: false,
						ChallengeLabAllowUserToDeleteChallenge:       false,
					},
				},
			},
		}
		err := svc.IsDeleteAllowed(ctx, "user@microsoft.com", "lab1")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("uses calling user from context not from parameter", func(t *testing.T) {
		// userId param is the challenged user, but the calling user (from context) is the owner
		ctx := logger.WithUserID(context.Background(), "owner@microsoft.com")
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				challenge: entity.Challenge{
					LabId:     "lab1",
					UserId:    "attacker@microsoft.com",
					CreatedBy: "attacker@microsoft.com",
				},
			},
			labService: &mockLabService{
				lab: entity.LabType{
					Owners: []string{"owner@microsoft.com"},
					LabControls: labentity.LabControlsType{
						ChallengeLabAllowChallengerToDeleteChallenge: false,
						ChallengeLabAllowUserToDeleteChallenge:       false,
					},
				},
			},
		}
		// Even though lab controls are false, owner from context should be allowed
		err := svc.IsDeleteAllowed(ctx, "attacker@microsoft.com", "lab1")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})
}

func TestApplyStatusTransition(t *testing.T) {
	tests := []struct {
		name            string
		challenge       entity.Challenge
		status          string
		now             string
		wantStatus      string
		wantAcceptedOn  string
		wantCompletedOn string
		wantErr         bool
	}{
		{
			name: "accepted sets AcceptedOn and status",
			challenge: entity.Challenge{
				ChallengeId: "user@microsoft.com+lab1",
				UserId:      "user@microsoft.com",
				LabId:       "lab1",
				Status:      "challenged",
			},
			status:         entity.ChallengeStatusAccepted,
			now:            "2026-04-08T00:00:00Z",
			wantStatus:     entity.ChallengeStatusAccepted,
			wantAcceptedOn: "2026-04-08T00:00:00Z",
			wantErr:        false,
		},
		{
			name: "completed sets CompletedOn and status",
			challenge: entity.Challenge{
				ChallengeId: "user@microsoft.com+lab1",
				UserId:      "user@microsoft.com",
				LabId:       "lab1",
				Status:      "accepted",
				AcceptedOn:  "2026-04-07T00:00:00Z",
			},
			status:          entity.ChallengeStatusCompleted,
			now:             "2026-04-08T00:00:00Z",
			wantStatus:      entity.ChallengeStatusCompleted,
			wantAcceptedOn:  "2026-04-07T00:00:00Z",
			wantCompletedOn: "2026-04-08T00:00:00Z",
			wantErr:         false,
		},
		{
			name: "invalid status returns error",
			challenge: entity.Challenge{
				ChallengeId: "user@microsoft.com+lab1",
				UserId:      "user@microsoft.com",
				LabId:       "lab1",
				Status:      "challenged",
			},
			status:  "bogus",
			now:     "2026-04-08T00:00:00Z",
			wantErr: true,
		},
		{
			name: "accepted does not overwrite existing CompletedOn",
			challenge: entity.Challenge{
				ChallengeId: "user@microsoft.com+lab1",
				CompletedOn: "2026-04-06T00:00:00Z",
			},
			status:          entity.ChallengeStatusAccepted,
			now:             "2026-04-08T00:00:00Z",
			wantStatus:      entity.ChallengeStatusAccepted,
			wantAcceptedOn:  "2026-04-08T00:00:00Z",
			wantCompletedOn: "2026-04-06T00:00:00Z",
			wantErr:         false,
		},
		{
			name:      "empty status returns error",
			challenge: entity.Challenge{},
			status:    "",
			now:       "2026-04-08T00:00:00Z",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := applyStatusTransition(tt.challenge, tt.status, tt.now)
			if (err != nil) != tt.wantErr {
				t.Errorf("applyStatusTransition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", got.Status, tt.wantStatus)
			}
			if got.AcceptedOn != tt.wantAcceptedOn {
				t.Errorf("AcceptedOn = %q, want %q", got.AcceptedOn, tt.wantAcceptedOn)
			}
			if got.CompletedOn != tt.wantCompletedOn {
				t.Errorf("CompletedOn = %q, want %q", got.CompletedOn, tt.wantCompletedOn)
			}
		})
	}
}

func TestUpdateChallengeOrchestrator(t *testing.T) {
	t.Run("returns error when GetChallengesByUserId fails", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				err: errors.New("db error"),
			},
		}
		err := svc.UpdateChallenge(context.Background(), "user@microsoft.com", "lab1", "accepted")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("returns error when challenge not found", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				challenges: []entity.Challenge{
					{ChallengeId: "user@microsoft.com+lab-other", LabId: "lab-other", UserId: "user@microsoft.com"},
				},
			},
		}
		err := svc.UpdateChallenge(context.Background(), "user@microsoft.com", "lab1", "accepted")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("returns error when no challenges exist for user", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				challenges: []entity.Challenge{},
			},
		}
		err := svc.UpdateChallenge(context.Background(), "user@microsoft.com", "lab1", "accepted")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("returns error for invalid status", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				challenges: []entity.Challenge{
					{ChallengeId: "user@microsoft.com+lab1", LabId: "lab1", UserId: "user@microsoft.com"},
				},
			},
		}
		err := svc.UpdateChallenge(context.Background(), "user@microsoft.com", "lab1", "bogus")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("returns error when UpsertChallenge fails", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				challenges: []entity.Challenge{
					{ChallengeId: "user@microsoft.com+lab1", LabId: "lab1", UserId: "user@microsoft.com"},
				},
				upsertErr: errors.New("upsert failed"),
			},
		}
		err := svc.UpdateChallenge(context.Background(), "user@microsoft.com", "lab1", "accepted")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("succeeds with valid accepted status", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				challenges: []entity.Challenge{
					{ChallengeId: "user@microsoft.com+lab1", LabId: "lab1", UserId: "user@microsoft.com"},
				},
			},
		}
		err := svc.UpdateChallenge(context.Background(), "user@microsoft.com", "lab1", "accepted")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("succeeds with valid completed status", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				challenges: []entity.Challenge{
					{ChallengeId: "user@microsoft.com+lab1", LabId: "lab1", UserId: "user@microsoft.com"},
				},
			},
		}
		err := svc.UpdateChallenge(context.Background(), "user@microsoft.com", "lab1", "completed")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})
}
