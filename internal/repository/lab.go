package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"actlabs-hub/internal/auth"
	"actlabs-hub/internal/config"
	"actlabs-hub/internal/entity"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

type labRepository struct {
	// https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#DefaultAzureCredential
	auth      *auth.Auth
	appConfig *config.Config
	rdb       *redis.Client
}

func NewLabRepository(
	auth *auth.Auth,
	appConfig *config.Config,
	rdb *redis.Client,
) (entity.LabRepository, error) {
	return &labRepository{
		auth:      auth,
		appConfig: appConfig,
		rdb:       rdb,
	}, nil
}

const ReproProjectPrefix = "repro-project-"

func (l *labRepository) ListBlobs(
	ctx context.Context,
	labType string,
) ([]entity.Blob, error) {

	slog.Debug("listing labs",
		slog.String("labType", labType),
	)

	// check if the list of the labs exist in redis
	// if they do, return them
	blobsStr, err := l.rdb.Get(ctx, "blobs-"+labType).Result()
	if err == nil {
		var blobs []entity.Blob
		if err := json.Unmarshal([]byte(blobsStr), &blobs); err != nil {
			return nil, err
		}
		slog.Debug("list of blobs found in redis",
			slog.String("labType", labType),
		)
		return blobs, nil
	}

	// URL of the container to list the blobs
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", l.appConfig.ActlabsHubStorageAccount)
	client, err := azblob.NewClient(serviceURL, l.auth.Cred, nil)
	if err != nil {
		return nil, err
	}

	pager := client.NewListBlobsFlatPager(ReproProjectPrefix+labType, &azblob.ListBlobsFlatOptions{
		Include: container.ListBlobsInclude{
			Versions: false,
		},
	})

	var blobs []entity.Blob
	for pager.More() {
		resp, err := pager.NextPage(context.Background())
		if err != nil {
			return nil, err
		}
		for _, blob := range resp.Segment.BlobItems {
			slog.Debug("blob",
				slog.String("name", *blob.Name),
				slog.String("versionId", *blob.VersionID),
				slog.Bool("isCurrentVersion", true),
			)
			blobs = append(blobs, entity.Blob{
				Name:             *blob.Name,
				VersionId:        *blob.VersionID,
				IsCurrentVersion: true,
			})
		}
	}

	// save the blobs in redis
	blobsBytes, err := json.Marshal(blobs)
	if err != nil {
		slog.Debug("not able to marshal blobs", slog.String("error", err.Error()))
		return blobs, nil
	}
	blobsStr = string(blobsBytes)
	if err := l.rdb.Set(ctx, "blobs-"+labType, blobsStr, 0).Err(); err != nil {
		slog.Debug("not able to set blobs in redis", slog.String("error", err.Error()))
		return blobs, nil
	}

	return blobs, nil
}

func (l *labRepository) GetLabWithVersions(ctx context.Context, typeOfLab string, labId string) ([]entity.LabType, error) {
	labs := []entity.LabType{}

	// check if the lab exists in redis
	labStr, err := l.rdb.Get(ctx, redisKey("labWithVersions", typeOfLab, appendDotJson(labId))).Result()
	if err == nil {
		if err := json.Unmarshal([]byte(labStr), &labs); err == nil {
			return labs, nil
		}
		slog.Debug("not able to unmarshal lab found in redis", slog.String("error", err.Error()))
		slog.Debug("getting lab from storage account")
	}

	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", l.appConfig.ActlabsHubStorageAccount)

	client, err := azblob.NewClient(serviceURL, l.auth.Cred, nil)
	if err != nil {
		return labs, err
	}

	containerURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s", l.appConfig.ActlabsHubStorageAccount, ReproProjectPrefix+typeOfLab)
	containerClient, err := container.NewClient(containerURL, l.auth.Cred, nil)
	if err != nil {
		return labs, err
	}

	blobClient := containerClient.NewBlobClient(appendDotJson(labId))

	pager := client.NewListBlobsFlatPager(ReproProjectPrefix+typeOfLab, &azblob.ListBlobsFlatOptions{
		Include: container.ListBlobsInclude{
			Versions: true, // include all versions of each blob
		},
	})

	for pager.More() {
		resp, err := pager.NextPage(ctx)
		if err != nil {
			return labs, err
		}
		for _, blob := range resp.Segment.BlobItems {
			if *blob.Name == appendDotJson(labId) {
				blobClientWithVersionId, err := blobClient.WithVersionID(*blob.VersionID)
				if err != nil {
					slog.Debug("not able to get blob client with version id",
						slog.String("labId", labId),
						slog.String("versionId", *blob.VersionID),
						slog.String("error", err.Error()),
					)
					return labs, err
				}

				downloadResponse, err := blobClientWithVersionId.DownloadStream(ctx, nil)
				if err != nil {
					slog.Debug("not able to download blob with version id",
						slog.String("labId", labId),
						slog.String("versionId", *blob.VersionID),
						slog.String("error", err.Error()),
					)
					return labs, err
				}

				actualBlobData, err := io.ReadAll(downloadResponse.Body)
				if err != nil {
					slog.Debug("not able to read blob data",
						slog.String("labId", labId),
						slog.String("versionId", *blob.VersionID),
						slog.String("error", err.Error()),
					)
					return labs, err
				}

				slog.Debug("blob data : " + string(actualBlobData))

				var lab entity.LabType

				if err := json.Unmarshal(actualBlobData, &lab); err != nil {
					slog.Debug("not able to unmarshal lab",
						slog.String("labId", labId),
						slog.String("versionId", *blob.VersionID),
						slog.String("error", err.Error()),
					)
					return labs, err
				}

				lab.VersionId = *blob.VersionID
				lab.IsCurrentVersion = false
				if blob.IsCurrentVersion != nil {
					lab.IsCurrentVersion = *blob.IsCurrentVersion
				}

				labs = append(labs, lab)
			}
		}
	}

	if len(labs) == 0 {
		return labs, fmt.Errorf("lab not found")
	}

	// add labs to redis
	labsBytes, err := json.Marshal(labs)
	if err != nil {
		slog.Debug("not able to marshal labs", slog.String("error", err.Error()))
		return labs, nil
	}

	if err := l.rdb.Set(ctx, redisKey("labWithVersions", typeOfLab, appendDotJson(labId)), string(labsBytes), 0).Err(); err != nil {
		slog.Debug("not able to set lab with versions in redis",
			slog.String("error", err.Error()),
			slog.String("labId", labId),
		)
	}

	return labs, nil
}

