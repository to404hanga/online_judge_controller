package ioc

import (
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"github.com/to404hanga/online_judge_controller/service"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
	"gorm.io/gorm"
)

func InitRankingService(db *gorm.DB, redis redis.Cmdable, logger loggerv2.Logger) service.RankingService {
	exportDir := viper.GetString("exporter.dir")
	return service.NewRankingService(db, redis, logger, exportDir)
}
