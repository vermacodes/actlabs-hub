package repository

import (
	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/logger"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type challengeRepository struct {
	auth      *auth.Auth
	appConfig *config.Config
	rdb       *redis.Client
}

func NewChallengeRepository(
	auth *auth.Auth,
	appConfig *config.Config,
	rdb *redis.Client,
) (entity.ChallengeRepository, error) {
	return &challengeRepository{
		auth:      auth,
		appConfig: appConfig,
		rdb:       rdb,
	}, nil
}

func (c *challengeRepository) GetAllChallenges(ctx context.Context) ([]entity.Challenge, error) {
	challenge := entity.Challenge{}
	challenges := []entity.Challenge{}

	pager := c.auth.ActlabsChallengesTableClient.NewListEntitiesPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			logger.LogError(ctx, "failed to get entities from table storage",
				"error", err,
			)
			return challenges, err
		}

		for _, entity := range page.Entities {
			if err := json.Unmarshal(entity, &challenge); err != nil {
				logger.LogError(ctx, "failed to unmarshal entity",
					"error", err,
				)
				continue
			}
			challenges = append(challenges, challenge)
		}
	}

	return challenges, nil
}

func (c *challengeRepository) GetChallengeByUserIdAndLabId(ctx context.Context, userId string, labId string) (entity.Challenge, error) {
	challenge := entity.Challenge{}
	rowKey := userId + "+" + labId

	response, err := c.auth.ActlabsChallengesTableClient.GetEntity(ctx, labId, rowKey, nil)
	if err != nil {
		logger.LogError(ctx, "failed to get challenge from table storage",
			"challenge_id", rowKey,
			"error", err,
		)
		return challenge, fmt.Errorf("challenge with id %s not found", rowKey)
	}

	if err := json.Unmarshal(response.Value, &challenge); err != nil {
		logger.LogError(ctx, "failed to unmarshal challenge",
			"challenge_id", rowKey,
			"error", err,
		)
		return challenge, err
	}

	return challenge, nil
}

func (c *challengeRepository) GetChallengesByLabId(ctx context.Context, labId string) ([]entity.Challenge, error) {
	challenge := entity.Challenge{}
	challenges := []entity.Challenge{}

	filter := fmt.Sprintf("PartitionKey eq '%s'", labId)
	pager := c.auth.ActlabsChallengesTableClient.NewListEntitiesPager(&aztables.ListEntitiesOptions{Filter: &filter})
	for pager.More() {
		response, err := pager.NextPage(ctx)
		if err != nil {
			logger.LogError(ctx, "failed to get entities from table storage",
				"lab_id", labId,
				"error", err,
			)
			return challenges, err
		}

		for _, element := range response.Entities {
			if err := json.Unmarshal(element, &challenge); err != nil {
				logger.LogError(ctx, "failed to unmarshal entity",
					"lab_id", labId,
					"error", err,
				)
				continue
			}
			challenges = append(challenges, challenge)
		}
	}

	return challenges, nil
}

func (c *challengeRepository) GetChallengesByUserId(ctx context.Context, userId string) ([]entity.Challenge, error) {
	challenge := entity.Challenge{}
	challenges := []entity.Challenge{}

	filter := fmt.Sprintf("userId eq '%s'", userId)
	pager := c.auth.ActlabsChallengesTableClient.NewListEntitiesPager(&aztables.ListEntitiesOptions{Filter: &filter})
	for pager.More() {
		response, err := pager.NextPage(ctx)
		if err != nil {
			logger.LogError(ctx, "failed to get entities from table storage",
				"user_id", userId,
				"error", err,
			)
			return challenges, err
		}

		for _, element := range response.Entities {
			if err := json.Unmarshal(element, &challenge); err != nil {
				logger.LogError(ctx, "failed to unmarshal entity",
					"user_id", userId,
					"error", err,
				)
				continue
			}
			challenges = append(challenges, challenge)
		}
	}

	return challenges, nil
}

func (c *challengeRepository) DeleteChallenge(ctx context.Context, challengeId string) error {
	// challengeId is in format of userId+labId. Split it to get labId.
	// PartitionKey in table storage is labId, RowKey is userId+labId
	partitionKey := strings.SplitN(challengeId, "+", 2)[1]

	_, err := c.auth.ActlabsChallengesTableClient.DeleteEntity(ctx, partitionKey, challengeId, nil)
	if err != nil {
		logger.LogError(ctx, "failed to delete challenge from table storage",
			"challenge_id", challengeId,
			"error", err,
		)
		return err
	}

	return nil
}

func (c *challengeRepository) UpsertChallenge(ctx context.Context, challenge entity.Challenge) error {
	if challenge.ChallengeId == "" {
		challenge.ChallengeId = uuid.NewString()
	}

	challenge.PartitionKey = challenge.LabId
	challenge.RowKey = challenge.UserId + "+" + challenge.LabId

	val, err := json.Marshal(challenge)
	if err != nil {
		logger.LogError(ctx, "failed to marshal challenge",
			"challenge_id", challenge.ChallengeId,
			"user_id", challenge.UserId,
			"lab_id", challenge.LabId,
			"error", err,
		)
		return err
	}

	_, err = c.auth.ActlabsChallengesTableClient.UpsertEntity(ctx, val, nil)
	if err != nil {
		logger.LogError(ctx, "failed to upsert challenge in table storage",
			"challenge_id", challenge.ChallengeId,
			"user_id", challenge.UserId,
			"lab_id", challenge.LabId,
			"error", err,
		)
		return err
	}

	return nil
}

func (c *challengeRepository) ValidateUser(ctx context.Context, userId string) (bool, error) {
	return true, nil
}
