package entity

import (
	"context"
)

const OwnerRoleDefinitionId string = "/Microsoft.Authorization/roleDefinitions/8e3af657-a8ff-443c-a75c-2fe8c4bcb635"

type ServerStatus string

const (
	ServerStatusAutoDestroyed ServerStatus = "AutoDestroyed"
	ServerStatusDeployed      ServerStatus = "Deployed"
	ServerStatusDeploying     ServerStatus = "Deploying"
	ServerStatusDestroyed     ServerStatus = "Destroyed"
	ServerSTatusDestroying    ServerStatus = "Destroying"
	ServerStatusFailed        ServerStatus = "Failed"
	ServerStatusRegistered    ServerStatus = "Registered"
	ServerStatusRunning       ServerStatus = "Running"
	ServerStatusStarting      ServerStatus = "Starting"
	ServerStatusStopped       ServerStatus = "Stopped"
	ServerStatusStopping      ServerStatus = "Stopping"
	ServerStatusSucceeded     ServerStatus = "Succeeded"
	ServerStatusUnknown       ServerStatus = "Unknown"
	ServerStatusUnregistered  ServerStatus = "Unregistered"
	ServerStatusUpdating      ServerStatus = "Updating"
)

type Server struct {
	PartitionKey                string       `json:"PartitionKey"`
	RowKey                      string       `json:"RowKey"`
	Endpoint                    string       `json:"endpoint"`
	Status                      ServerStatus `json:"status"`
	Region                      string       `json:"region"`
	UserPrincipalId             string       `json:"userPrincipalId"`
	UserPrincipalName           string       `json:"userPrincipalName"`
	UserAlias                   string       `json:"userAlias"`
	ManagedIdentityResourceId   string       `json:"managedIdentityResourceId"`
	ManagedIdentityClientId     string       `json:"managedIdentityClientId"`
	ManagedIdentityPrincipalId  string       `json:"managedIdentityPrincipalId"`
	SubscriptionId              string       `json:"subscriptionId"`
	ResourceGroup               string       `json:"resourceGroup"`
	LogLevel                    string       `json:"logLevel"`
	LastUserActivityTime        string       `json:"lastActivityTime"`
	DestroyedAtTime             string       `json:"destroyedAtTime"`
	DeployedAtTime              string       `json:"deployedAtTime"`
	AutoCreate                  bool         `json:"autoCreate"`
	AutoDestroy                 bool         `json:"autoDestroy"`
	InactivityDurationInMinutes int          `json:"inactivityDurationInMinutes"`
}

type ManagedServerActionStatus struct {
	InProgress bool `json:"inProgress"`
}

type ServerService interface {
	RegisterSubscription(subscriptionId string, userPrincipalName string, userPrincipalId string) error

	UpdateServer(server Server) error // just updates in db. used to set flags like autoDestroy, autoCreate, etc.
	DeployServer(server Server) (Server, error)
	DestroyServer(userPrincipalName string) error
	GetServer(userPrincipalName string) (Server, error)

	UpdateActivityStatus(userPrincipalName string) error
}

type ServerRepository interface {
	GetAzureContainerGroup(server Server) (Server, error)
	GetUserAssignedManagedIdentity(server Server) (Server, error)

	DeployAzureContainerGroup(server Server) (Server, error)
	CreateUserAssignedManagedIdentity(server Server) (Server, error)

	EnsureServerUp(server Server) error
	EnsureServerIdle(server Server) (bool, error)

	DestroyAzureContainerGroup(server Server) error

	IsUserOwner(server Server) (bool, error)

	UpsertServerInDatabase(server Server) error
	GetServerFromDatabase(partitionKey string, rowKey string) (Server, error)
	GetAllServersFromDatabase(ctx context.Context) ([]Server, error)
}
