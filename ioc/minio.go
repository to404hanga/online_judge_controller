package ioc

import (
	"log"

	"github.com/spf13/viper"
	"github.com/to404hanga/online_judge_controller/config"
	"github.com/to404hanga/online_judge_controller/pkg/minio"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
)

func InitMinIO(l loggerv2.Logger) *minio.MinIOService {
	var cfg config.MinIOConfig
	if err := viper.UnmarshalKey(cfg.Key(), &cfg); err != nil {
		log.Panicf("unmarshal minio config failed: %v", err)
	}
	return minio.NewMinIOService(l, cfg.Endpoint, cfg.UseSSL)
}
