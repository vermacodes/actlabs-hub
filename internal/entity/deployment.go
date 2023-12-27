package entity

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/data/aztables"
)

type DeploymentStatus string

const (
	InitInProgress       DeploymentStatus = "Init In Progress"
	InitFailed           DeploymentStatus = "Init Failed"
	InitCompleted        DeploymentStatus = "Init Completed"
	PlanInProgress       DeploymentStatus = "Plan In Progress"
	PlanFailed           DeploymentStatus = "Plan Failed"
	PlanCompleted        DeploymentStatus = "Plan Completed"
	DeploymentInProgress DeploymentStatus = "Deployment In Progress"
	DeploymentFailed     DeploymentStatus = "Deployment Failed"
	DeploymentCompleted  DeploymentStatus = "Deployment Completed"
	DeploymentNotStarted DeploymentStatus = "Deployment Not Started"
	DestroyInProgress    DeploymentStatus = "Destroy In Progress"
	DestroyCompleted     DeploymentStatus = "Destroy Completed"
	DestroyFailed        DeploymentStatus = "Destroy Failed"
)

type Deployment struct {
	//aztables.Entity              `json:"-"`
	DeploymentId                 string           `json:"deploymentId"`
	DeploymentUserId             string           `json:"deploymentUserId"`
	DeploymentSubscriptionId     string           `json:"deploymentSubscriptionId"`
	DeploymentWorkspace          string           `json:"deploymentWorkspace"`
	DeploymentStatus             DeploymentStatus `json:"deploymentStatus"`
	DeploymentLab                LabType          `json:"deploymentLab"`
	DeploymentAutoDelete         bool             `json:"deploymentAutoDelete"`
	DeploymentLifespan           int64            `json:"deploymentLifespan"`
	DeploymentAutoDeleteUnixTime int64            `json:"deploymentAutoDeleteUnixTime"`
}

type DeploymentEntry struct {
	aztables.Entity
	Deployment string
}

type OperationEntry struct {
	PartitionKey                 string           `json:"PartitionKey"`
	RowKey                       string           `json:"RowKey"`
	DeploymentWorkspace          string           `json:"DeploymentWorkspace"`
	DeploymentUserId             string           `json:"DeploymentUserId"`
	DeploymentSubscriptionId     string           `json:"DeploymentSubscriptionId"`
	DeploymentStatus             DeploymentStatus `json:"DeploymentStatus"`
	DeploymentAutoDelete         bool             `json:"DeploymentAutoDelete"`
	DeploymentLifespan           int64            `json:"DeploymentLifespan"`
	DeploymentAutoDeleteUnixTime int64            `json:"DeploymentAutoDeleteUnixTime"`
	DeploymentLab                string           `json:"DeploymentLab"`
}

type DeploymentService interface {
	GetAllDeployments(ctx context.Context) ([]Deployment, error)
	GetUserDeployments(ctx context.Context, userPrincipalName string) ([]Deployment, error)
	UpsertDeployment(ctx context.Context, deployment Deployment) error
	DeleteDeployment(ctx context.Context, userPrincipalName string, subscriptionId string, workspace string) error

	MonitorAndDeployAutoDestroyedServersToDestroyPendingDeployments(ctx context.Context)

	// may be this function doesn't belong here. but right now no other service needs this so keeping it here
	GetUserPrincipalNameByMSIPrincipalID(ctx context.Context, msiPrincipalID string) (string, error)
}

type DeploymentRepository interface {
	GetAllDeployments(ctx context.Context) ([]Deployment, error)
	GetUserDeployments(ctx context.Context, userPrincipalName string) ([]Deployment, error)
	GetDeployment(ctx context.Context, userPrincipalName string, workspace string, subscriptionId string) (Deployment, error)
	UpsertDeployment(ctx context.Context, deployment Deployment) error
	DeploymentOperationEntry(ctx context.Context, deployment Deployment) error
	DeleteDeployment(ctx context.Context, userPrincipalName string, subscriptionId string, workspace string) error

	// may be this function doesn't belong here. but right now no other service needs this so keeping it here
	GetUserPrincipalNameByMSIPrincipalID(ctx context.Context, msiPrincipalID string) (string, error)
}
