package service

import (
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
	"actlabs-hub/internal/logger"
	"context"
	"errors"
	"fmt"
	"strings"
)

type challengeService struct {
	challengeRepository entity.ChallengeRepository
	labService          entity.LabService
}

func NewChallengeService(challengeRepository entity.ChallengeRepository, labService entity.LabService) entity.ChallengeService {
	return &challengeService{
		challengeRepository: challengeRepository,
		labService:          labService,
	}
}

func (a *challengeService) GetAllLabsRedacted(ctx context.Context) ([]entity.LabType, error) {
	challengeLabRedacted := []entity.LabType{}

	labs, err := a.labService.GetAllPrivateLabs(ctx, "challengelab")
	if err != nil {
		logger.LogError(ctx, "failed to get challenge labs",
			"error", err,
		)
		return challengeLabRedacted, errors.New("not able to get challenge labs")
	}

	for _, lab := range labs {
		// Only show published labs
		if !lab.IsPublished {
			continue
		}

		lab.ExtendScript = "redacted"
		lab.Description = lab.Message //Replace description with message
		lab.Type = "challenge"
		lab.Tags = []string{"challenge"}
		challengeLabRedacted = append(challengeLabRedacted, lab)
	}
	return challengeLabRedacted, nil
}

func (c *challengeService) GetChallengesLabsRedactedByUserId(ctx context.Context, userId string) ([]entity.LabType, error) {
	challengeLabs := []entity.LabType{}

	challenges, err := c.GetChallengesByUserId(ctx, userId)
	if err != nil {
		return challengeLabs, err
	}

	redactedLabs, err := c.GetAllLabsRedacted(ctx)
	if err != nil {
		return challengeLabs, err
	}

	for _, challenge := range challenges {
		for _, lab := range redactedLabs {
			if challenge.LabId == lab.Id && challenge.UserId == userId {
				challengeLabs = append(challengeLabs, lab)
				break
			}
		}
	}

	return challengeLabs, nil
}

func (c *challengeService) GetAllChallenges(ctx context.Context) ([]entity.Challenge, error) {
	challenges, err := c.challengeRepository.GetAllChallenges(ctx)
	if err != nil {
		logger.LogError(ctx, "failed to get all challenges",
			"error", err,
		)
		return challenges, errors.New("not able to get challenges")
	}

	return challenges, nil
}

func (c *challengeService) GetChallengesByLabId(ctx context.Context, labId string) ([]entity.Challenge, error) {
	challenges, err := c.challengeRepository.GetChallengesByLabId(ctx, labId)
	if err != nil {
		logger.LogError(ctx, "failed to get challenges by lab id",
			"lab_id", labId,
			"error", err,
		)
		return challenges, fmt.Errorf("not able to get challenges for lab id %s", labId)
	}

	return challenges, nil
}

func (c *challengeService) GetChallengesByUserId(ctx context.Context, userId string) ([]entity.Challenge, error) {
	challenges, err := c.challengeRepository.GetChallengesByUserId(ctx, userId)
	if err != nil {
		logger.LogError(ctx, "failed to get challenges by user id",
			"user_id", userId,
			"error", err,
		)
		return challenges, fmt.Errorf("not able to get challenges for user id %s", userId)
	}
	return challenges, nil
}

func (c *challengeService) UpsertChallenges(ctx context.Context, challenges []entity.Challenge) error {

	// Is createdBy owner or editor of the lab?
	// OR
	// Has createdBy completed the challenge? Yes? Have they challenged this to two people already? Yes? Return error.

	for _, challenge := range challenges {
		if challenge.Status == entity.ChallengeStatusAccepted {
			if challenge.ChallengeId == "" {
				challenge.AcceptedOn = helper.GetTodaysDateTimeString()
			}
		} else if challenge.Status == entity.ChallengeStatusCreated {
			if challenge.ChallengeId == "" {
				challenge.CreatedOn = helper.GetTodaysDateTimeString()
			}
		} else {
			logger.LogError(ctx, "invalid status",
				"user_id", challenge.UserId,
				"lab_id", challenge.LabId,
				"status", challenge.Status,
			)
			return errors.New("invalid status")
		}

		if err := c.challengeRepository.UpsertChallenge(ctx, challenge); err != nil {
			logger.LogError(ctx, "failed to upsert challenge",
				"user_id", challenge.UserId,
				"lab_id", challenge.LabId,
				"error", err,
			)
			return fmt.Errorf("not able to upsert challenge for user id %s and lab id %s. may be all challenges not added", challenge.UserId, challenge.LabId)
		}
	}

	return nil
}

