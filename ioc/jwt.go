package ioc

import (
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"github.com/to404hanga/online_judge_controller/config"
	"github.com/to404hanga/online_judge_controller/web/jwt"
)

func InitJWTHandler(rdb redis.Cmdable) jwt.Handler {
	var cfg config.JWTConfig
	if err := viper.UnmarshalKey(cfg.Key(), &cfg); err != nil {
		log.Panicf("unmarshal jwt config failed: %v", err)
	}

	jwtHandler := jwt.NewRedisJWTHandler(rdb, []byte(cfg.JWTKey), []byte(cfg.RefreshKey), time.Duration(cfg.JWTExpiration)*time.Minute, time.Duration(cfg.RefreshExpiration)*time.Minute)
	return jwtHandler
}
