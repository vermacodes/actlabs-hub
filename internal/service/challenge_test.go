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
		wantCreatedOn   string
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
			name: "created sets CreatedOn and status",
			challenge: entity.Challenge{
				UserId: "user@microsoft.com",
				LabId:  "lab1",
			},
			status:        entity.ChallengeStatusCreated,
			now:           "2026-04-08T00:00:00Z",
			wantStatus:    entity.ChallengeStatusCreated,
			wantCreatedOn: "2026-04-08T00:00:00Z",
			wantErr:       false,
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
			if got.CreatedOn != tt.wantCreatedOn {
				t.Errorf("CreatedOn = %q, want %q", got.CreatedOn, tt.wantCreatedOn)
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

func TestNormalizeUserId(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "valid full email",
			input: "user@microsoft.com",
			want:  "user@microsoft.com",
		},
		{
			name:  "alias without domain gets suffix appended",
			input: "ashish",
			want:  "ashish@microsoft.com",
		},
		{
			name:  "alias with hyphen is valid",
			input: "ashish-kumar",
			want:  "ashish-kumar@microsoft.com",
		},
		{
			name:    "empty string is rejected",
			input:   "",
			wantErr: true,
		},
		{
			name:    "non-microsoft domain is rejected",
			input:   "user@gmail.com",
			wantErr: true,
		},
		{
			name:    "similar domain like microsoft.community is rejected",
			input:   "user@microsoft.community",
			wantErr: true,
		},
		{
			name:    "attacker domain is rejected",
			input:   "user@evil.com",
			wantErr: true,
		},
		{
			name:    "uppercase letters in alias are rejected",
			input:   "User",
			wantErr: true,
		},
		{
			name:    "numbers in alias are rejected",
			input:   "user123",
			wantErr: true,
		},
		{
			name:    "special characters in alias are rejected",
			input:   "user.name",
			wantErr: true,
		},
		{
			name:    "spaces in alias are rejected",
			input:   "user name",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeUserId(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("normalizeUserId(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("normalizeUserId(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCreateChallengesOrchestrator(t *testing.T) {
	t.Run("skips invalid user ids", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				validateUser: true,
			},
		}
		// "User" has uppercase, should be skipped; no upsert call, no error
		err := svc.CreateChallenges(context.Background(), []string{"User"}, []string{"lab1"}, "creator@microsoft.com")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("skips attacker email domain", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				validateUser: true,
			},
		}
		err := svc.CreateChallenges(context.Background(), []string{"attacker@evil.com"}, []string{"lab1"}, "creator@microsoft.com")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("normalizes alias and creates challenge", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				validateUser: true,
			},
		}
		err := svc.CreateChallenges(context.Background(), []string{"ashish"}, []string{"lab1"}, "creator@microsoft.com")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("creates challenge for valid full email", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				validateUser: true,
			},
		}
		err := svc.CreateChallenges(context.Background(), []string{"user@microsoft.com"}, []string{"lab1"}, "creator@microsoft.com")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("returns error when upsert fails", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				validateUser: true,
				upsertErr:    errors.New("upsert failed"),
			},
		}
		err := svc.CreateChallenges(context.Background(), []string{"user@microsoft.com"}, []string{"lab1"}, "creator@microsoft.com")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("creates challenges for multiple users and labs", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				validateUser: true,
			},
		}
		err := svc.CreateChallenges(context.Background(), []string{"user-a", "user-b"}, []string{"lab1", "lab2"}, "creator@microsoft.com")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("skips empty user id", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				validateUser: true,
			},
		}
		err := svc.CreateChallenges(context.Background(), []string{""}, []string{"lab1"}, "creator@microsoft.com")
		if err != nil {
			t.Errorf("expected nil (skipped), got %v", err)
		}
	})

	t.Run("continues past invalid users and creates for valid ones", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				validateUser: true,
			},
		}
		err := svc.CreateChallenges(context.Background(), []string{"attacker@evil.com", "valid-user"}, []string{"lab1"}, "creator@microsoft.com")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})
}

