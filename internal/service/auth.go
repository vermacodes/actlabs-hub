package service

import (
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
	"actlabs-hub/internal/logger"
	"context"
	"errors"
)

type AuthService struct {
	authRepository entity.AuthRepository
}

func NewAuthService(authRepository entity.AuthRepository) entity.AuthService {
	return &AuthService{
		authRepository: authRepository,
	}
}

func (s *AuthService) CreateProfile(ctx context.Context, profile entity.Profile) error {
	logger.LogInfo(ctx, "creating user profile",
		"user_principal", profile.UserPrincipal,
	)

	existingProfile, err := s.authRepository.GetProfile(ctx, profile.UserPrincipal)
	if err != nil {
		logger.LogError(ctx, "failed to check existing profile",
			"user_principal", profile.UserPrincipal,
			"error", err,
		)
	}

	//Make sure that profile is complete
	if profile.DisplayName == "" || profile.UserPrincipal == "" {
		logger.LogError(ctx, "profile validation failed - missing required fields",
			"user_principal", profile.UserPrincipal,
			"display_name", profile.DisplayName,
		)
		return errors.New("profile is incomplete")
	}

	// if the user already exists, then update the profile with existing roles
	if existingProfile.UserPrincipal != "" {
		profile.Roles = existingProfile.Roles
	} else {
		// if the user does not exist, then add the user role
		profile.Roles = []string{"user"} // IMPORTANT: remove all roles and add only user role
	}

	if err := s.authRepository.UpsertProfile(ctx, profile); err != nil {
		logger.LogError(ctx, "failed to upsert profile",
			"user_principal", profile.UserPrincipal,
			"error", err,
		)
		return err
	}

	return nil
}

func (s *AuthService) GetProfile(ctx context.Context, userPrincipal string) (entity.Profile, error) {
	profile, err := s.authRepository.GetProfile(ctx, userPrincipal)
	if err != nil {
		logger.LogError(ctx, "failed to get profile",
			"user_principal", userPrincipal,
			"error", err,
		)
		return entity.Profile{}, err
	}

	return profile, nil
}

// return profiles without roles - returns all profiles but potentially redacted for regular users
func (s *AuthService) GetAllProfilesRedacted(ctx context.Context) ([]entity.Profile, error) {
	profiles, err := s.GetAllProfiles(ctx)
	if err != nil {
		return nil, err
	}

	// For now, return the same profiles - redaction logic can be added later if needed
	return profiles, nil
}

func (s *AuthService) GetAllProfiles(ctx context.Context) ([]entity.Profile, error) {
	profiles, err := s.authRepository.GetAllProfiles(ctx)
	if err != nil {
		logger.LogError(ctx, "failed to get all profiles",
			"error", err,
		)
		return nil, err
	}

	return profiles, nil
}

func (s *AuthService) DeleteRole(ctx context.Context, userPrincipal string, role string) error {
	logger.LogInfo(ctx, "deleting role from user",
		"user_principal", userPrincipal,
		"role", role,
	)

	profile, err := s.GetProfile(ctx, userPrincipal)
	if err != nil {
		return err
	}

	// if the profile has only one role, then delete the profile
	if len(profile.Roles) == 1 {
		return s.authRepository.DeleteProfile(ctx, userPrincipal)
	}

	// remove the role from the profile
	profile.Roles = remove(profile.Roles, role)
	if err := s.authRepository.UpsertProfile(ctx, profile); err != nil {
		logger.LogError(ctx, "failed to update profile after role deletion",
			"user_principal", userPrincipal,
			"role", role,
			"error", err,
		)
		return err
	}

	return nil
}

func (s *AuthService) AddRole(ctx context.Context, userPrincipal string, role string) error {
	logger.LogInfo(ctx, "adding role to user",
		"user_principal", userPrincipal,
		"role", role,
	)

	// Get the profile
	profile, err := s.GetProfile(ctx, userPrincipal)
	if err != nil {
		return err
	}

	// if the role already exists, then return.
	if helper.Contains(profile.Roles, role) {
		return nil
	}

	profile.Roles = append(profile.Roles, role)
	if err := s.authRepository.UpsertProfile(ctx, profile); err != nil {
		logger.LogError(ctx, "failed to update profile after role addition",
			"user_principal", userPrincipal,
			"role", role,
			"error", err,
		)
		return err
	}

	return nil
}

// Helper Function to remove an element from a slice
func remove(roles []string, role string) []string {
	for i, v := range roles {
		if v == role {
			roles = append(roles[:i], roles[i+1:]...)
			break
		}
	}
	return roles
}