func (c *challengeService) CreateChallenges(ctx context.Context, userIds []string, labIds []string, createdBy string) error {

	for _, userId := range userIds {

		if !strings.Contains(userId, "@microsoft.com") {
			userId = userId + "@microsoft.com"
		}

		valid, err := c.challengeRepository.ValidateUser(ctx, userId)
		if err != nil {
			logger.LogError(ctx, "failed to validate user id",
				"user_id", userId,
				"error", err,
			)
			continue
		}

		if !valid {
			err := errors.New("user id is not valid")
			logger.LogError(ctx, "user id is not valid",
				"user_id", userId,
				"error", err,
			)
			continue
		}

		for _, labId := range labIds {

			challenge := entity.Challenge{
				PartitionKey: userId,
				RowKey:       labId,
				ChallengeId:  userId + "+" + labId,
				UserId:       userId,
				LabId:        labId,
				CreatedBy:    createdBy,
				CreatedOn:    helper.GetTodaysDateTimeString(),
				Status:       "challenged",
			}

			if err := c.challengeRepository.UpsertChallenge(ctx, challenge); err != nil {
				logger.LogError(ctx, "failed to create challenge",
					"user_id", userId,
					"lab_id", labId,
					"error", err,
				)
				return fmt.Errorf("not able to create challenge for user id %s and lab id %s", userId, labId)
			}
		}
	}

	return nil
}

func (c *challengeService) UpdateChallenge(ctx context.Context, userId string, labId string, status string) error {
	challenges, err := c.challengeRepository.GetChallengesByUserId(ctx, userId)
	if err != nil {
		logger.LogError(ctx, "failed to get challenge",
			"user_id", userId,
			"lab_id", labId,
			"error", err,
		)
		return fmt.Errorf("not able to get challenges for user id %s", userId)
	}

	var challenge entity.Challenge
	for _, c := range challenges {
		if c.LabId == labId {
			challenge = c
			break
		}
	}

	if challenge.ChallengeId == "" {
		logger.LogError(ctx, "challenge not found",
			"user_id", userId,
			"lab_id", labId,
		)
		return fmt.Errorf("challenge not found for user id %s and lab id %s", userId, labId)
	}

	if status == entity.ChallengeStatusAccepted {
		challenge.AcceptedOn = helper.GetTodaysDateTimeString()
	} else if status == entity.ChallengeStatusCompleted {
		challenge.CompletedOn = helper.GetTodaysDateTimeString()
	} else {
		logger.LogError(ctx, "invalid status",
			"user_id", userId,
			"lab_id", labId,
			"status", status,
		)
		return errors.New("invalid status")
	}

	challenge.Status = status

	if err := c.challengeRepository.UpsertChallenge(ctx, challenge); err != nil {
		logger.LogError(ctx, "failed to update challenge",
			"user_id", userId,
			"lab_id", labId,
			"error", err,
		)
		return fmt.Errorf("not able to update challenge for user id %s and lab id %s", userId, labId)
	}

	return nil
}

func (c *challengeService) DeleteChallenges(ctx context.Context, challengeIds []string) error {
	for _, challengeId := range challengeIds {
		if err := c.challengeRepository.DeleteChallenge(ctx, challengeId); err != nil {
			logger.LogError(ctx, "failed to delete challenge",
				"challenge_id", challengeId,
				"error", err,
			)
			return fmt.Errorf("not able to delete challenge for challenge id %s. stopped processing remaining challenges", challengeId)
		}
	}

	return nil
}
