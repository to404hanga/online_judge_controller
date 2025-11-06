package ioc

import (
	"log"
	"time"

	"github.com/spf13/viper"
	"github.com/to404hanga/online_judge_controller/cmd/cronjob/config"
	"github.com/to404hanga/online_judge_controller/job"
	"github.com/to404hanga/online_judge_controller/job/cleaner"
	"github.com/to404hanga/online_judge_controller/service"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
)

func InitSubmissionCleaner(submissionSvc service.SubmissionService, l loggerv2.Logger) *job.JobConfig {
	var cfg config.SubmissionCleanerConfig
	err := viper.UnmarshalKey(cfg.Key(), &cfg)
	if err != nil {
		log.Panicf("unmarshal submission cleaner config fail, err: %v", err)
	}

	m := cleaner.NewSubmissionCleaner(submissionSvc, l, time.Duration(cfg.TimeRange)*24*time.Hour)
	jbCfg := &job.JobConfig{
		Name:        "提交记录清理",
		CronExpr:    cfg.CronExpr,
		JobFunc:     m.RunCleanup,
		Description: "清理失败的提交记录",
		Enabled:     cfg.Enabled,
		Timeout:     time.Duration(cfg.Timeout) * time.Millisecond,
	}
	return jbCfg
}
