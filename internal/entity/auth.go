package entity

import (
	"github.com/gin-gonic/gin"
)

// These variables are used to store the Azure Storage Account name and SAS token.
// They must be set before the application starts.
var SasToken string
var StorageAccountName string

type Profile struct {
	ObjectId      string   `json:"objectId"`
	UserPrincipal string   `json:"userPrincipal"`
	DisplayName   string   `json:"displayName"`
	ProfilePhoto  string   `json:"profilePhoto"`
	Roles         []string `json:"roles"`
}

// Azure storage table doesn't support adding an array of strings. Thus, the hack.
// This is not the best way to do it, but it works for now.
type ProfileRecord struct {
	PartitionKey  string `json:"PartitionKey"`
	RowKey        string `json:"RowKey"`
	ObjectId      string `json:"ObjectId"`
	UserPrincipal string `json:"UserPrincipal"`
	DisplayName   string `json:"DisplayName"`
	ProfilePhoto  string `json:"ProfilePhoto"`
	Roles         string `json:"Roles"`
}

type AuthService interface {
	// Create a new user profile.
	// Privilege: User (can only create own profile)
	// Only allows 'user' role.
	CreateProfile(profile Profile) error

	// Get a given user's profile.
	// Privilege: User (can only get own profile)
	// Privilege: Admin (can get any profile)
	GetProfile(userPrincipal string) (Profile, error)

	// Get all profiles
	// Privilege: User
	GetAllProfilesRedacted() ([]Profile, error)

	// Get all profiles
	// Privilege: Admin
	GetAllProfiles() ([]Profile, error)

	// deletes the role from the user.
	// if the user has no roles left, then the user is deleted.
	// Privilege: Admin
	DeleteRole(userPrincipal string, role string) error

	// adds a role to the user.
	// User profile must exist before adding a role.
	// Privilege: Admin
	AddRole(userPrincipal string, role string) error
}

type AuthHandler interface {
	CreateProfile(c *gin.Context)
	GetProfile(c *gin.Context)
	GetAllProfiles(c *gin.Context)
	DeleteRole(c *gin.Context)
	AddRole(c *gin.Context)
}

type AuthRepository interface {
	// Get Profile from the table.
	GetProfile(userPrincipal string) (Profile, error)

	// Get all profiles from the table.
	GetAllProfiles() ([]Profile, error)

	// This method is used to delete the record for UserPrincipal from the table.
	// This is used only when the last role is removed from the user.
	DeleteProfile(userPrincipal string) error

	// This method is used to create or update profile.
	UpsertProfile(profile Profile) error
}
