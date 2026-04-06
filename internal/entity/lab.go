package entity

import (
	"actlabs/labentity"
	"context"
	"io"
	"mime/multipart"
)

// Re-export shared types for backward compatibility within hub
type TfvarResourceGroupType = labentity.TfvarResourceGroupType
type TfvarDefaultNodePoolType = labentity.TfvarDefaultNodePoolType
type TfvarServiceMeshType = labentity.TfvarServiceMeshType
type TfvarAddonsType = labentity.TfvarAddonsType
type TfvarKubernetesClusterType = labentity.TfvarKubernetesClusterType
type TfvarAroClusterType = labentity.TfvarAroClusterType
type TfvarVirtualNetworkType = labentity.TfvarVirtualNetworkType
type TfvarSubnetType = labentity.TfvarSubnetType
type TfvarNetworkSecurityGroupType = labentity.TfvarNetworkSecurityGroupType
type TfvarJumpserverType = labentity.TfvarJumpserverType
type TfvarFirewallType = labentity.TfvarFirewallType
type ContainerRegistryType = labentity.ContainerRegistryType
type AppGatewayType = labentity.AppGatewayType
type TfvarConfigType = labentity.TfvarConfigType
type Blob = labentity.Blob
type LabType = labentity.LabType

// Re-export lab categories
var (
	PrivateLab    = labentity.PrivateLab
	PublicLab     = labentity.PublicLab
	ProtectedLabs = labentity.ProtectedLabs
)

type LabService interface {
	// Private Labs
	// Role: user
	// Types: privatelab, challengelab
	GetAllPrivateLabs(ctx context.Context, typeOfLab string) ([]LabType, error) // Don't expose via API directly.
	GetPrivateLabs(ctx context.Context, typeOfLab string, userId string) ([]LabType, error)
	GetPrivateLab(ctx context.Context, typeOfLab string, labId string) (LabType, error)
	GetPrivateLabVersions(ctx context.Context, typeOfLab string, labId string, userId string) ([]LabType, error)
	UpsertPrivateLab(ctx context.Context, lab LabType) (LabType, error)
	DeletePrivateLab(ctx context.Context, typeOfLab string, labId string, userId string) error

	// Public Labs
	// Role: user
	// Types: publiclab
	GetPublicLabs(ctx context.Context, typeOfLab string) ([]LabType, error)
	GetPublicLabVersions(ctx context.Context, typeOfLab string, labId string) ([]LabType, error)
	UpsertPublicLab(ctx context.Context, lab LabType) (LabType, error)
	DeletePublicLab(ctx context.Context, typeOfLab string, labId string, userId string) error

	// Protected Labs
	// Role: mentor
	// Types: readinesslab, mockcase
	GetProtectedLabs(ctx context.Context, typeOfLab string, userId string, requestIsWithSecret bool) ([]LabType, error)
	GetProtectedLab(ctx context.Context, typeOfLab string, labId string, userId string, requestIsWithSecret bool) (LabType, error)
	GetProtectedLabVersions(ctx context.Context, typeOfLab string, labId string) ([]LabType, error)
	UpsertProtectedLab(ctx context.Context, lab LabType, userId string) (LabType, error)
	DeleteProtectedLab(ctx context.Context, typeOfLab string, labId string) error

	// Shared functions
	GetLabs(ctx context.Context, typeOfLab string) ([]LabType, error)
	GetLabVersions(ctx context.Context, typeOfLab string, labId string) ([]LabType, error)
	UpsertLab(ctx context.Context, lab LabType) (LabType, error)
	DeleteLab(ctx context.Context, typeOfLab string, labId string) error

	// Supporting Documents
	UpsertSupportingDocument(ctx context.Context, supportingDocument multipart.File) (string, error)
	DeleteSupportingDocument(ctx context.Context, supportingDocumentId string) error
	GetSupportingDocument(ctx context.Context, supportingDocumentId string) (io.ReadCloser, error)
	DoesSupportingDocumentExist(ctx context.Context, supportingDocumentId string) bool
}

type LabRepository interface {
	ListBlobs(ctx context.Context, typeOfLab string) ([]Blob, error)

	GetLab(
		ctx context.Context,
		typeOfLab string,
		labId string,
	) (LabType, error) // send empty versionId ("") to get current version.
	GetLabWithVersions(ctx context.Context, typeOfLab string, labId string) ([]LabType, error)

	UpsertLab(ctx context.Context, labId string, lab string, typeOfLab string) error
	DeleteLab(ctx context.Context, typeOfLab string, labId string) error

	// Supporting Documents
	UpsertSupportingDocument(ctx context.Context, supportingDocument multipart.File) (string, error)
	DeleteSupportingDocument(ctx context.Context, supportingDocumentId string) error
	GetSupportingDocument(ctx context.Context, supportingDocumentId string) (io.ReadCloser, error)
	DoesSupportingDocumentExist(ctx context.Context, supportingDocumentId string) bool
}
