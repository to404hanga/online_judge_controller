package ioc

import (
	"log"

	"github.com/spf13/viper"
	"github.com/to404hanga/online_judge_controller/config"
	"github.com/to404hanga/online_judge_controller/service"
	"github.com/to404hanga/online_judge_controller/web"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
)

func InitSubmissionHandler(submissionSvc service.SubmissionService, competitionSvc service.CompetitionService, l loggerv2.Logger) *web.SubmissionHandler {
	var cfg config.SubmissionMinIOConfig
	if err := viper.UnmarshalKey(cfg.Key(), &cfg); err != nil {
		log.Panicf("unmarshal problem minio config failed: %v", err)
	}
	return web.NewSubmissionHandler(submissionSvc, competitionSvc, l,
		cfg.Bucket,
		cfg.UploadDurationSeconds,
		cfg.DownloadDurationSeconds)
}