func TestUpsertChallengesOrchestrator(t *testing.T) {
	t.Run("new challenge with accepted status sets AcceptedOn", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{},
		}
		challenges := []entity.Challenge{
			{
				ChallengeId: "",
				UserId:      "user@microsoft.com",
				LabId:       "lab1",
				Status:      entity.ChallengeStatusAccepted,
			},
		}
		err := svc.UpsertChallenges(context.Background(), challenges)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("new challenge with created status sets CreatedOn", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{},
		}
		challenges := []entity.Challenge{
			{
				ChallengeId: "",
				UserId:      "user@microsoft.com",
				LabId:       "lab1",
				Status:      entity.ChallengeStatusCreated,
			},
		}
		err := svc.UpsertChallenges(context.Background(), challenges)
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("new challenge with invalid status returns error", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{},
		}
		challenges := []entity.Challenge{
			{
				ChallengeId: "",
				UserId:      "user@microsoft.com",
				LabId:       "lab1",
				Status:      "bogus",
			},
		}
		err := svc.UpsertChallenges(context.Background(), challenges)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("existing challenge skips status transition", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{},
		}
		challenges := []entity.Challenge{
			{
				ChallengeId: "existing-id",
				UserId:      "user@microsoft.com",
				LabId:       "lab1",
				Status:      "bogus", // invalid but should be skipped since ChallengeId is set
			},
		}
		err := svc.UpsertChallenges(context.Background(), challenges)
		if err != nil {
			t.Errorf("expected nil (skipped transition), got %v", err)
		}
	})

	t.Run("returns error when upsert fails", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				upsertErr: errors.New("upsert failed"),
			},
		}
		challenges := []entity.Challenge{
			{
				ChallengeId: "",
				UserId:      "user@microsoft.com",
				LabId:       "lab1",
				Status:      entity.ChallengeStatusAccepted,
			},
		}
		err := svc.UpsertChallenges(context.Background(), challenges)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("processes multiple challenges and stops on first error", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				upsertErr: errors.New("upsert failed"),
			},
		}
		challenges := []entity.Challenge{
			{
				ChallengeId: "",
				UserId:      "user-a@microsoft.com",
				LabId:       "lab1",
				Status:      entity.ChallengeStatusCreated,
			},
			{
				ChallengeId: "",
				UserId:      "user-b@microsoft.com",
				LabId:       "lab2",
				Status:      entity.ChallengeStatusAccepted,
			},
		}
		err := svc.UpsertChallenges(context.Background(), challenges)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("succeeds with empty challenges list", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{},
		}
		err := svc.UpsertChallenges(context.Background(), []entity.Challenge{})
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})
}

func TestGetAllChallengesOrchestrator(t *testing.T) {
	t.Run("returns challenges on success", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				challenges: []entity.Challenge{
					{ChallengeId: "id1", UserId: "user@microsoft.com", LabId: "lab1"},
					{ChallengeId: "id2", UserId: "user@microsoft.com", LabId: "lab2"},
				},
			},
		}
		got, err := svc.GetAllChallenges(context.Background())
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
		if len(got) != 2 {
			t.Errorf("expected 2 challenges, got %d", len(got))
		}
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				err: errors.New("db error"),
			},
		}
		_, err := svc.GetAllChallenges(context.Background())
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("returns empty slice when no challenges", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				challenges: []entity.Challenge{},
			},
		}
		got, err := svc.GetAllChallenges(context.Background())
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected 0 challenges, got %d", len(got))
		}
	})
}

