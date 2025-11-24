package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/to404hanga/online_judge_controller/constants"
	ojjwt "github.com/to404hanga/online_judge_controller/web/jwt"
	"github.com/to404hanga/pkg404/logger"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
	"gorm.io/gorm"
)

type JWTMiddlewareBuilder struct {
	ojjwt.Handler
	db                   *gorm.DB
	log                  loggerv2.Logger
	checkCompetitionPath []string
}

func NewJWTMiddlewareBuilder(handler ojjwt.Handler, db *gorm.DB, log loggerv2.Logger, checkCompetitionPath []string) *JWTMiddlewareBuilder {
	return &JWTMiddlewareBuilder{
		Handler:              handler,
		db:                   db,
		log:                  log,
		checkCompetitionPath: checkCompetitionPath,
	}
}

// CheckCompetition 检查比赛状态
func (m *JWTMiddlewareBuilder) CheckCompetition() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		path := ctx.Request.URL.Path
		flag := false
		for _, p := range m.checkCompetitionPath {
			if strings.HasPrefix(path, p) {
				flag = true
				break
			}
		}
		if !flag {
			ctx.Next()
			return
		}

		var uc ojjwt.CompetitionUserClaims
		token, err := jwt.ParseWithClaims(m.ExtractToken(ctx), &uc, func(t *jwt.Token) (any, error) {
			return m.JwtKey(), nil
		})
		if err != nil || token == nil || !token.Valid {
			// 仅当 GetProblem 接口无 token 或者解析失败时放行, 有 token 但非法不放行
			if strings.HasPrefix(path, constants.GetProblemPath) && (err != nil || token == nil) {
				ctx.Next()
				return
			}
			m.log.ErrorContext(ctx, "CheckCompetition failed",
				logger.Error(err),
				logger.Bool("token==nil", token == nil),
			)
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		if err = m.CheckSession(ctx, uc.Ssid); err != nil {
			m.log.ErrorContext(ctx, "CheckCompetition failed", logger.Error(err))
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		ctx.Set(constants.ContextUserClaimsKey, uc)
		ctx.Next()
	}
}
