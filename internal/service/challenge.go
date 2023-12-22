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
	challengeLabRedacted := []entity.LabType{}

	labs, err := a.labService.GetAllPrivateLabs("challengelab")
	if err != nil {
		slog.Error("not able to get challenge labs", err)
		return challengeLabRedacted, err
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
	challengeLabs := []entity.LabType{}

	challenges, err := c.GetChallengesByUserId(userId)
	if err != nil {
		slog.Error("not able to get challenges for user "+userId, err)
		return challengeLabs, err
	}

	redactedLabs, err := c.GetAllLabsRedacted()
	if err != nil {
		slog.Error("not able to get redacted labs", err)
		return challengeLabs, err
	}

	for _, challenge := range challenges {
		slog.Debug("Challenge ID : " + challenge.ChallengeId)
		for _, lab := range redactedLabs {
			slog.Debug("Checking lab Name : " + lab.Name)
			if challenge.LabId == lab.Id && challenge.UserId == userId {
				slog.Debug("Challenge Id " + challenge.ChallengeId + " matches with lab Name " + lab.Name + " for user " + userId)
				challengeLabs = append(challengeLabs, lab)
				break
			}
		}
	}

	return challengeLabs, nil
}

func (c *challengeService) GetAllChallenges() ([]entity.Challenge, error) {
	challenges, err := c.challengeRepository.GetAllChallenges()
	if err != nil {
		return challenges, err
	}
	return challenges, nil
}

func (c *challengeService) GetChallengesByLabId(labId string) ([]entity.Challenge, error) {
	challenges, err := c.challengeRepository.GetChallengesByLabId(labId)
	if err != nil {
		return challenges, err
	}
	return challenges, nil
}

func (c *challengeService) GetChallengesByUserId(userId string) ([]entity.Challenge, error) {
	challenges, err := c.challengeRepository.GetChallengesByUserId(userId)
	if err != nil {
		return challenges, err
	}
	return challenges, nil
}

func (c *challengeService) UpsertChallenges(challenges []entity.Challenge) error {

	// Is createdBy owner or editor of the lab?
	// OR
	// Has createdBy completed the challenge? Yes? Have they challenged this to two people already? Yes? Return error.

	var _err error

	for _, challenge := range challenges {
		if err := c.challengeRepository.UpsertChallenge(challenge); err != nil {
			slog.Error(fmt.Sprintf("Not able to challenge %s for lab %s", challenge.UserId, challenge.LabId), err)
			_err = err
		}
	}

	return _err
}

func (c *challengeService) CreateChallenges(userIds []string, labIds []string, createdBy string) error {

	for _, userId := range userIds {

		if !strings.Contains(userId, "@microsoft.com") {
			userId = userId + "@microsoft.com"
		}

		valid, err := c.challengeRepository.ValidateUser(userId)
		if err != nil {
			slog.Error("not able to validate user id"+userId, err)
			continue
		}

		if !valid {
			err := errors.New("user id is not valid")
			slog.Error("user id is not valid"+userId, err)
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

			if err := c.challengeRepository.UpsertChallenge(challenge); err != nil {
				slog.Error("not able to create challenge", err)
				return err
			}

			slog.Debug(userId + " challenged to solve lab " + labId)
		}
	}

	return nil
}

func (c *challengeService) DeleteChallenges(challengeIds []string) error {
	for _, challengeId := range challengeIds {
		if err := c.challengeRepository.DeleteChallenge(challengeId); err != nil {
			slog.Error("not able to delete challenge with id "+challengeId, err)
			continue
		}
	}

	return nil
}
