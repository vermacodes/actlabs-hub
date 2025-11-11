package entity

import "context"

type ChallengeStatus = string

const (
	ChallengeStatusCreated   ChallengeStatus = "created"
	ChallengeStatusCompleted ChallengeStatus = "completed"
	ChallengeStatusFailed    ChallengeStatus = "failed"
	ChallengeStatusAccepted  ChallengeStatus = "accepted"
)

type Challenge struct {
	PartitionKey string          `json:"PartitionKey"`
	RowKey       string          `json:"RowKey"`
	ChallengeId  string          `json:"challengeId"`
	UserId       string          `json:"userId"`
	LabId        string          `json:"labId"`
	CreatedBy    string          `json:"createdBy"`
	CreatedOn    string          `json:"createdOn"`
	AcceptedOn   string          `json:"acceptedOn"`
	CompletedOn  string          `json:"completedOn"`
	Status       ChallengeStatus `json:"status"`
}

type BulkChallenge struct {
	UserIds []string `json:"userIds"`
	LabIds  []string `json:"labIds"`
}

type ChallengeService interface {
	// GetAllLabsRedacted retrieves all labs challenges of a user, with sensitive information redacted.
	// Returns an array of LabType (with redacted information) and any error encountered.
	GetAllLabsRedacted(ctx context.Context) ([]LabType, error)

	// GetMyChallengesLabs retrieves all labs challenges to a specific user.
	// userId: The ID of the user.
	// Returns an array of LabType and any error encountered.
	GetChallengesLabsRedactedByUserId(ctx context.Context, userId string) ([]LabType, error)

	// GetAllChallenges retrieves all available challenges.
	// Returns an array of challenges and any error encountered.
	GetAllChallenges(ctx context.Context) ([]Challenge, error)

	// GetChallengesByLabId retrieves challenges associated with a specific lab.
	// labId: The ID of the lab.
	// Returns an array of challenges and any error encountered.
	GetChallengesByLabId(ctx context.Context, labId string) ([]Challenge, error)

	// GetChallengesByUserId retrieves challenges associated with a specific user.
	// userId: The ID of the user.
	// Returns an array of challenges and any error encountered.
	GetChallengesByUserId(ctx context.Context, userId string) ([]Challenge, error)

	// UpsertChallenges upsert challenge.
	// Returns any error encountered.
	UpsertChallenges(ctx context.Context, Challenges []Challenge) error

	// UpdateChallenge updates a challenge.
	// userId : The ID of the user.
	// labId : The ID of the lab.
	// status: The new status of the challenge.
	// Returns any error encountered.
	UpdateChallenge(ctx context.Context, userId string, labId string, status string) error

	// CreateChallenges creates new challenges for a set of users and labs.
	// userIds: The IDs of the users.
	// labIds: The IDs of the labs.
	// createdBy: The ID of the user who created the challenges.
	// Returns any error encountered.
	CreateChallenges(ctx context.Context, userIds []string, labIds []string, createdBy string) error

	// DeleteChallenges deletes a set of challenges.
	// challengeIds: The IDs of the challenges to delete.
	// Returns any error encountered.
	DeleteChallenges(ctx context.Context, challengeIds []string) error
}

type ChallengeRepository interface {
	// GetAllChallenges retrieves all available challenges.
	// Returns an array of challenges and any error encountered.
	GetAllChallenges(ctx context.Context) ([]Challenge, error)

	// GetChallengesByLabId retrieves challenges associated with a specific lab.
	// labId: The ID of the lab.
	// Returns an array of challenges and any error encountered.
	GetChallengesByLabId(ctx context.Context, labId string) ([]Challenge, error)

	// GetChallengesByUserId retrieves challenges associated with a specific user.
	// userId: The ID of the user.
	// Returns an array of challenges and any error encountered.
	GetChallengesByUserId(ctx context.Context, userId string) ([]Challenge, error)

	// DeleteChallenge deletes a specific challenge.
	// challengeId: The ID of the challenge to delete.
	// Returns any error encountered.
	DeleteChallenge(ctx context.Context, challengeId string) error

	// UpsertChallenge inserts or updates an challenge.
	// challenge: The challenge to insert or update.
	// Returns any error encountered.
	UpsertChallenge(ctx context.Context, challenge Challenge) error

	// ValidateUser checks if a user is valid.
	// userId: The ID of the user to validate.
	// Returns a boolean indicating if the user is valid and any error encountered.
	ValidateUser(ctx context.Context, userId string) (bool, error)
}
