package ioc

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/to404hanga/online_judge_controller/config"
	"github.com/to404hanga/online_judge_controller/web"
	"github.com/to404hanga/online_judge_controller/web/jwt"
	"github.com/to404hanga/online_judge_controller/web/middleware"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
	"gorm.io/gorm"
)

func InitGinServer(l loggerv2.Logger, jwtHandler jwt.Handler, db *gorm.DB, competitionHandler *web.CompetitionHandler, problemHandler *web.ProblemHandler, submissionHandler *web.SubmissionHandler) *web.GinServer {
	var cfg config.GinConfig
	err := viper.UnmarshalKey(cfg.Key(), &cfg)
	if err != nil {
		log.Panicf("unmarshal gin config failed, err: %v", err)
	}

	corsBuilder := middleware.NewCORSMiddlewareBuilder(
		cfg.AllowOrigins,
		cfg.AllowMethods,
		cfg.AllowHeaders,
		cfg.ExposeHeaders,
		cfg.AllowCredentials,
		time.Duration(cfg.MaxAge)*time.Second)
	jwtBuilder := middleware.NewJWTMiddlewareBuilder(jwtHandler, db, l, cfg.CheckCompetitionPath)

	engine := gin.Default()
	engine.Use(
		corsBuilder.Build(),
		jwtBuilder.CheckCompetition(),
	)

	competitionHandler.Register(engine)
	problemHandler.Register(engine)
	submissionHandler.Register(engine)

	return &web.GinServer{
		Engine: engine,
		Addr:   cfg.Addr,
	}
}
