package middleware

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type CORSMiddlewareBuilder struct {
	AllowOrigins     []string      // 允许的来源，* 表示所有来源
	AllowMethods     []string      // 允许的方法，* 表示所有方法
	AllowHeaders     []string      // 允许的请求头，* 表示所有请求头
	ExposeHeaders    []string      // 暴露的响应头，* 表示所有响应头
	AllowCredentials bool          // 是否允许携带凭证（如 Cookies）
	MaxAge           time.Duration // 预检请求的缓存时间
}

func NewCORSMiddlewareBuilder(allowOrigins, allowMethods, allowHeaders, exposeHeaders []string, allowCredentials bool, maxAge time.Duration) *CORSMiddlewareBuilder {
	return &CORSMiddlewareBuilder{
		AllowOrigins:     allowOrigins,
		AllowMethods:     allowMethods,
		AllowHeaders:     allowHeaders,
		ExposeHeaders:    exposeHeaders,
		AllowCredentials: allowCredentials,
		MaxAge:           maxAge,
	}
}

func (m *CORSMiddlewareBuilder) Build() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     m.AllowOrigins,
		AllowMethods:     m.AllowMethods,
		AllowHeaders:     m.AllowHeaders,
		ExposeHeaders:    m.ExposeHeaders,
		AllowCredentials: m.AllowCredentials,
		MaxAge:           m.MaxAge,
	})
}
