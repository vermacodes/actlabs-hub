package repository

import (
	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
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

func (c *challengeRepository) GetAllChallenges() ([]entity.Challenge, error) {
	challenge := entity.Challenge{}
	challenges := []entity.Challenge{}

	pager := c.auth.ActlabsChallengesTableClient.NewListEntitiesPager(nil)
	for pager.More() {
		response, err := pager.NextPage(context.Background())
		if err != nil {
			slog.Error("Error getting entities: ", err)
			return challenges, err
		}

		for _, element := range response.Entities {
			//var myEntity aztables.EDMEntity
			if err := json.Unmarshal(element, &challenge); err != nil {
				slog.Error("Error unmarshal entity: ", err)
				return challenges, err
			}
			challenges = append(challenges, challenge)
		}
	}

	return challenges, nil
}

func (c *challengeRepository) GetChallengesByLabId(labId string) ([]entity.Challenge, error) {
	challenge := entity.Challenge{}
	challenges := []entity.Challenge{}

	pager := c.auth.ActlabsChallengesTableClient.NewListEntitiesPager(nil)
	for pager.More() {
		response, err := pager.NextPage(context.Background())
		if err != nil {
			slog.Error("Error getting entities: ", err)
			return challenges, err
		}

		for _, element := range response.Entities {
			//var myEntity aztables.EDMEntity
			if err := json.Unmarshal(element, &challenge); err != nil {
				slog.Error("Error unmarshal entity: ", err)
				return challenges, err
			}
			challenges = append(challenges, challenge)
		}
	}

	return challenges, nil
}

func (c *challengeRepository) GetChallengesByUserId(userId string) ([]entity.Challenge, error) {
	challenge := entity.Challenge{}
	challenges := []entity.Challenge{}

	pager := c.auth.ActlabsChallengesTableClient.NewListEntitiesPager(nil)
	for pager.More() {
		response, err := pager.NextPage(context.Background())
		if err != nil {
			slog.Error("Error getting entities: ", err)
			return challenges, err
		}

		for _, element := range response.Entities {
			//var myEntity aztables.EDMEntity
			if err := json.Unmarshal(element, &challenge); err != nil {
				slog.Error("Error unmarshal entity: ", err)
				return challenges, err
			}

			if challenge.UserId == userId {
				challenges = append(challenges, challenge)
			}
		}
	}

	return challenges, nil
}

func (c *challengeRepository) DeleteChallenge(challengeId string) error {

	slog.Debug("Deleting challenge: " + challengeId)

	userId := strings.SplitN(challengeId, "+", 2)[1]

	_, err := c.auth.ActlabsChallengesTableClient.DeleteEntity(context.Background(), userId, challengeId, nil)
	if err != nil {
		slog.Error("Error deleting challenge: ", err)
		return err
	}
	slog.Debug("Deleted challenge: " + challengeId)

	return nil
}

func (c *challengeRepository) UpsertChallenge(challenge entity.Challenge) error {

	if challenge.ChallengeId == "" {
		challenge.ChallengeId = uuid.NewString()
	}

	challenge.PartitionKey = challenge.LabId
	challenge.RowKey = challenge.UserId + "+" + challenge.LabId

	val, err := json.Marshal(challenge)
	if err != nil {
		slog.Error("Error marshalling challenge: ", err)
		return err
	}

	_, err = c.auth.ActlabsChallengesTableClient.UpsertEntity(context.Background(), val, nil)
	if err != nil {
		slog.Error("Error updating/creating challenge: ", err)
		return err
	}

	slog.Debug("Challenge updated/created successfully")

	return nil
}

func (c *challengeRepository) ValidateUser(userId string) (bool, error) {
	return true, nil
}
