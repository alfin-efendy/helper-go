package storage

import (
	"context"
	"io"
	"mime/multipart"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/logger"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

var minioClient *minio.Client

func initMinino() {
	config := config.Config.Storage

	if config == nil && config.Driver != "minio" {
		return
	}

	var err error
	ctx := context.Background()

	minioClient, err = minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKey, config.SecretKey, ""),
		Secure: config.UseSSL,
	})
	if err != nil {
		logger.Fatal(ctx, err)
	}

	// Create bucket if it doesn't exist
	exists, err := minioClient.BucketExists(ctx, config.BucketName)
	if err != nil {
		logger.Fatal(ctx, err)
	}

	if !exists {
		err = minioClient.MakeBucket(ctx, config.BucketName, minio.MakeBucketOptions{})
		if err != nil {
			logger.Fatal(ctx, err)
		}

		logger.Info(ctx, "Bucket %s created successfully", config.BucketName)
	}

	// Set lifecycle policy for the bucket
	lifecycleConfig := lifecycle.NewConfiguration()

	lifecycleConfig.Rules = []lifecycle.Rule{
		{
			ID:     "expire-rule",
			Status: "Enabled",
			RuleFilter: lifecycle.Filter{
				And: lifecycle.And{
					Tags: []lifecycle.Tag{
						{
							Key:   "type",
							Value: "temporary",
						},
					},
				},
			},
			Expiration: lifecycle.Expiration{
				Days: lifecycle.ExpirationDays(config.RetentionDays),
			},
		},
	}

	err = minioClient.SetBucketLifecycle(ctx, config.BucketName, lifecycleConfig)
	if err != nil {
		logger.Fatal(ctx, err)
	}
}

func UploadFileToMinio(ctx context.Context, file *multipart.FileHeader, path string, isTemporary bool) (minio.UploadInfo, error) {
	if minioClient == nil {
		initMinino()
	}

	var info minio.UploadInfo

	src, err := file.Open()
	if err != nil {
		return info, err
	}
	defer src.Close()

	objectName := path + "/" + file.Filename
	objectOptions := minio.PutObjectOptions{
		ContentType: file.Header.Get("Content-Type"),
	}

	if isTemporary {
		objectOptions.UserMetadata = map[string]string{
			"type": "temporary",
		}
	}

	// Upload the file to the bucket
	info, err = minioClient.PutObject(
		ctx,
		config.Config.Storage.BucketName,
		objectName,
		src,
		file.Size,
		objectOptions,
	)
	return info, err
}

func DownloadFileFromMinio(ctx context.Context, objectName string) (minio.ObjectInfo, []byte, error) {
	if minioClient == nil {
		initMinino()
	}

	info, err := minioClient.StatObject(ctx, config.Config.Storage.BucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return info, nil, err
	}

	object, err := minioClient.GetObject(ctx, config.Config.Storage.BucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return info, nil, err
	}
	defer object.Close()

	data, err := io.ReadAll(object)
	return info, data, err
}
