package ioc

import (
	"log"
	"time"

	"github.com/spf13/viper"
	"github.com/to404hanga/online_judge_controller/cmd/cronjob/config"
	"github.com/to404hanga/online_judge_controller/job"
	"github.com/to404hanga/online_judge_controller/job/cleaner"
	"github.com/to404hanga/online_judge_controller/pkg/minio"
	"github.com/to404hanga/online_judge_controller/service"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
)

func InitMinIOCleaner(problemSvc service.ProblemService, minioSvc *minio.MinIOService, l loggerv2.Logger) *job.JobConfig {
	var cfg config.MinIOCleanerConfig
	err := viper.UnmarshalKey(cfg.Key(), &cfg)
	if err != nil {
		log.Panicf("unmarshal minio cleaner config fail, err: %v", err)
	}

	m := cleaner.NewMinIOCleaner(problemSvc, minioSvc, l, cfg.Bucket, cfg.OrphanFileCheckDays)
	jbCfg := &job.JobConfig{
		Name:        "MinIO 孤儿文件清理",
		CronExpr:    cfg.CronExpr,
		JobFunc:     m.RunCleanup,
		Description: "清理 MinIO 中未被引用的测试用例文件",
		Enabled:     cfg.Enabled,
		Timeout:     time.Duration(cfg.Timeout) * time.Millisecond,
	}
	return jbCfg
}
