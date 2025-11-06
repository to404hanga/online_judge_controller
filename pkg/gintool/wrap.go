package gintool

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
		// 1) URI
		if len(c.Params) > 0 {
			if err := c.ShouldBindUri(&param); err != nil {
				GinResponse(c, &Response{
					Code:    http.StatusBadRequest,
					Message: err.Error(),
				})
				log.ErrorContext(c.Request.Context(), "WrapHandler bind uri failed", logger.Error(err))
				return
			}
		}

		// 2) Header
		err := c.ShouldBindHeader(&param)
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
			err = c.ShouldBindQuery(&param)
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
		err = c.ShouldBindJSON(&param)
		if err != nil {
			GinResponse(c, &Response{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			})
			log.ErrorContext(c.Request.Context(), "WrapHandler bind json failed", logger.Error(err))
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
		// 1) URI
		if len(c.Params) > 0 {
			if err := c.ShouldBindUri(&param); err != nil {
				GinResponse(c, &Response{
					Code:    http.StatusBadRequest,
					Message: err.Error(),
				})
				log.ErrorContext(c.Request.Context(), "WrapCompetitionHandler bind uri failed", logger.Error(err))
				return
			}
		}

		// 2) Header
		err := c.ShouldBindHeader(&param)
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
			err = c.ShouldBindQuery(&param)
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
		err = c.ShouldBindJSON(&param)
		if err != nil {
			GinResponse(c, &Response{
				Code:    http.StatusBadRequest,
				Message: err.Error(),
			})
			log.ErrorContext(c.Request.Context(), "WrapCompetitionHandler bind json failed", logger.Error(err))
			return
		}

		userClaims, exists := c.Get(constants.ContextUserClaimsKey)
		if !exists {
			GinResponse(c, &Response{
				Code:    http.StatusBadRequest,
				Message: "competition user claims not found",
			})
			log.ErrorContext(c.Request.Context(), "WrapCompetitionHandler competition user claims not found")
			return
		}
		competitionUserClaims, ok := userClaims.(jwt.CompetitionUserClaims)
		if !ok {
			GinResponse(c, &Response{
				Code:    http.StatusBadRequest,
				Message: "competition user claims type assertion failed",
			})
			log.ErrorContext(c.Request.Context(), "WrapCompetitionHandler competition user claims type assertion failed")
			return
		}

		param.SetOperator(competitionUserClaims.UserId)
		param.SetCompetitionID(competitionUserClaims.CompetitionID)

		h(c, param)
	}
}

// WrapCompetitionWithoutBodyHandler 包装比赛处理函数，不绑定JSON体
func WrapCompetitionWithoutBodyHandler[T model.CompetitionCommonParamInterface](h func(c *gin.Context, pType T), log loggerv2.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var param T

		userClaims, exists := c.Get(constants.ContextUserClaimsKey)
		if !exists {
			GinResponse(c, &Response{
				Code:    http.StatusBadRequest,
				Message: "competition user claims not found",
			})
			log.ErrorContext(c.Request.Context(), "WrapCompetitionHandler competition user claims not found")
			return
		}
		competitionUserClaims, ok := userClaims.(jwt.CompetitionUserClaims)
		if !ok {
			GinResponse(c, &Response{
				Code:    http.StatusBadRequest,
				Message: "competition user claims type assertion failed",
			})
			log.ErrorContext(c.Request.Context(), "WrapCompetitionHandler competition user claims type assertion failed")
			return
		}

		param.SetOperator(competitionUserClaims.UserId)
		param.SetCompetitionID(competitionUserClaims.CompetitionID)

		h(c, param)
	}
}
