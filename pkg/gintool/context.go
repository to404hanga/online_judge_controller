package gintool

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/to404hanga/online_judge_controller/constants"
	"github.com/to404hanga/online_judge_controller/model"
	"github.com/to404hanga/pkg404/logger"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
)

// GinContextToLoggerContext 将 Gin 上下文转换为 Logger 上下文
func GinContextToLoggerContext(c *gin.Context) context.Context {
	baseCtx := c.Request.Context()

	fields := make([]logger.Field, 0, 2)

	if requestID := c.GetHeader(constants.HeaderRequestIDKey); requestID != "" {
		fields = append(fields, logger.String("RequestID", requestID))
	}
	if userID := c.GetHeader(constants.HeaderUserIDKey); userID != "" {
		fields = append(fields, logger.String("UserID", userID))
	}

	return context.WithValue(baseCtx, loggerv2.FieldsKey, fields)
}

// ExtractOperator 从 Gin 上下文提取操作人 ID
func ExtractOperator(c *gin.Context, p model.CommonParamInterface) error {
	userID := c.GetHeader(constants.HeaderUserIDKey)
	if userID == "" {
		GinResponse(c, &Response{
			Code:    http.StatusBadRequest,
			Message: "X-User-ID header is required",
		})
		return fmt.Errorf("X-User-ID header is required")
	}
	operator, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		GinResponse(c, &Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("X-User-ID header is not a valid uint64: %s", err.Error()),
		})
		return fmt.Errorf("X-User-ID header is not a valid uint64, X-User-ID: %s, err: %w", userID, err)
	}
	p.SetOperator(operator)
	return nil
}
