//go:build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/to404hanga/online_judge_controller/cmd/controller/ioc"
	commonioc "github.com/to404hanga/online_judge_controller/ioc"
	"github.com/to404hanga/online_judge_controller/service"
	"github.com/to404hanga/online_judge_controller/web"
	"github.com/to404hanga/online_judge_controller/event"
)

func BuildDependency() *web.GinServer {
	wire.Build(
		commonioc.InitDB,
		commonioc.InitLogger,
		commonioc.InitJWTHandler,
		commonioc.InitRedis,
		commonioc.InitMinIO,
		commonioc.InitKafka,
		commonioc.InitSyncProducer,

		event.NewSaramaProducer,

		service.NewCompetitionService,
		// service.NewUserService,
		service.NewProblemService,
		service.NewSubmissionService,
		ioc.InitRankingService,

		web.NewCompetitionHandler,
		web.NewHealthHandler,
		commonioc.InitProblemHandler,
		commonioc.InitSubmissionHandler,

		ioc.InitGinServer,
	)
	return &web.GinServer{}
}
