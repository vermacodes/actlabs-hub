package repository

import (
	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
	"actlabs-hub/internal/logger"
	"context"
	"encoding/json"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
	"github.com/redis/go-redis/v9"
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

func (r *AuthRepository) GetProfile(ctx context.Context, userPrincipal string) (entity.Profile, error) {
	client := r.auth.ActlabsProfilesTableClient

	filter := fmt.Sprintf("RowKey eq '%s'", userPrincipal)
	listOptions := &aztables.ListEntitiesOptions{
		Filter: &filter,
	}

	profileRecord := entity.ProfileRecord{}
	pager := client.NewListEntitiesPager(listOptions)

	for pager.More() {
		response, err := pager.NextPage(ctx)
		if err != nil {
			logger.LogError(ctx, "failed to query profile from table storage",
				"user_principal", userPrincipal,
				"error", err,
			)
			return entity.Profile{}, err
		}

		for _, item := range response.Entities {
			err := json.Unmarshal(item, &profileRecord)
			if err != nil {
				logger.LogError(ctx, "failed to unmarshal profile record",
					"user_principal", userPrincipal,
					"error", err,
				)
				return entity.Profile{}, err
			}
		}
	}

	profile := helper.ConvertRecordToProfile(profileRecord)
	return profile, nil
}

func (r *AuthRepository) GetAllProfiles(ctx context.Context) ([]entity.Profile, error) {
	profile := entity.Profile{}
	profiles := []entity.Profile{}

	pager := r.auth.ActlabsProfilesTableClient.NewListEntitiesPager(nil)
	for pager.More() {
		response, err := pager.NextPage(ctx)
		if err != nil {
			logger.LogError(ctx, "failed to get entities from table storage",
				"error", err,
			)
			return profiles, err
		}

		for _, entity := range response.Entities {
			var myEntity aztables.EDMEntity
			if err := json.Unmarshal(entity, &myEntity); err != nil {
				logger.LogError(ctx, "failed to unmarshal profile record",
					"error", err,
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
func (r *AuthRepository) DeleteProfile(ctx context.Context, userPrincipal string) error {
	_, err := r.auth.ActlabsProfilesTableClient.DeleteEntity(ctx, "actlabs", userPrincipal, nil)
	if err != nil {
		logger.LogError(ctx, "failed to delete profile from table storage",
			"user_principal", userPrincipal,
			"error", err,
		)
		return err
	}
	return nil
}

func (r *AuthRepository) UpsertProfile(ctx context.Context, profile entity.Profile) error {
	profileRecord := helper.ConvertProfileToRecord(profile)

	marshalledPrincipalRecord, err := json.Marshal(profileRecord)
	if err != nil {
		logger.LogError(ctx, "failed to marshal profile record",
			"user_principal", profile.UserPrincipal,
			"error", err,
		)
		return err
	}

	_, err = r.auth.ActlabsProfilesTableClient.UpsertEntity(ctx, marshalledPrincipalRecord, nil)
	if err != nil {
		logger.LogError(ctx, "failed to upsert profile to table storage",
			"user_principal", profile.UserPrincipal,
			"error", err,
		)
		return err
	}

	return nil
}
