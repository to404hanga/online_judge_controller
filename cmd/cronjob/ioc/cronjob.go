package ioc

import (
	"github.com/to404hanga/online_judge_controller/job"
	"github.com/to404hanga/online_judge_controller/service"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
)

func InitScheduler(l loggerv2.Logger, problemSvc service.ProblemService, submissionSvc service.SubmissionService) *job.CronScheduler {
	scheduler := job.NewCronScheduler(l)

	// scheduler.AddJob(InitMinIOCleaner(problemSvc, minioSvc, l))
	scheduler.AddJob(InitSubmissionCleaner(submissionSvc, l))

	return scheduler
}
