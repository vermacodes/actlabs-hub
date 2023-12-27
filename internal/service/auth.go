package service

import (
	"actlabs-hub/internal/entity"
	"actlabs-hub/internal/helper"
	"errors"

	"golang.org/x/exp/slog"
)

type AuthService struct {
	authRepository entity.AuthRepository
}

func NewAuthService(authRepository entity.AuthRepository) entity.AuthService {
	return &AuthService{
		authRepository: authRepository,
	}
}

func (s *AuthService) CreateProfile(profile entity.Profile) error {

	existingProfile, err := s.authRepository.GetProfile(profile.UserPrincipal)
	if err != nil {
		slog.Error("Error getting existing profile",
			slog.String("userPrincipal", profile.UserPrincipal),
			slog.String("error", err.Error()),
		)
	}

	//Make sure that profile is complete
	if profile.DisplayName == "" || profile.UserPrincipal == "" {
		slog.Error("incomplete profile",
			slog.String("userPrincipal", profile.UserPrincipal),
			slog.String("displayName", profile.DisplayName),
			slog.String("error", "profile is incomplete"),
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

	if err := s.authRepository.UpsertProfile(profile); err != nil {
		slog.Error("error creating profile",
			slog.String("userPrincipal", profile.UserPrincipal),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}

func (s *AuthService) GetProfile(userPrincipal string) (entity.Profile, error) {
	profile, err := s.authRepository.GetProfile(userPrincipal)
	if err != nil {
		slog.Error("error getting profile",
			slog.String("userPrincipal", userPrincipal),
			slog.String("error", err.Error()),
		)
	}

	return profile, err
}

// return profiles without roles
func (s *AuthService) GetAllProfilesRedacted() ([]entity.Profile, error) {
	profiles, err := s.GetAllProfiles()
	if err != nil {
		return []entity.Profile{}, err
	}

	redactedProfiles := []entity.Profile{}
	for _, profile := range profiles {
		redactedProfile := entity.Profile{
			DisplayName:   profile.DisplayName,
			ProfilePhoto:  profile.ProfilePhoto,
			UserPrincipal: profile.UserPrincipal,
		}
		redactedProfiles = append(redactedProfiles, redactedProfile)
	}
	return redactedProfiles, err
}

func (s *AuthService) GetAllProfiles() ([]entity.Profile, error) {
	profiles, err := s.authRepository.GetAllProfiles()
	if err != nil {
		slog.Error("error getting profiles: ",
			slog.String("error", err.Error()),
		)
	}
	return profiles, err
}

func (s *AuthService) DeleteRole(userPrincipal string, role string) error {

	// Get the profile
	profile, err := s.GetProfile(userPrincipal)
	if err != nil {
		return err
	}

	// if the user has only one role, then delete the profile.
	profile.Roles = remove(profile.Roles, role)
	if len(profile.Roles) == 0 {
		return s.authRepository.DeleteProfile(userPrincipal)
	}

	// remove the the role and upsert the profile.
	profile.Roles = remove(profile.Roles, role)
	if err := s.authRepository.UpsertProfile(profile); err != nil {
		slog.Error("error deleting role",
			slog.String("userPrincipal", userPrincipal),
			slog.String("role", role),
		)

		return err
	}

	return nil
}

func (s *AuthService) AddRole(userPrincipal string, role string) error {

	// Get the profile
	profile, err := s.GetProfile(userPrincipal)
	if err != nil {
		return err
	}

	// if the role already exists, then return.
	if helper.Contains(profile.Roles, role) {
		slog.Debug("Role already exists: " + role)
		return nil
	}

	profile.Roles = append(profile.Roles, role)
	if err := s.authRepository.UpsertProfile(profile); err != nil {
		slog.Error("error deleting role",
			slog.String("userPrincipal", userPrincipal),
			slog.String("role", role),
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
