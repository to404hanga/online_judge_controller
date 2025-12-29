package cleaner

import (
	"context"
	"time"

	"github.com/to404hanga/online_judge_controller/service"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
)

type SubmissionCleaner struct {
	submissionSvc service.SubmissionService
	log           loggerv2.Logger
	timeRange     time.Duration
}

// NewSubmissionCleaner 创建新的提交清理器
func NewSubmissionCleaner(submissionSvc service.SubmissionService, log loggerv2.Logger, timeRange time.Duration) *SubmissionCleaner {
	return &SubmissionCleaner{
		submissionSvc: submissionSvc,
		log:           log,
		timeRange:     timeRange,
	}
}

// RunCleanup 运行提交清理任务
func (c *SubmissionCleaner) RunCleanup(ctx context.Context) error {
	c.log.InfoContext(ctx, "Starting submission cleanup job")

	if err := c.submissionSvc.CleanUserFailedSubmission(ctx, time.Now().Add(-c.timeRange)); err != nil {
		return err
	}

	c.log.InfoContext(ctx, "Submission cleanup completed")
	return nil
}
