package repository

import (
	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
	"context"
	"encoding/json"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

type AuthRepository struct {
	auth      *auth.Auth
	appConfig *config.Config
	rdb       *redis.Client
}

func NewAuthRepository(
	auth *auth.Auth,
	appConfig *config.Config,
	rdb *redis.Client,
) (entity.AuthRepository, error) {
	return &AuthRepository{
		auth:      auth,
		appConfig: appConfig,
		rdb:       rdb,
	}, nil
}

func (r *AuthRepository) GetProfile(userPrincipal string) (entity.Profile, error) {
	principalRecord, err := r.auth.ActlabsProfilesTableClient.GetEntity(context.TODO(), "actlabs", userPrincipal, nil)
	if err != nil {
		slog.Error("Error getting entity: ", err)
		return entity.Profile{}, err
	}

	profileRecord := entity.ProfileRecord{}
	if err := json.Unmarshal(principalRecord.Value, &profileRecord); err != nil {
		slog.Error("Error unmarshal principal record: ", err)
		return entity.Profile{}, err
	}

	return helper.ConvertRecordToProfile(profileRecord), nil
}

func (r *AuthRepository) GetAllProfiles() ([]entity.Profile, error) {
	profiles := []entity.Profile{}
	profile := entity.Profile{}

	pager := r.auth.ActlabsProfilesTableClient.NewListEntitiesPager(nil)
	for pager.More() {
		response, err := pager.NextPage(context.Background())
		if err != nil {
			slog.Error("Error getting entities: ", err)
			return profiles, err
		}

		for _, entity := range response.Entities {
			var myEntity aztables.EDMEntity
			if err := json.Unmarshal(entity, &myEntity); err != nil {
				slog.Error("Error unmarshal principal record: ", err)
				return profiles, err
			}

			if value, ok := myEntity.Properties["ObjectId"]; ok {
				profile.ObjectId = value.(string)
			} else {
				profile.ObjectId = ""
			}

			if value, ok := myEntity.Properties["DisplayName"]; ok {
				profile.DisplayName = value.(string)
			} else {
				profile.DisplayName = ""
			}

			if value, ok := myEntity.Properties["ProfilePhoto"]; ok {
				profile.ProfilePhoto = value.(string)
			} else {
				profile.ProfilePhoto = ""
			}

			if value, ok := myEntity.Properties["UserPrincipal"]; ok {
				profile.UserPrincipal = value.(string)
			} else {
				profile.UserPrincipal = ""
			}

			if value, ok := myEntity.Properties["Roles"]; ok {
				profile.Roles = helper.StringToSlice(value.(string))
			} else {
				profile.Roles = []string{}
			}

			profiles = append(profiles, profile)
		}
	}

	return profiles, nil
}

// Use this function to complete delete the record for UserPrincipal.
func (r *AuthRepository) DeleteProfile(userPrincipal string) error {
	_, err := r.auth.ActlabsProfilesTableClient.DeleteEntity(context.TODO(), "actlabs", userPrincipal, nil)
	if err != nil {
		slog.Error("Error deleting entity: ", err)
	}
	return err
}

func (r *AuthRepository) UpsertProfile(profile entity.Profile) error {

	//Make sure that profile is complete
	if profile.DisplayName == "" || profile.UserPrincipal == "" {
		slog.Error("Error creating profile: profile is incomplete", nil)
		return errors.New("profile is incomplete")
	}

	profileRecord := helper.ConvertProfileToRecord(profile)

	marshalledPrincipalRecord, err := json.Marshal(profileRecord)
	if err != nil {
		slog.Error("Error marshalling principal record: ", err)
		return err
	}

	slog.Info("Adding or Updating entity")

	_, err = r.auth.ActlabsProfilesTableClient.UpsertEntity(context.TODO(), marshalledPrincipalRecord, nil)
	if err != nil {
		slog.Error("Error adding entity: ", err)
		return err
	}

	return nil
}