func TestGetChallengeByUserIdAndLabIdOrchestrator(t *testing.T) {
	t.Run("returns challenge on success", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				challenge: entity.Challenge{ChallengeId: "id1", UserId: "user@microsoft.com", LabId: "lab1"},
			},
		}
		got, err := svc.GetChallengeByUserIdAndLabId(context.Background(), "user@microsoft.com", "lab1")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
		if got.ChallengeId != "id1" {
			t.Errorf("expected ChallengeId=id1, got %s", got.ChallengeId)
		}
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				err: errors.New("not found"),
			},
		}
		_, err := svc.GetChallengeByUserIdAndLabId(context.Background(), "user@microsoft.com", "lab1")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestGetChallengesByLabIdOrchestrator(t *testing.T) {
	t.Run("returns challenges on success", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				challenges: []entity.Challenge{
					{ChallengeId: "id1", LabId: "lab1"},
					{ChallengeId: "id2", LabId: "lab1"},
				},
			},
		}
		got, err := svc.GetChallengesByLabId(context.Background(), "lab1")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
		if len(got) != 2 {
			t.Errorf("expected 2 challenges, got %d", len(got))
		}
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				err: errors.New("db error"),
			},
		}
		_, err := svc.GetChallengesByLabId(context.Background(), "lab1")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestGetChallengesByUserIdOrchestrator(t *testing.T) {
	t.Run("returns challenges on success", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				challenges: []entity.Challenge{
					{ChallengeId: "id1", UserId: "user@microsoft.com"},
					{ChallengeId: "id2", UserId: "user@microsoft.com"},
				},
			},
		}
		got, err := svc.GetChallengesByUserId(context.Background(), "user@microsoft.com")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
		if len(got) != 2 {
			t.Errorf("expected 2 challenges, got %d", len(got))
		}
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				err: errors.New("db error"),
			},
		}
		_, err := svc.GetChallengesByUserId(context.Background(), "user@microsoft.com")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestFilterLabsByChallenges(t *testing.T) {
	tests := []struct {
		name       string
		challenges []entity.Challenge
		labs       []entity.LabType
		wantIds    []string
	}{
		{
			name: "matches challenges to labs",
			challenges: []entity.Challenge{
				{LabId: "lab1"},
				{LabId: "lab3"},
			},
			labs: []entity.LabType{
				{Id: "lab1", Name: "Lab One"},
				{Id: "lab2", Name: "Lab Two"},
				{Id: "lab3", Name: "Lab Three"},
			},
			wantIds: []string{"lab1", "lab3"},
		},
		{
			name:       "empty challenges returns empty",
			challenges: []entity.Challenge{},
			labs: []entity.LabType{
				{Id: "lab1"},
			},
			wantIds: nil,
		},
		{
			name: "empty labs returns empty",
			challenges: []entity.Challenge{
				{LabId: "lab1"},
			},
			labs:    []entity.LabType{},
			wantIds: nil,
		},
		{
			name: "no matching lab ids",
			challenges: []entity.Challenge{
				{LabId: "lab99"},
			},
			labs: []entity.LabType{
				{Id: "lab1"},
				{Id: "lab2"},
			},
			wantIds: nil,
		},
		{
			name: "duplicate challenge lab ids only match once",
			challenges: []entity.Challenge{
				{LabId: "lab1"},
				{LabId: "lab1"},
			},
			labs: []entity.LabType{
				{Id: "lab1", Name: "Lab One"},
			},
			wantIds: []string{"lab1"},
		},
		{
			name:       "both empty returns empty",
			challenges: []entity.Challenge{},
			labs:       []entity.LabType{},
			wantIds:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterLabsByChallenges(tt.challenges, tt.labs)
			var gotIds []string
			for _, lab := range got {
				gotIds = append(gotIds, lab.Id)
			}
			if len(gotIds) != len(tt.wantIds) {
				t.Errorf("expected %v, got %v", tt.wantIds, gotIds)
				return
			}
			for i := range gotIds {
				if gotIds[i] != tt.wantIds[i] {
					t.Errorf("expected %v, got %v", tt.wantIds, gotIds)
					return
				}
			}
		})
	}
}

