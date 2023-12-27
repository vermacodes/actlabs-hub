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
	slog.Debug("getting all challenges")

	challenge := entity.Challenge{}
	challenges := []entity.Challenge{}

	pager := c.auth.ActlabsChallengesTableClient.NewListEntitiesPager(nil)
	for pager.More() {
		response, err := pager.NextPage(context.Background())
		if err != nil {
			slog.Debug("error getting entities",
				slog.String("error", err.Error()),
			)
			return challenges, err
		}

		for _, element := range response.Entities {
			//var myEntity aztables.EDMEntity
			if err := json.Unmarshal(element, &challenge); err != nil {
				slog.Debug("error unmarshal entity",
					slog.String("error", err.Error()),
				)
				return challenges, err
			}
			challenges = append(challenges, challenge)
		}
	}

	return challenges, nil
}

func (c *challengeRepository) GetChallengesByLabId(labId string) ([]entity.Challenge, error) {
	slog.Debug("getting challenges by lab id",
		slog.String("labId", labId),
	)

	challenge := entity.Challenge{}
	challenges := []entity.Challenge{}

	pager := c.auth.ActlabsChallengesTableClient.NewListEntitiesPager(nil)
	for pager.More() {
		response, err := pager.NextPage(context.Background())
		if err != nil {
			slog.Debug("error getting entities",
				slog.String("labId", labId),
				slog.String("error", err.Error()),
			)
			return challenges, err
		}

		for _, element := range response.Entities {
			//var myEntity aztables.EDMEntity
			if err := json.Unmarshal(element, &challenge); err != nil {
				slog.Debug("error unmarshal entity",
					slog.String("labId", labId),
					slog.String("error", err.Error()),
				)
				return challenges, err
			}
			challenges = append(challenges, challenge)
		}
	}

	return challenges, nil
}

func (c *challengeRepository) GetChallengesByUserId(userId string) ([]entity.Challenge, error) {
	slog.Debug("getting challenges by user id",
		slog.String("userId", userId),
	)

	challenge := entity.Challenge{}
	challenges := []entity.Challenge{}

	pager := c.auth.ActlabsChallengesTableClient.NewListEntitiesPager(nil)
	for pager.More() {
		response, err := pager.NextPage(context.Background())
		if err != nil {
			slog.Debug("error getting entities: ",
				slog.String("userId", userId),
			)
			return challenges, err
		}

		for _, element := range response.Entities {
			//var myEntity aztables.EDMEntity
			if err := json.Unmarshal(element, &challenge); err != nil {
				slog.Debug("error unmarshal entity: ",
					slog.String("userId", userId),
				)
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
	slog.Debug("deleting challenge",
		slog.String("challengeId", challengeId),
	)

	userId := strings.SplitN(challengeId, "+", 2)[1]

	_, err := c.auth.ActlabsChallengesTableClient.DeleteEntity(context.Background(), userId, challengeId, nil)
	if err != nil {
		slog.Debug("error deleting challenge",
			slog.String("challengeId", challengeId),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}

func (c *challengeRepository) UpsertChallenge(challenge entity.Challenge) error {
	slog.Debug("upserting challenge",
		slog.String("challengeId", challenge.ChallengeId),
		slog.String("userId", challenge.UserId),
		slog.String("labId", challenge.LabId),
	)

	if challenge.ChallengeId == "" {
		challenge.ChallengeId = uuid.NewString()
	}

	challenge.PartitionKey = challenge.LabId
	challenge.RowKey = challenge.UserId + "+" + challenge.LabId

	val, err := json.Marshal(challenge)
	if err != nil {
		slog.Debug("error marshalling challenge",
			slog.String("challengeId", challenge.ChallengeId),
			slog.String("userId", challenge.UserId),
			slog.String("labId", challenge.LabId),
			slog.String("error", err.Error()),
		)
		return err
	}

	_, err = c.auth.ActlabsChallengesTableClient.UpsertEntity(context.Background(), val, nil)
	if err != nil {
		slog.Debug("error updating/creating challenge",
			slog.String("challengeId", challenge.ChallengeId),
			slog.String("userId", challenge.UserId),
			slog.String("labId", challenge.LabId),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}

func (c *challengeRepository) ValidateUser(userId string) (bool, error) {
	return true, nil
}
