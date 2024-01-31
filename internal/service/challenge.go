package service

import (
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/exp/slog"
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

func (a *challengeService) GetAllLabsRedacted() ([]entity.LabType, error) {
	slog.Info("getting all challenge labs redacted")

	challengeLabRedacted := []entity.LabType{}

	labs, err := a.labService.GetAllPrivateLabs("challengelab")
	if err != nil {
		slog.Error("not able to get challenge labs",
			slog.String("error", err.Error()),
		)
		return challengeLabRedacted, errors.New("not able to get challenge labs")
	}

	for _, lab := range labs {
		slog.Debug("Lab ID : " + lab.Name)
		lab.ExtendScript = "redacted"
		lab.Description = lab.Message //Replace description with message
		lab.Type = "challenge"
		lab.Tags = []string{"challenge"}
		challengeLabRedacted = append(challengeLabRedacted, lab)
	}
	return challengeLabRedacted, nil
}

func (c *challengeService) GetChallengesLabsRedactedByUserId(userId string) ([]entity.LabType, error) {
	slog.Info("getting all challenge labs redacted by user",
		slog.String("userId", userId),
	)

	challengeLabs := []entity.LabType{}

	challenges, err := c.GetChallengesByUserId(userId)
	if err != nil {
		return challengeLabs, err
	}

	redactedLabs, err := c.GetAllLabsRedacted()
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

func (c *challengeService) GetAllChallenges() ([]entity.Challenge, error) {
	slog.Info("getting all challenges")

	challenges, err := c.challengeRepository.GetAllChallenges()
	if err != nil {
		slog.Error("not able to get all challenges",
			slog.String("error", err.Error()),
		)
		return challenges, errors.New("not able to get challenges")
	}

	return challenges, nil
}

func (c *challengeService) GetChallengesByLabId(labId string) ([]entity.Challenge, error) {
	slog.Info("getting challenges by lab id",
		slog.String("labId", labId),
	)

	challenges, err := c.challengeRepository.GetChallengesByLabId(labId)
	if err != nil {
		slog.Error("not able to get challenges by lab id",
			slog.String("labId", labId),
			slog.String("error", err.Error()),
		)

		return challenges, fmt.Errorf("not able to get challenges for lab id %s", labId)
	}

	return challenges, nil
}

func (c *challengeService) GetChallengesByUserId(userId string) ([]entity.Challenge, error) {
	slog.Info("getting challenges by user id",
		slog.String("userId", userId),
	)

	challenges, err := c.challengeRepository.GetChallengesByUserId(userId)
	if err != nil {
		slog.Error("not able to get challenges by user id",
			slog.String("userId", userId),
			slog.String("error", err.Error()),
		)

		return challenges, fmt.Errorf("not able to get challenges for user id %s", userId)
	}
	return challenges, nil
}

func (c *challengeService) UpsertChallenges(challenges []entity.Challenge) error {

	// Is createdBy owner or editor of the lab?
	// OR
	// Has createdBy completed the challenge? Yes? Have they challenged this to two people already? Yes? Return error.

	for _, challenge := range challenges {
		slog.Info("upserting challenge",
			slog.String("userId", challenge.UserId),
			slog.String("labId", challenge.LabId),
		)

		if err := c.challengeRepository.UpsertChallenge(challenge); err != nil {
			slog.Error("not able to upsert challenge",
				slog.String("userId", challenge.UserId),
				slog.String("labId", challenge.LabId),
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("not able to upsert challenge for user id %s and lab id %s. may be all challenges not added", challenge.UserId, challenge.LabId)
		}
	}

	return nil
}

func (c *challengeService) CreateChallenges(userIds []string, labIds []string, createdBy string) error {

	for _, userId := range userIds {

		if !strings.Contains(userId, "@microsoft.com") {
			userId = userId + "@microsoft.com"
		}

		valid, err := c.challengeRepository.ValidateUser(userId)
		if err != nil {
			slog.Error("not able to validate user id",
				slog.String("userId", userId),
				slog.String("error", err.Error()),
			)
			continue
		}

		if !valid {
			err := errors.New("user id is not valid")
			slog.Error("user id is not valid",
				slog.String("userId", userId),
				slog.String("error", err.Error()),
			)
			continue
		}

		for _, labId := range labIds {

			slog.Info("creating challenge",
				slog.String("userId", userId),
				slog.String("labId", labId),
			)

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

			if err := c.challengeRepository.UpsertChallenge(challenge); err != nil {
				slog.Error("not able to create challenge",
					slog.String("userId", userId),
					slog.String("labId", labId),
				)
				return fmt.Errorf("not able to create challenge for user id %s and lab id %s", userId, labId)
			}
		}
	}

	return nil
}

func (c *challengeService) UpdateChallenge(userId string, labId string, status string) error {
	slog.Info("updating challenge",
		slog.String("userId", userId),
		slog.String("labId", labId),
		slog.String("status", status),
	)

	challenges, err := c.challengeRepository.GetChallengesByUserId(userId)
	if err != nil {
		slog.Error("not able to get challenge",
			slog.String("userId", userId),
			slog.String("labId", labId),
			slog.String("error", err.Error()),
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
		slog.Error("challenge not found",
			slog.String("userId", userId),
			slog.String("labId", labId),
		)

		return fmt.Errorf("challenge not found for user id %s and lab id %s", userId, labId)
	}

	challenge.Status = status

	if err := c.challengeRepository.UpsertChallenge(challenge); err != nil {
		slog.Error("not able to update challenge",
			slog.String("userId", userId),
			slog.String("labId", labId),
			slog.String("error", err.Error()),
		)

		return fmt.Errorf("not able to update challenge for user id %s and lab id %s", userId, labId)
	}

	return nil
}

func (c *challengeService) DeleteChallenges(challengeIds []string) error {
	slog.Info("deleting challenges",
		slog.String("challengeIds", strings.Join(challengeIds, ",")),
	)

	for _, challengeId := range challengeIds {
		if err := c.challengeRepository.DeleteChallenge(challengeId); err != nil {
			slog.Error("not able to delete challenge",
				slog.String("challengeId", challengeId),
			)

			return fmt.Errorf("not able to delete challenge for challenge id %s. stopped processing remaining challenges", challengeId)
		}
	}

	return nil
}
