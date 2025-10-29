package minio

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/to404hanga/pkg404/logger"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
)

type MinIOService struct {
	client   *minio.Client
	log      loggerv2.Logger
	endpoint string
	useSSL   bool
}

func NewMinIOSTSService(log loggerv2.Logger, endpoint string, useSSL bool) *MinIOService {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(os.Getenv(EnvMinIOAccessKeyID), os.Getenv(EnvMinIOSecretAccessKey), ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Error("Failed to create MinIO client", logger.Error(err))
		return nil
	}

	return &MinIOService{
		client:   client,
		log:      log,
		endpoint: endpoint,
		useSSL:   useSSL,
	}
}

// GetPresignedUploadURL 获取预签名上传URL
func (s *MinIOService) GetPresignedUploadURL(ctx context.Context, bucketName, objectKey string, durationSeconds int) (string, error) {
	expiration := time.Duration(durationSeconds) * time.Second

	presignedURL, err := s.client.PresignedPutObject(ctx, bucketName, objectKey, expiration)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload URL: %w", err)
	}

	return presignedURL.String(), nil
}

// GetPresignedDownloadURL 获取预签名下载URL
func (s *MinIOService) GetPresignedDownloadURL(ctx context.Context, bucketName, objectKey string, durationSeconds int) (string, error) {
	expiration := time.Duration(durationSeconds) * time.Second

	presignedURL, err := s.client.PresignedGetObject(ctx, bucketName, objectKey, expiration, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned download URL: %w", err)
	}

	return presignedURL.String(), nil
}

// ObjectInfo 对象信息结构体
type ObjectInfo struct {
	Key          string    `json:"key"`          // 对象键
	Size         int64     `json:"size"`         // 文件大小
	LastModified time.Time `json:"lastModified"` // 最后修改时间
	ETag         string    `json:"etag"`         // ETag
	ContentType  string    `json:"contentType"`  // 内容类型
}

// ListAllObjects 获取指定bucket下所有对象的objectKey
func (s *MinIOService) ListAllObjects(ctx context.Context, bucketName string) ([]string, error) {
	var objectKeys []string

	// 使用ListObjects方法列出所有对象
	objectCh := s.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Recursive: true, // 递归列出所有子目录中的对象
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", object.Err)
		}
		objectKeys = append(objectKeys, object.Key)
	}

	return objectKeys, nil
}

// ListObjectsWithPrefix 获取指定前缀的对象列表
func (s *MinIOService) ListObjectsWithPrefix(ctx context.Context, bucketName, prefix string) ([]string, error) {
	var objectKeys []string

	// 使用指定前缀列出对象
	objectCh := s.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects with prefix: %w", object.Err)
		}
		objectKeys = append(objectKeys, object.Key)
	}

	return objectKeys, nil
}

// ListObjectsWithDetails 获取指定bucket下所有对象的详细信息
func (s *MinIOService) ListObjectsWithDetails(ctx context.Context, bucketName string) ([]ObjectInfo, error) {
	var objects []ObjectInfo

	// 使用ListObjects方法列出所有对象
	objectCh := s.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", object.Err)
		}

		objects = append(objects, ObjectInfo{
			Key:          object.Key,
			Size:         object.Size,
			LastModified: object.LastModified,
			ETag:         object.ETag,
			ContentType:  object.ContentType,
		})
	}

	return objects, nil
}

// ListObjectsWithPagination 分页获取对象列表
func (s *MinIOService) ListObjectsWithPagination(ctx context.Context, bucketName string, maxKeys int, startAfter string) ([]ObjectInfo, string, error) {
	var objects []ObjectInfo
	var nextMarker string

	// 使用ListObjects方法分页列出对象
	objectCh := s.client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{
		Recursive:  true,
		MaxKeys:    maxKeys,
		StartAfter: startAfter,
	})

	count := 0
	for object := range objectCh {
		if object.Err != nil {
			return nil, "", fmt.Errorf("failed to list objects: %w", object.Err)
		}

		objects = append(objects, ObjectInfo{
			Key:          object.Key,
			Size:         object.Size,
			LastModified: object.LastModified,
			ETag:         object.ETag,
			ContentType:  object.ContentType,
		})

		nextMarker = object.Key
		count++

		// 如果达到最大数量，停止
		if count >= maxKeys {
			break
		}
	}

	return objects, nextMarker, nil
}

// DeleteObject 删除指定的对象
func (s *MinIOService) DeleteObject(ctx context.Context, bucketName, objectKey string) error {
	err := s.client.RemoveObject(ctx, bucketName, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to delete object",
			logger.Error(err),
			logger.String("bucketName", bucketName),
			logger.String("objectKey", objectKey),
		)
		return fmt.Errorf("failed to delete object %s: %w", objectKey, err)
	}

	s.log.InfoContext(ctx, "Successfully deleted object",
		logger.String("bucketName", bucketName),
		logger.String("objectKey", objectKey),
	)

	return nil
}

// DeleteObjects 批量删除对象
func (s *MinIOService) DeleteObjects(ctx context.Context, bucketName string, objectKeys []string) error {
	if len(objectKeys) == 0 {
		return nil
	}

	// 创建删除对象的通道
	objectsCh := make(chan minio.ObjectInfo, len(objectKeys))

	// 发送要删除的对象
	go func() {
		defer close(objectsCh)
		for _, key := range objectKeys {
			objectsCh <- minio.ObjectInfo{Key: key}
		}
	}()

	// 执行批量删除
	errorCh := s.client.RemoveObjects(ctx, bucketName, objectsCh, minio.RemoveObjectsOptions{})

	// 检查删除错误
	var errors []error
	deletedCount := 0

	for deleteError := range errorCh {
		if deleteError.Err != nil {
			errors = append(errors, fmt.Errorf("failed to delete object %s: %w", deleteError.ObjectName, deleteError.Err))
			s.log.ErrorContext(ctx, "Failed to delete object in batch",
				logger.Error(deleteError.Err),
				logger.String("objectName", deleteError.ObjectName),
			)
		} else {
			deletedCount++
		}
	}

	s.log.InfoContext(ctx, "Batch delete completed",
		logger.String("bucketName", bucketName),
		logger.Int("totalObjects", len(objectKeys)),
		logger.Int("deletedCount", deletedCount),
		logger.Int("errorCount", len(errors)),
	)

	if len(errors) > 0 {
		return fmt.Errorf("batch delete failed with %d errors: %v", len(errors), errors)
	}

	return nil
}

// ObjectExists 检查对象是否存在
func (s *MinIOService) ObjectExists(ctx context.Context, bucketName, objectKey string) (bool, error) {
	_, err := s.client.StatObject(ctx, bucketName, objectKey, minio.StatObjectOptions{})
	if err != nil {
		// 检查是否是对象不存在的错误
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		s.log.ErrorContext(ctx, "Failed to check object existence",
			logger.Error(err),
			logger.String("bucketName", bucketName),
			logger.String("objectKey", objectKey),
		)
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}

// GetObjectInfo 获取对象详细信息
func (s *MinIOService) GetObjectInfo(ctx context.Context, bucketName, objectKey string) (*ObjectInfo, error) {
	objInfo, err := s.client.StatObject(ctx, bucketName, objectKey, minio.StatObjectOptions{})
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to get object info",
			logger.Error(err),
			logger.String("bucketName", bucketName),
			logger.String("objectKey", objectKey),
		)
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}

	return &ObjectInfo{
		Key:          objInfo.Key,
		Size:         objInfo.Size,
		LastModified: objInfo.LastModified,
		ETag:         objInfo.ETag,
		ContentType:  objInfo.ContentType,
	}, nil
}
