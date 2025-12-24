package gintool

import (
	"fmt"
	"net/http"
	"reflect"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/to404hanga/online_judge_controller/constants"
	"github.com/to404hanga/online_judge_controller/model"
	"github.com/to404hanga/online_judge_controller/web/jwt"
	"github.com/to404hanga/pkg404/logger"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
)

// WrapHandler 包装处理函数
func WrapHandler[T model.CommonParamInterface](h func(c *gin.Context, pType T), log loggerv2.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var param T
		// 确保指针类型的 T 不为 nil，避免在 ExtractOperator 中调用 SetOperator 时报空指针
		rv := reflect.ValueOf(param)
		if rv.IsValid() && rv.Kind() == reflect.Ptr && rv.IsNil() {
			param = reflect.New(rv.Type().Elem()).Interface().(T)
		}

		// 1) URI
		if len(c.Params) > 0 {
			m := make(map[string][]string, len(c.Params))
			for _, v := range c.Params {
				m[v.Key] = []string{v.Value}
			}
			if err := binding.Uri.BindUri(m, param); err != nil {
				GinResponse(c, &Response{
					Code:    http.StatusBadRequest,
					Message: err.Error(),
				})
				log.ErrorContext(c.Request.Context(), "WrapHandler bind uri failed", logger.Error(err))
				return
			}
		}

		// 2) Header
		err := binding.Header.Bind(c.Request, param)
		if err != nil {
			GinResponse(c, &Response{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			})
			log.ErrorContext(c.Request.Context(), "WrapHandler bind header failed", logger.Error(err))
			return
		}

		// 3) Query/Form
		if c.Request.URL != nil && c.Request.URL.RawQuery != "" {
			err = binding.Query.Bind(c.Request, param)
			if err != nil {
				GinResponse(c, &Response{
					Code:    http.StatusBadRequest,
					Message: err.Error(),
				})
				log.ErrorContext(c.Request.Context(), "WrapHandler bind query failed", logger.Error(err))
				return
			}
		}

		// 4) JSON
		if c.ContentType() == binding.MIMEJSON {
			jsonBytes, _ := c.GetRawData()
			if len(jsonBytes) > 0 {
				err = json.Unmarshal(jsonBytes, param)
				if err != nil {
					GinResponse(c, &Response{
						Code:    http.StatusBadRequest,
						Message: err.Error(),
					})
					log.ErrorContext(c.Request.Context(), "WrapHandler bind json failed", logger.Error(err))
					return
				}
			}
		}

		err = Validator.Struct(param)
		if err != nil {
			GinResponse(c, &Response{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			})
			log.ErrorContext(c.Request.Context(), "WrapHandler validate failed", logger.Error(err))
			return
		}

		err = ExtractOperator(c, param)
		if err != nil {
			GinResponse(c, &Response{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			})
			log.ErrorContext(c.Request.Context(), "WrapHandler ExtractOperator failed", logger.Error(err))
			return
		}

		h(c, param)
	}
}

// WrapWithoutBodyHandler 包装处理函数
func WrapWithoutBodyHandler[T model.CommonParamInterface](h func(c *gin.Context, pType T), log loggerv2.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var param T

		// 确保指针类型的 T 不为 nil，避免在 ExtractOperator 中调用 SetOperator 时报空指针
		rv := reflect.ValueOf(param)
		if rv.IsValid() && rv.Kind() == reflect.Ptr && rv.IsNil() {
			param = reflect.New(rv.Type().Elem()).Interface().(T)
		}

		err := ExtractOperator(c, param)
		if err != nil {
			GinResponse(c, &Response{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			})
			log.ErrorContext(c.Request.Context(), "WrapHandler ExtractOperator failed", logger.Error(err))
			return
		}

		h(c, param)
	}
}

