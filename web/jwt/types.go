package jwt

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type Handler interface {
	ExtractToken(ctx *gin.Context) string
	SetCompetitionToken(ctx *gin.Context, competitionId, userId uint64) error
	SetJWTToken(ctx *gin.Context, competitionId, userId uint64, ssid string) error
	CheckSession(ctx *gin.Context, ssid string) error

	JwtKey() []byte
	GetUserClaims(ctx *gin.Context) (*CompetitionClaims, error)
}

type CompetitionClaims struct {
	jwt.RegisteredClaims
	UserId        uint64
	CompetitionID uint64
	Ssid          string
	UserAgent     string
}
