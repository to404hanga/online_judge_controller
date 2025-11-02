package middleware

import (
	"bytes"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/to404hanga/pkg404/logger"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
)

func Logger(log loggerv2.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		query := ctx.Request.URL.Query().Encode()
		path := ctx.Request.URL.Path
		if reqBodyBytes, err := io.ReadAll(ctx.Request.Body); err != nil {
			log.WarnContext(ctx.Request.Context(), "read request body failed",
				logger.Error(err),
				logger.String("path", path),
				logger.String("query_parameters", query),
			)
		} else {
			reqBody := string(reqBodyBytes)
			log.InfoContext(ctx.Request.Context(), "request info",
				logger.String("path", path),
				logger.String("body", reqBody),
				logger.String("query_parameters", query),
			)
			ctx.Request.Body = io.NopCloser(bytes.NewBufferString(reqBody)) // 必须要重新设置，否则后续的 handler 会读取到空体
		}
		ctx.Next()
	}
}
