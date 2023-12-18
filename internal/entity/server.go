package entity

const OwnerRoleDefinitionId string = "/subscriptions/da846304-0089-48e0-bfa7-65f68a3eb74f/providers/Microsoft.Authorization/roleDefinitions/8e3af657-a8ff-443c-a75c-2fe8c4bcb635"

type Server struct {
	PartitionKey                string `json:"PartitionKey"`
	RowKey                      string `json:"RowKey"`
	Endpoint                    string `json:"endpoint"`
	Status                      string `json:"status"`
	Region                      string `json:"region"`
	UserPrincipalId             string `json:"userPrincipalId"`
	UserPrincipalName           string `json:"userPrincipalName"`
	UserAlias                   string `json:"userAlias"`
	ManagedIdentityResourceId   string `json:"managedIdentityResourceId"`
	ManagedIdentityClientId     string `json:"managedIdentityClientId"`
	ManagedIdentityPrincipalId  string `json:"managedIdentityPrincipalId"`
	SubscriptionId              string `json:"subscriptionId"`
	ResourceGroup               string `json:"resourceGroup"`
	LogLevel                    string `json:"logLevel"`
	LastUserActivityTime        string `json:"lastActivityTime"`
	AutoCreate                  bool   `json:"autoCreate"`
	AutoDestroy                 bool   `json:"autoDestroy"`
	InactivityDurationInMinutes int    `json:"inactivityDurationInMinutes"`
}

type ServerService interface {
	DeployServer(server Server) (Server, error)
	DestroyServer(server Server) error
	GetServer(server Server) (Server, error)

	UpdateActivityStatus(userPrincipalName string) error
}

type ServerRepository interface {
	GetAzureContainerGroup(server Server) (Server, error)
	GetUserAssignedManagedIdentity(server Server) (Server, error)

	DeployAzureContainerGroup(server Server) (Server, error)
	CreateUserAssignedManagedIdentity(server Server) (Server, error)

	EnsureServerUp(server Server) error

	DestroyAzureContainerGroup(server Server) error

	IsUserOwner(server Server) (bool, error)

	UpsertServerInDatabase(server Server) error
	GetServerFromDatabase(partitionKey string, rowKey string) (Server, error)
}
