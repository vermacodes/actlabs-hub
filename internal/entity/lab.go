package entity

import (
	"context"
	"io"
	"mime/multipart"
)

type TfvarResourceGroupType struct {
	Location string `json:"location"`
}

type TfvarDefaultNodePoolType struct {
	EnableAutoScaling         bool   `json:"enableAutoScaling"`
	MinCount                  int    `json:"minCount"`
	MaxCount                  int    `json:"maxCount"`
	VmSize                    string `json:"vmSize"`
	OnlyCriticalAddonsEnabled bool   `json:"onlyCriticalAddonsEnabled"`
	OsSku                     string `json:"osSku"`
}

type TfvarAddonsType struct {
	AppGateway        bool `json:"appGateway"`
	MicrosoftDefender bool `json:"microsoftDefender"`
}

type TfvarKubernetesClusterType struct {
	KubernetesVersion     string                   `json:"kubernetesVersion"`
	NetworkPlugin         string                   `json:"networkPlugin"`
	NetworkPolicy         string                   `json:"networkPolicy"`
	NetworkPluginMode     string                   `json:"networkPluginMode"`
	OutboundType          string                   `json:"outboundType"`
	PrivateClusterEnabled string                   `json:"privateClusterEnabled"`
	Addons                TfvarAddonsType          `json:"addons"`
	DefaultNodePool       TfvarDefaultNodePoolType `json:"defaultNodePool"`
}

type TfvarVirtualNetworkType struct {
	AddressSpace []string
}

type TfvarSubnetType struct {
	Name            string
	AddressPrefixes []string
}

type TfvarNetworkSecurityGroupType struct{}

type TfvarJumpserverType struct {
	AdminPassword string `json:"adminPassword"`
	AdminUserName string `json:"adminUsername"`
}

type TfvarFirewallType struct {
	SkuName string `json:"skuName"`
	SkuTier string `json:"skuTier"`
}

type ContainerRegistryType struct{}

type AppGatewayType struct{}

type TfvarConfigType struct {
	ResourceGroup         TfvarResourceGroupType          `json:"resourceGroup"`
	VirtualNetworks       []TfvarVirtualNetworkType       `json:"virtualNetworks"`
	Subnets               []TfvarSubnetType               `json:"subnets"`
	Jumpservers           []TfvarJumpserverType           `json:"jumpservers"`
	NetworkSecurityGroups []TfvarNetworkSecurityGroupType `json:"networkSecurityGroups"`
	KubernetesClusters    []TfvarKubernetesClusterType    `json:"kubernetesClusters"`
	Firewalls             []TfvarFirewallType             `json:"firewalls"`
	ContainerRegistries   []ContainerRegistryType         `json:"containerRegistries"`
	AppGateways           []AppGatewayType                `json:"appGateways"`
}

// didn't find exported equivalent of https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/storage/azblob@v1.2.1/internal/generated#BlobItem
type Blob struct {
	Name             string `xml:"Name"             json:"name"`
	VersionId        string `xml:"VersionId"        json:"versionId"`
	IsCurrentVersion bool   `xml:"IsCurrentVersion" json:"isCurrentVersion"`
	// Url  string `xml:"Url" json:"url"`
}

var (
	PrivateLab    = []string{"privatelab", "challengelab"}
	PublicLab     = []string{"publiclab"}
	ProtectedLabs = []string{"readinesslab", "mockcase"}
)

type LabType struct {
	Id                       string          `json:"id"`
	Name                     string          `json:"name"`
	Description              string          `json:"description"`
	Tags                     []string        `json:"tags"`
	Template                 TfvarConfigType `json:"template"`
	ExtendScript             string          `json:"extendScript"`
	Message                  string          `json:"message"`
	Category                 string          `json:"category"`
	Type                     string          `json:"type"`
	CreatedBy                string          `json:"createdBy"`
	CreatedOn                string          `json:"createdOn"`
	UpdatedBy                string          `json:"updatedBy"`
	UpdatedOn                string          `json:"updatedOn"`
	Owners                   []string        `json:"owners"`
	Editors                  []string        `json:"editors"`
	Viewers                  []string        `json:"viewers"`
	RbacEnforcedProtectedLab bool            `json:"rbacEnforcedProtectedLab"`
	IsPublished              bool            `json:"isPublished"`
	VersionId                string          `json:"versionId"`
	IsCurrentVersion         bool            `json:"isCurrentVersion"`
	SupportingDocumentId     string          `json:"supportingDocumentId"`
}

type LabService interface {
	// Private Labs
	// Role: user
	// Types: privatelab, challengelab
	GetAllPrivateLabs(typeOfLab string) ([]LabType, error) // Don't expose via API directly.
	GetPrivateLabs(typeOfLab string, userId string) ([]LabType, error)
	GetPrivateLab(typeOfLab string, labId string) (LabType, error)
	GetPrivateLabVersions(typeOfLab string, labId string, userId string) ([]LabType, error)
	UpsertPrivateLab(LabType) (LabType, error)
	DeletePrivateLab(typeOfLab string, labId string, userId string) error

	// Public Labs
	// Role: user
	// Types: publiclab
	GetPublicLabs(typeOfLab string) ([]LabType, error)
	GetPublicLabVersions(typeOfLab string, labId string) ([]LabType, error)
	UpsertPublicLab(LabType) (LabType, error)
	DeletePublicLab(typeOfLab string, labId string, userId string) error

	// Protected Labs
	// Role: mentor
	// Types: readinesslab, mockcase
	GetProtectedLabs(typeOfLab string, userId string, requestIsWithSecret bool) ([]LabType, error)
	GetProtectedLab(typeOfLab string, labId string, userId string, requestIsWithSecret bool) (LabType, error)
	GetProtectedLabVersions(typeOfLab string, labId string) ([]LabType, error)
	UpsertProtectedLab(lab LabType, userId string) (LabType, error)
	DeleteProtectedLab(typeOfLab string, labId string) error

	// Shared functions
	GetLabs(typeOfLab string) ([]LabType, error)
	GetLabVersions(typeOfLab string, labId string) ([]LabType, error)
	UpsertLab(LabType) (LabType, error)
	DeleteLab(typeOfLab string, labId string) error

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
