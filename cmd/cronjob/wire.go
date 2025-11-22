//go:build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/to404hanga/online_judge_controller/cmd/cronjob/ioc"
	commonioc "github.com/to404hanga/online_judge_controller/ioc"
	"github.com/to404hanga/online_judge_controller/job"
	"github.com/to404hanga/online_judge_controller/service"
)

func InitScheduler() *job.CronScheduler {
	wire.Build(
		commonioc.InitDB,
		commonioc.InitLogger,
		ioc.InitNilRedis,
		ioc.InitNilKafka,
		service.NewProblemService,
		service.NewSubmissionService,
		ioc.InitScheduler,
	)
	return &job.CronScheduler{}
}
