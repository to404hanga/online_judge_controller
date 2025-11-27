//go:build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/to404hanga/online_judge_controller/cmd/controller/ioc"
	"github.com/to404hanga/online_judge_controller/event"
	commonioc "github.com/to404hanga/online_judge_controller/ioc"
	"github.com/to404hanga/online_judge_controller/service"
	"github.com/to404hanga/online_judge_controller/web"
)

func BuildDependency() *web.GinServer {
	wire.Build(
		commonioc.InitDB,
		commonioc.InitLogger,
		commonioc.InitJWTHandler,
		commonioc.InitRedis,
		commonioc.InitKafka,
		commonioc.InitSyncProducer,
		commonioc.InitEtcdClient,

		event.NewSaramaProducer,

		service.NewCompetitionService,
		service.NewUserService,
		service.NewProblemService,
		service.NewSubmissionService,
		ioc.InitRankingService,

		web.NewCompetitionHandler,
		web.NewHealthHandler,
		// commonioc.InitProblemHandler,
		web.NewProblemHandler,
		// commonioc.InitSubmissionHandler,
		web.NewSubmissionHandler,
		web.NewUserHandler,

		ioc.InitGinServer,
	)
	return &web.GinServer{}
}
