package cleaner

import (
	"context"
	"strings"
	"time"

	"github.com/to404hanga/online_judge_controller/pkg/minio"
	"github.com/to404hanga/online_judge_controller/service"
	"github.com/to404hanga/pkg404/logger"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
)

const (
	ProblemKey  = "problem"
	TestcaseKey = "testcase"
)

// MinIOCleaner 清理 minio 中的相关文件
type MinIOCleaner struct {
	problemSvc          service.ProblemService
	minioSvc            *minio.MinIOService
	log                 loggerv2.Logger
	bucket              string
	orphanFileCheckDays int
}

func NewMinIOCleaner(problemSvc service.ProblemService, minioSvc *minio.MinIOService, log loggerv2.Logger, bucket string, orphanFileCheckDays int) *MinIOCleaner {
	return &MinIOCleaner{
		problemSvc:          problemSvc,
		minioSvc:            minioSvc,
		log:                 log,
		bucket:              bucket,
		orphanFileCheckDays: orphanFileCheckDays,
	}
}

func (c *MinIOCleaner) RunCleanup(ctx context.Context) error {
	c.log.InfoContext(ctx, "Starting minio problem cleanup job")

	orphanStats, err := c.cleanupOrphanFiles(ctx)
	if err != nil {
		c.log.ErrorContext(ctx, "cleanupOrphanFiles failed", logger.Error(err))
		return err
	}

	c.log.InfoContext(ctx, "MinIO problem cleanup job completed", logger.Any("stats", orphanStats))
	return err
}

type CleanupStats struct {
	TotalFiles      int           `json:"total_files"`
	DeletedFiles    int           `json:"deleted_files"`
	DeletedSize     int64         `json:"deleted_size"`
	ErrorCount      int           `json:"error_count"`
	ProcessDuration time.Duration `json:"process_duration"`
	StartTime       time.Time     `json:"start_time"`
	EndTime         time.Time     `json:"end_time"`
}

func (c *MinIOCleaner) cleanupOrphanFiles(ctx context.Context) (stats *CleanupStats, err error) {
	stats = &CleanupStats{
		StartTime: time.Now(),
	}
	defer func() {
		stats.EndTime = time.Now()
		stats.ProcessDuration = stats.EndTime.Sub(stats.StartTime)
	}()

	cutoffTime := time.Now().AddDate(0, 0, -c.orphanFileCheckDays)

	// 列出存储桶中所有对象
	infos, err := c.minioSvc.ListObjectsWithDetails(ctx, c.bucket)
	if err != nil {
		c.log.ErrorContext(ctx, "ListObjectsWithDetails failed", logger.Error(err))
		stats.ErrorCount++
		return
	}

	for _, obj := range infos {
		// 跳过临时文件和系统文件
		if isTempFile(obj.Key) || isSystemFile(obj.Key) {
			continue
		}

		// 只检查超过指定天数的文件
		if obj.LastModified.After(cutoffTime) {
			continue
		}

		if exist, err := c.problemSvc.CheckExistByTestcaseZipURL(ctx, obj.Key); err != nil {
			c.log.ErrorContext(ctx, "CheckExist failed",
				logger.Error(err),
				logger.String("object_key", obj.Key),
				logger.String("bucket", c.bucket),
			)
			stats.ErrorCount++
			continue
		} else if exist {
			continue
		}

		c.log.InfoContext(ctx, "Object not exist",
			logger.String("object_key", obj.Key),
			logger.String("bucket", c.bucket),
		)

		if err = c.minioSvc.DeleteObject(ctx, c.bucket, obj.Key); err != nil {
			c.log.ErrorContext(ctx, "DeleteObject failed",
				logger.Error(err),
				logger.String("object_key", obj.Key),
				logger.String("bucket", c.bucket),
			)
			stats.ErrorCount++
			continue
		}
		stats.DeletedFiles++
		stats.DeletedSize += obj.Size
	}
	return stats, nil
}

// isTempFile 判断文件是否为临时文件
func isTempFile(objectKey string) bool {
	lowerKey := strings.ToLower(objectKey)
	return strings.Contains(lowerKey, "temp/") ||
		strings.Contains(lowerKey, "tmp/") ||
		strings.HasSuffix(lowerKey, ".tmp") ||
		strings.HasSuffix(lowerKey, ".temp")
}

// isSystemFile 判断文件是否为系统文件
func isSystemFile(objectKey string) bool {
	lowerKey := strings.ToLower(objectKey)
	return strings.HasPrefix(lowerKey, "system/") ||
		strings.HasPrefix(lowerKey, ".") ||
		strings.Contains(lowerKey, "/.") ||
		strings.Contains(lowerKey, "config/")
}