func (l *labRepository) GetLab(
	ctx context.Context,
	typeOfLab string,
	labId string,
) (entity.LabType, error) {
	lab := entity.LabType{}

	// check if the lab exists in redis
	labStr, err := l.rdb.Get(ctx, redisKey("lab", typeOfLab, appendDotJson(labId))).Result()
	if err == nil {
		if err := json.Unmarshal([]byte(labStr), &lab); err == nil {
			return lab, nil
		}
		slog.Debug("not able to unmarshal lab found in redis", slog.String("error", err.Error()))
		slog.Debug("getting lab from storage account")
	}

	containerURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s", l.appConfig.ActlabsHubStorageAccount, ReproProjectPrefix+typeOfLab)
	containerClient, err := container.NewClient(containerURL, l.auth.Cred, nil)
	if err != nil {
		return lab, err
	}

	blobClient := containerClient.NewBlobClient(appendDotJson(labId))

	downloadResponse, err := blobClient.DownloadStream(ctx, nil)
	if err != nil {
		return lab, err
	}

	actualBlobData, err := io.ReadAll(downloadResponse.Body)
	if err != nil {
		return lab, err
	}

	// save the lab in redis
	if err := l.rdb.Set(ctx, redisKey("lab", typeOfLab, appendDotJson(labId)), string(actualBlobData), 0).Err(); err != nil {
		slog.Debug("not able to set lab in redis", slog.String("error", err.Error()))
	}

	if err := json.Unmarshal(actualBlobData, &lab); err != nil {
		return lab, err
	}

	return lab, nil

}

func (l *labRepository) UpsertLab(ctx context.Context, labId string, lab string, typeOfLab string) error {
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", l.appConfig.ActlabsHubStorageAccount)
	client, err := azblob.NewClient(serviceURL, l.auth.Cred, nil)
	if err != nil {
		return err
	}

	containerName := ReproProjectPrefix + typeOfLab
	blobName := appendDotJson(labId)
	blobData := []byte(lab) // convert string to []byte
	blobContentReader := bytes.NewReader(blobData)

	slog.Debug("uploading lab to storage account",
		slog.String("containerName", containerName),
		slog.String("blobName", blobName),
	)

	_, err = client.UploadStream(ctx, containerName, blobName, blobContentReader, nil)
	if err != nil {
		return err
	}

	// since we just uploaded a new version of the lab, we need to delete the cache
	l.rdb.Del(ctx, "blobs-"+typeOfLab)
	l.rdb.Del(ctx, redisKey("labWithVersions", typeOfLab, appendDotJson(labId)))
	l.rdb.Del(ctx, redisKey("lab", typeOfLab, appendDotJson(labId)))

	return err
}

func (l *labRepository) DeleteLab(ctx context.Context, typeOfLab string, labId string) error {
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", l.appConfig.ActlabsHubStorageAccount)

	client, err := azblob.NewClient(serviceURL, l.auth.Cred, nil)
	if err != nil {
		return err
	}

	_, err = client.DeleteBlob(ctx, ReproProjectPrefix+typeOfLab, appendDotJson(labId), nil)
	if err != nil {
		return err
	}

	// since we just uploaded a new version of the lab, we need to delete the cache
	l.rdb.Del(ctx, "blobs-"+typeOfLab)
	l.rdb.Del(ctx, redisKey("labWithVersions", typeOfLab, appendDotJson(labId)))
	l.rdb.Del(ctx, redisKey("lab", typeOfLab, appendDotJson(labId)))

	return nil
}

func appendDotJson(labId string) string {
	if labId[len(labId)-5:] == ".json" {
		return labId
	}
	return labId + ".json"
}

func redisKey(funcIdentifier string, typeOfLab string, labId string) string {
	return funcIdentifier + "-" + typeOfLab + "-" + labId
}