// WrapCompetitionHandler 包装比赛处理函数
func WrapCompetitionHandler[T model.CompetitionCommonParamInterface](h func(c *gin.Context, pType T), log loggerv2.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var param T
		// 确保指针类型的 T 不为 nil，避免在 ExtractOperator 中调用 SetOperator 时报空指针
		rv := reflect.ValueOf(param)
		if rv.IsValid() && rv.Kind() == reflect.Ptr && rv.IsNil() {
			param = reflect.New(rv.Type().Elem()).Interface().(T)
		}

		// 1) URI
		if len(c.Params) > 0 {
			m := make(map[string][]string, len(c.Params))
			for _, v := range c.Params {
				m[v.Key] = []string{v.Value}
			}
			if err := binding.Uri.BindUri(m, &param); err != nil {
				GinResponse(c, &Response{
					Code:    http.StatusBadRequest,
					Message: err.Error(),
				})
				log.ErrorContext(c.Request.Context(), "WrapCompetitionHandler bind uri failed", logger.Error(err))
				return
			}
		}

		// 2) Header
		err := binding.Header.Bind(c.Request, &param)
		if err != nil {
			GinResponse(c, &Response{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			})
			log.ErrorContext(c.Request.Context(), "WrapCompetitionHandler bind header failed", logger.Error(err))
			return
		}

		// 3) Query/Form
		if c.Request.URL != nil && c.Request.URL.RawQuery != "" {
			err = binding.Query.Bind(c.Request, &param)
			if err != nil {
				GinResponse(c, &Response{
					Code:    http.StatusBadRequest,
					Message: err.Error(),
				})
				log.ErrorContext(c.Request.Context(), "WrapCompetitionHandler bind query failed", logger.Error(err))
				return
			}
		}

		// 4) JSON
		if c.ContentType() == binding.MIMEJSON {
			jsonBytes, _ := c.GetRawData()
			if len(jsonBytes) > 0 {
				err = json.Unmarshal(jsonBytes, param)
				if err != nil {
					GinResponse(c, &Response{
						Code:    http.StatusBadRequest,
						Message: err.Error(),
					})
					log.ErrorContext(c.Request.Context(), "WrapCompetitionHandler bind json failed", logger.Error(err))
					return
				}
			}
		}

		err = Validator.Struct(param)
		if err != nil {
			GinResponse(c, &Response{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			})
			log.ErrorContext(c.Request.Context(), "WrapCompetitionHandler validate failed", logger.Error(err))
			return
		}

		userClaims, exists := c.Get(constants.ContextCompetitionClaimsKey)
		if !exists {
			GinResponse(c, &Response{
				Code:    http.StatusBadRequest,
				Message: "competition user claims not found",
			})
			log.ErrorContext(c.Request.Context(), "WrapCompetitionHandler competition user claims not found")
			return
		}
		competitionClaims, ok := userClaims.(jwt.CompetitionClaims)
		if !ok {
			GinResponse(c, &Response{
				Code:    http.StatusBadRequest,
				Message: "competition user claims type assertion failed",
			})
			log.ErrorContext(c.Request.Context(), "WrapCompetitionHandler competition user claims type assertion failed")
			return
		}

		param.SetOperator(competitionClaims.UserId)
		param.SetCompetitionID(competitionClaims.CompetitionID)

		h(c, param)
	}
}

// WrapCompetitionWithoutBodyHandler 包装比赛处理函数，不绑定JSON体
func WrapCompetitionWithoutBodyHandler[T model.CompetitionCommonParamInterface](h func(c *gin.Context, pType T), log loggerv2.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var param T

		userClaims, exists := c.Get(constants.ContextCompetitionClaimsKey)
		if !exists {
			GinResponse(c, &Response{
				Code:    http.StatusBadRequest,
				Message: "competition user claims not found",
			})
			log.ErrorContext(c.Request.Context(), "WrapCompetitionHandler competition user claims not found")
			return
		}
		competitionClaims, ok := userClaims.(jwt.CompetitionClaims)
		if !ok {
			GinResponse(c, &Response{
				Code:    http.StatusBadRequest,
				Message: "competition user claims type assertion failed",
			})
			log.ErrorContext(c.Request.Context(), "WrapCompetitionHandler competition user claims type assertion failed")
			return
		}

		param.SetOperator(competitionClaims.UserId)
		param.SetCompetitionID(competitionClaims.CompetitionID)

		h(c, param)
	}
}

func WrapCompetitionSSEHandler[T model.CompetitionCommonParamInterface](h func(c *gin.Context, pType T) chan string, log loggerv2.Logger, heartCheckDuration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		var param T

		userClaims, exists := c.Get(constants.ContextCompetitionClaimsKey)
		if !exists {
			GinResponse(c, &Response{
				Code:    http.StatusBadRequest,
				Message: "competition user claims not found",
			})
			log.ErrorContext(c.Request.Context(), "WrapCompetitionHandler competition user claims not found")
			return
		}
		competitionClaims, ok := userClaims.(jwt.CompetitionClaims)
		if !ok {
			GinResponse(c, &Response{
				Code:    http.StatusBadRequest,
				Message: "competition user claims type assertion failed",
			})
			log.ErrorContext(c.Request.Context(), "WrapCompetitionHandler competition user claims type assertion failed")
			return
		}

		param.SetOperator(competitionClaims.UserId)
		param.SetCompetitionID(competitionClaims.CompetitionID)

		// 设置 SSE 必要响应头
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")

		// 用于接收关闭通知
		clientClosed := c.Writer.CloseNotify()

		ticker := time.NewTicker(heartCheckDuration)
		defer ticker.Stop()

		// 调用处理函数，获取事件通道
		eventChan := h(c, param)

		for {
			select {
			case <-clientClosed:
				log.InfoContext(c.Request.Context(), "WrapCompetitionSSEHandler client closed")
				return
			case t := <-ticker.C:
				_, _ = fmt.Fprintf(c.Writer, "data: %v\n\n", t.Format("2006-01-02 15:04:05"))
				c.Writer.Flush()
			case event, ok := <-eventChan:
				if !ok {
					log.InfoContext(c.Request.Context(), "WrapCompetitionSSEHandler event channel closed")
					_, _ = fmt.Fprintf(c.Writer, "data: closed\n\n")
					return
				}
				_, _ = fmt.Fprintf(c.Writer, "data: %v\n\n", event)
				c.Writer.Flush()
			}
		}
	}
}
