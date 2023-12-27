package repository

import (
	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
	"context"
	"encoding/json"

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
	slog.Debug("getting profile",
		slog.String("userPrincipal", userPrincipal),
	)

	principalRecord, err := r.auth.ActlabsProfilesTableClient.GetEntity(context.TODO(), "actlabs", userPrincipal, nil)
	if err != nil {
		slog.Debug("error getting entity",
			slog.String("userPrincipal", userPrincipal),
			slog.String("error", err.Error()),
		)
		return entity.Profile{}, err
	}

	profileRecord := entity.ProfileRecord{}
	if err := json.Unmarshal(principalRecord.Value, &profileRecord); err != nil {
		slog.Debug("error unmarshal principal record",
			slog.String("userPrincipal", userPrincipal),
			slog.String("error", err.Error()),
		)
		return entity.Profile{}, err
	}

	return helper.ConvertRecordToProfile(profileRecord), nil
}

func (r *AuthRepository) GetAllProfiles() ([]entity.Profile, error) {
	slog.Debug("getting all profiles")

	profile := entity.Profile{}
	profiles := []entity.Profile{}

	pager := r.auth.ActlabsProfilesTableClient.NewListEntitiesPager(nil)
	for pager.More() {
		response, err := pager.NextPage(context.Background())
		if err != nil {
			slog.Debug("error getting entities",
				slog.String("error", err.Error()),
			)
			return profiles, err
		}

		for _, entity := range response.Entities {
			var myEntity aztables.EDMEntity
			if err := json.Unmarshal(entity, &myEntity); err != nil {
				slog.Debug("error unmarshal principal record",
					slog.String("error", err.Error()),
				)
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
	slog.Debug("deleting profile",
		slog.String("userPrincipal", userPrincipal),
	)

	_, err := r.auth.ActlabsProfilesTableClient.DeleteEntity(context.TODO(), "actlabs", userPrincipal, nil)
	if err != nil {
		slog.Debug("error deleting entity",
			slog.String("userPrincipal", userPrincipal),
		)
	}
	return err
}

func (r *AuthRepository) UpsertProfile(profile entity.Profile) error {
	slog.Debug("upserting profile",
		slog.String("userPrincipal", profile.UserPrincipal),
	)

	profileRecord := helper.ConvertProfileToRecord(profile)

	marshalledPrincipalRecord, err := json.Marshal(profileRecord)
	if err != nil {
		slog.Debug("error marshalling principal record",
			slog.String("userPrincipal", profile.UserPrincipal),
			slog.String("error", err.Error()),
		)
		return err
	}

	_, err = r.auth.ActlabsProfilesTableClient.UpsertEntity(context.TODO(), marshalledPrincipalRecord, nil)
	if err != nil {
		slog.Debug("error adding entity",
			slog.String("userPrincipal", profile.UserPrincipal),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}
