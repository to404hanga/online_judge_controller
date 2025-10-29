package gintool

import (
	"github.com/gin-gonic/gin"
)

// ContextMiddleware 上下文中间件
func ContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Clone(GinContextToLoggerContext(c))
	}
}