func TestGetChallengesLabsRedactedByUserIdOrchestrator(t *testing.T) {
	t.Run("returns matching redacted labs", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				challenges: []entity.Challenge{
					{LabId: "lab1", UserId: "user@microsoft.com"},
					{LabId: "lab2", UserId: "user@microsoft.com"},
				},
			},
			labService: &mockLabService{
				labs: []entity.LabType{
					{Id: "lab1", Name: "Lab One", ExtendScript: "secret", IsPublished: true, Message: "msg1"},
					{Id: "lab2", Name: "Lab Two", ExtendScript: "secret", IsPublished: true, Message: "msg2"},
					{Id: "lab3", Name: "Lab Three", ExtendScript: "secret", IsPublished: true, Message: "msg3"},
				},
			},
		}
		got, err := svc.GetChallengesLabsRedactedByUserId(context.Background(), "user@microsoft.com")
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 labs, got %d", len(got))
		}
		// Verify redaction was applied
		for _, lab := range got {
			if lab.ExtendScript != "redacted" {
				t.Errorf("expected ExtendScript=redacted, got %s", lab.ExtendScript)
			}
			if lab.Type != "challenge" {
				t.Errorf("expected Type=challenge, got %s", lab.Type)
			}
		}
	})

	t.Run("returns error when GetChallengesByUserId fails", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				err: errors.New("repo error"),
			},
			labService: &mockLabService{},
		}
		_, err := svc.GetChallengesLabsRedactedByUserId(context.Background(), "user@microsoft.com")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("returns error when GetAllLabsRedacted fails", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				challenges: []entity.Challenge{{LabId: "lab1"}},
			},
			labService: &mockLabService{
				err: errors.New("lab service error"),
			},
		}
		_, err := svc.GetChallengesLabsRedactedByUserId(context.Background(), "user@microsoft.com")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("returns nil when no challenges", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				challenges: []entity.Challenge{},
			},
			labService: &mockLabService{
				labs: []entity.LabType{
					{Id: "lab1", IsPublished: true, Message: "msg"},
				},
			},
		}
		got, err := svc.GetChallengesLabsRedactedByUserId(context.Background(), "user@microsoft.com")
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("skips unpublished labs", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				challenges: []entity.Challenge{
					{LabId: "lab1"},
					{LabId: "lab2"},
				},
			},
			labService: &mockLabService{
				labs: []entity.LabType{
					{Id: "lab1", IsPublished: true, Message: "msg1"},
					{Id: "lab2", IsPublished: false, Message: "msg2"},
				},
			},
		}
		got, err := svc.GetChallengesLabsRedactedByUserId(context.Background(), "user@microsoft.com")
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 lab, got %d", len(got))
		}
		if got[0].Id != "lab1" {
			t.Errorf("expected lab1, got %s", got[0].Id)
		}
	})

	t.Run("no matching labs returns nil", func(t *testing.T) {
		svc := &challengeService{
			challengeRepository: &mockChallengeRepository{
				challenges: []entity.Challenge{
					{LabId: "lab99"},
				},
			},
			labService: &mockLabService{
				labs: []entity.LabType{
					{Id: "lab1", IsPublished: true, Message: "msg"},
				},
			},
		}
		got, err := svc.GetChallengesLabsRedactedByUserId(context.Background(), "user@microsoft.com")
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}

func TestRedactLabs(t *testing.T) {
	tests := []struct {
		name    string
		labs    []entity.LabType
		wantLen int
		check   func(t *testing.T, got []entity.LabType)
	}{
		{
			name: "filters unpublished and redacts published",
			labs: []entity.LabType{
				{Id: "lab1", IsPublished: true, ExtendScript: "secret-script", Message: "challenge msg", Description: "original desc"},
				{Id: "lab2", IsPublished: false, ExtendScript: "secret"},
				{Id: "lab3", IsPublished: true, ExtendScript: "another-secret", Message: "msg3"},
			},
			wantLen: 2,
			check: func(t *testing.T, got []entity.LabType) {
				for _, lab := range got {
					if lab.ExtendScript != "redacted" {
						t.Errorf("lab %s: expected ExtendScript=redacted, got %s", lab.Id, lab.ExtendScript)
					}
					if lab.Type != "challenge" {
						t.Errorf("lab %s: expected Type=challenge, got %s", lab.Id, lab.Type)
					}
					if len(lab.Tags) != 1 || lab.Tags[0] != "challenge" {
						t.Errorf("lab %s: expected Tags=[challenge], got %v", lab.Id, lab.Tags)
					}
				}
			},
		},
		{
			name: "description replaced with message",
			labs: []entity.LabType{
				{Id: "lab1", IsPublished: true, Message: "challenge message", Description: "original"},
			},
			wantLen: 1,
			check: func(t *testing.T, got []entity.LabType) {
				if got[0].Description != "challenge message" {
					t.Errorf("expected Description='challenge message', got %s", got[0].Description)
				}
			},
		},
		{
			name:    "all unpublished returns nil",
			labs:    []entity.LabType{{Id: "lab1", IsPublished: false}},
			wantLen: 0,
		},
		{
			name:    "empty input returns nil",
			labs:    []entity.LabType{},
			wantLen: 0,
		},
		{
			name:    "nil input returns nil",
			labs:    nil,
			wantLen: 0,
		},
		{
			name: "preserves other fields",
			labs: []entity.LabType{
				{Id: "lab1", Name: "My Lab", IsPublished: true, Message: "msg", CreatedBy: "author"},
			},
			wantLen: 1,
			check: func(t *testing.T, got []entity.LabType) {
				if got[0].Id != "lab1" || got[0].Name != "My Lab" || got[0].CreatedBy != "author" {
					t.Errorf("expected other fields preserved, got %+v", got[0])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redactLabs(tt.labs)
			if len(got) != tt.wantLen {
				t.Fatalf("expected %d labs, got %d", tt.wantLen, len(got))
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestGetAllLabsRedactedOrchestrator(t *testing.T) {
	t.Run("returns redacted published labs", func(t *testing.T) {
		svc := &challengeService{
			labService: &mockLabService{
				labs: []entity.LabType{
					{Id: "lab1", IsPublished: true, ExtendScript: "secret", Message: "msg1"},
					{Id: "lab2", IsPublished: false, ExtendScript: "secret"},
				},
			},
		}
		got, err := svc.GetAllLabsRedacted(context.Background())
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 lab, got %d", len(got))
		}
		if got[0].ExtendScript != "redacted" {
			t.Errorf("expected ExtendScript=redacted, got %s", got[0].ExtendScript)
		}
	})

	t.Run("returns error when lab service fails", func(t *testing.T) {
		svc := &challengeService{
			labService: &mockLabService{
				err: errors.New("service error"),
			},
		}
		_, err := svc.GetAllLabsRedacted(context.Background())
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("returns nil when no labs", func(t *testing.T) {
		svc := &challengeService{
			labService: &mockLabService{
				labs: []entity.LabType{},
			},
		}
		got, err := svc.GetAllLabsRedacted(context.Background())
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		if got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})
}
