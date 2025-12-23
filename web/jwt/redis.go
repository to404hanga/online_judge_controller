package jwt

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/to404hanga/online_judge_controller/constants"
)

var ssidKey = "users:ssid:%s"

type RedisJWTHandler struct {
	client        redis.Cmdable
	signingMethod jwt.SigningMethod
	jwtExpiration time.Duration
	jwtKey        []byte
}

func NewRedisJWTHandler(client redis.Cmdable, jwtKey []byte, jwtExpiration time.Duration) Handler {
	return &RedisJWTHandler{
		client:        client,
		signingMethod: jwt.SigningMethodHS512,
		jwtExpiration: jwtExpiration,
		jwtKey:        jwtKey,
	}
}

var _ Handler = &RedisJWTHandler{}

func (h *RedisJWTHandler) CheckSession(ctx *gin.Context, ssid string) error {
	cnt, err := h.client.Exists(ctx, fmt.Sprintf(ssidKey, ssid)).Result()
	if err != nil {
		return err
	}
	if cnt > 0 {
		return errors.New("token invalid")
	}
	return nil
}

func (h *RedisJWTHandler) SetCompetitionToken(ctx *gin.Context, competitionId, userId uint64) error {
	ssid := uuid.New().String()
	return h.SetJWTToken(ctx, competitionId, userId, ssid)
}

func (h *RedisJWTHandler) ExtractToken(ctx *gin.Context) string {
	// 优先从 X-Competition-JWT-Token Header 提取 token
	authCode := ctx.GetHeader(constants.HeaderCompetitionTokenKey)
	if authCode != "" {
		return authCode
	}

	// 如果 Header 中没有，尝试从 Cookie 中提取
	tokenFromCookie, err := ctx.Cookie(constants.HeaderCompetitionTokenKey)
	if err != nil || tokenFromCookie == "" {
		ctx.AbortWithStatus(http.StatusForbidden)
		return ""
	}

	return tokenFromCookie
}

func (h *RedisJWTHandler) SetJWTToken(ctx *gin.Context, competitionId, userId uint64, ssid string) error {
	uc := CompetitionClaims{
		CompetitionID: competitionId,
		UserId:        userId,
		Ssid:          ssid,
		UserAgent:     ctx.GetHeader("User-Agent"),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(h.jwtExpiration)),
		},
	}
	token := jwt.NewWithClaims(h.signingMethod, uc)
	tokenStr, err := token.SignedString(h.jwtKey)
	if err != nil {
		return err
	}

	// 设置响应头
	ctx.Header(constants.HeaderCompetitionTokenKey, tokenStr)

	// 同时设置Cookie，支持浏览器自动携带
	ctx.SetCookie(
		constants.HeaderCompetitionTokenKey, // cookie名称
		tokenStr,                            // cookie 值
		int(h.jwtExpiration.Seconds()),      // 过期时间（秒）
		"/",                                 // 路径
		"",                                  // 域名
		false,                               // secure (HTTPS)
		true,                                // httpOnly
	)

	return nil
}

func (h *RedisJWTHandler) JwtKey() []byte {
	return h.jwtKey
}

func (h *RedisJWTHandler) GetUserClaims(ctx *gin.Context) (*CompetitionClaims, error) {
	ucAny, exists := ctx.Get(constants.ContextCompetitionClaimsKey)
	if !exists {
		return nil, fmt.Errorf("user claims not found in context")
	}
	uc, ok := ucAny.(CompetitionClaims)
	if !ok {
		return nil, fmt.Errorf("user claims type assertion error")
	}
	return &uc, nil
}
