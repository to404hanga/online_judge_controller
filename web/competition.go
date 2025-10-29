package web

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	ojmodel "github.com/to404hanga/online_judge_common/model"
	"github.com/to404hanga/online_judge_controller/constants"
	"github.com/to404hanga/online_judge_controller/model"
	"github.com/to404hanga/online_judge_controller/pkg/gintool"
	"github.com/to404hanga/online_judge_controller/service"
	"github.com/to404hanga/online_judge_controller/web/jwt"
	"github.com/to404hanga/pkg404/logger"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
)

type CompetitionHandler struct {
	competitionSvc service.CompetitionService
	rankingSvc     service.RankingService
	jwtHandler     jwt.Handler
	log            loggerv2.Logger
}

var _ Handler = (*CompetitionHandler)(nil)

func NewCompetitionHandler(competitionSvc service.CompetitionService, rankingSvc service.RankingService, jwtHandler jwt.Handler, log loggerv2.Logger) *CompetitionHandler {
	return &CompetitionHandler{
		competitionSvc: competitionSvc,
		rankingSvc:     rankingSvc,
		jwtHandler:     jwtHandler,
		log:            log,
	}
}

func (h *CompetitionHandler) Register(r *gin.Engine) {
	r.POST(constants.CreateCompetitionPath, gintool.WrapHandler(h.CreateCompetition, h.log))
	r.PUT(constants.UpdateCompetitionPath, gintool.WrapHandler(h.UpdateCompetition, h.log))
	r.POST(constants.AddCompetitionProblemPath, gintool.WrapHandler(h.AddCompetitionProblem, h.log))
	r.DELETE(constants.RemoveCompetitionProblemPath, gintool.WrapHandler(h.RemoveCompetitionProblem, h.log))
	r.PUT(constants.EnableCompetitionProblemPath, gintool.WrapHandler(h.EnableCompetitionProblem, h.log))
	r.PUT(constants.DisableCompetitionProblemPath, gintool.WrapHandler(h.DisableCompetitionProblem, h.log))
	r.POST(constants.StartCompetitionPath, gintool.WrapHandler(h.StartCompetition, h.log))
	r.GET(constants.GetCompetitionProblemListWithPresignedURLPath, gintool.WrapCompetitionWithoutBodyHandler(h.GetCompetitionProblemListWithPresignedURL, h.log))
	r.GET(constants.GetCompetitionRankingListPath, gintool.WrapCompetitionHandler(h.GetCompetitionRankingList, h.log))
}

func (h *CompetitionHandler) CreateCompetition(c *gin.Context, param *model.CreateCompetitionParam) {
	if !param.EndTime.After(param.StartTime) {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusBadRequest,
			Message: "EndTime must be after StartTime",
		})
		h.log.ErrorContext(c.Request.Context(), "CreateCompetition EndTime must be after StartTime",
			logger.String("start_time", param.StartTime.GoString()),
			logger.String("end_time", param.EndTime.GoString()))
		return
	}

	ctx := c.Request.Context()

	err := h.competitionSvc.CreateCompetition(ctx, param)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("CreateCompetition failed: %s", err.Error()),
		})
		h.log.ErrorContext(ctx, "CreateCompetition failed", logger.Error(err))
		return
	}
	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
	})
}

func (h *CompetitionHandler) UpdateCompetition(c *gin.Context, param *model.UpdateCompetitionParam) {
	ctx := loggerv2.ContextWithFields(c.Request.Context(),
		logger.Uint64("competition_id", param.ID))

	if param.StartTime != nil && param.EndTime != nil {
		if !param.EndTime.After(*param.StartTime) {
			gintool.GinResponse(c, &gintool.Response{
				Code:    http.StatusBadRequest,
				Message: "EndTime must be after StartTime",
			})
			h.log.ErrorContext(ctx, "UpdateCompetition EndTime must be after StartTime",
				logger.String("start_time", param.StartTime.GoString()),
				logger.String("end_time", param.EndTime.GoString()))
			return
		}
	}
	if param.Status != nil {
		if *param.Status != int8(ojmodel.CompetitionStatusUnpublished) &&
			*param.Status != int8(ojmodel.CompetitionStatusPublished) &&
			*param.Status != int8(ojmodel.CompetitionStatusDeleted) {
			gintool.GinResponse(c, &gintool.Response{
				Code:    http.StatusBadRequest,
				Message: "Status must be Unpublished, Published, or Deleted",
			})
			h.log.ErrorContext(ctx, "UpdateCompetition Status must be Unpublished, Published, or Deleted",
				logger.Int8("status", *param.Status))
			return
		}
	}

	err := h.competitionSvc.UpdateCompetition(ctx, param)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("UpdateCompetition failed: %s", err.Error()),
		})
		h.log.ErrorContext(ctx, "UpdateCompetition failed", logger.Error(err))
		return
	}
	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
	})
}

func (h *CompetitionHandler) AddCompetitionProblem(c *gin.Context, param *model.CompetitionProblemParam) {
	ctx := loggerv2.ContextWithFields(c.Request.Context(),
		logger.Uint64("competition_id", param.CompetitionID),
		logger.Slice("problem_ids", param.ProblemIDs))

	err := h.competitionSvc.AddCompetitionProblem(ctx, param)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("AddCompetitionProblem failed: %s", err.Error()),
		})
		h.log.ErrorContext(ctx, "AddCompetitionProblem failed", logger.Error(err))
		return
	}
	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
	})
}

func (h *CompetitionHandler) RemoveCompetitionProblem(c *gin.Context, param *model.CompetitionProblemParam) {
	ctx := loggerv2.ContextWithFields(c.Request.Context(),
		logger.Uint64("competition_id", param.CompetitionID),
		logger.Slice("problem_ids", param.ProblemIDs))

	err := h.competitionSvc.RemoveCompetitionProblem(ctx, param)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("RemoveCompetitionProblem failed: %s", err.Error()),
		})
		h.log.ErrorContext(ctx, "RemoveCompetitionProblem failed", logger.Error(err))
		return
	}
	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
	})
}

func (h *CompetitionHandler) EnableCompetitionProblem(c *gin.Context, param *model.CompetitionProblemParam) {
	ctx := loggerv2.ContextWithFields(c.Request.Context(),
		logger.Uint64("competition_id", param.CompetitionID),
		logger.Slice("problem_ids", param.ProblemIDs))

	err := h.competitionSvc.EnableCompetitionProblem(ctx, param)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("EnableCompetitionProblem failed: %s", err.Error()),
		})
		h.log.ErrorContext(ctx, "EnableCompetitionProblem failed", logger.Error(err))
		return
	}
	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
	})
}

func (h *CompetitionHandler) DisableCompetitionProblem(c *gin.Context, param *model.CompetitionProblemParam) {
	ctx := loggerv2.ContextWithFields(c.Request.Context(),
		logger.Uint64("competition_id", param.CompetitionID),
		logger.Slice("problem_ids", param.ProblemIDs))

	err := h.competitionSvc.DisableCompetitionProblem(ctx, param)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("DisableCompetitionProblem failed: %s", err.Error()),
		})
		h.log.ErrorContext(ctx, "DisableCompetitionProblem failed", logger.Error(err))
		return
	}
	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
	})
}

func (h *CompetitionHandler) StartCompetition(c *gin.Context, param *model.StartCompetitionParam) {
	ctx := loggerv2.ContextWithFields(c.Request.Context(), logger.Uint64("competition_id", param.CompetitionID))

	// 检查用户是否在比赛名单中
	ok, err := h.competitionSvc.CheckUserInCompetition(ctx, param.CompetitionID, param.Operator)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("CheckUserInCompetition failed: %s", err.Error()),
		})
		h.log.ErrorContext(ctx, "CheckUserInCompetition failed", logger.Error(err))
		return
	}
	if !ok {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusForbidden,
			Message: "You are not in the competition user list",
		})
		h.log.InfoContext(ctx, "CheckUserInCompetition failed, user not in competition user list")
		return
	}

	// 检查是否在比赛时间内
	ok, err = h.competitionSvc.CheckCompetitionTime(ctx, param.CompetitionID)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("CheckCompetitionTime failed: %s", err.Error()),
		})
		h.log.ErrorContext(ctx, "CheckCompetitionTime failed", logger.Error(err))
		return
	}
	if !ok {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusForbidden,
			Message: "不在比赛时间内",
		})
		return
	}

	// 设置比赛 token
	err = h.jwtHandler.SetCompetitionToken(c, param.CompetitionID, param.Operator)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("SetCompetitionToken failed: %s", err.Error()),
		})
		h.log.ErrorContext(ctx, "SetCompetitionToken failed", logger.Error(err))
		return
	}

	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
	})
}

func (h *CompetitionHandler) GetCompetitionProblemListWithPresignedURL(c *gin.Context, param *model.GetCompetitionProblemListWithPresignedURLParam) {
	ctx := loggerv2.ContextWithFields(c.Request.Context(),
		logger.Uint64("competition_id", param.CompetitionID))

	problems, err := h.competitionSvc.GetCompetitionProblemListWithPresignedURL(ctx, param.CompetitionID)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("GetCompetitionProblemListWithPresignedURL failed: %s", err.Error()),
		})
		h.log.ErrorContext(ctx, "GetCompetitionProblemListWithPresignedURL failed", logger.Error(err))
		return
	}
	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    problems,
	})
}

func (h *CompetitionHandler) GetCompetitionRankingList(c *gin.Context, param *model.GetCompetitionRankingListParam) {
	ctx := loggerv2.ContextWithFields(c.Request.Context(),
		logger.Uint64("competition_id", param.CompetitionID))

	rankingList, total, err := h.rankingSvc.GetCompetitionRankingList(ctx, param.CompetitionID, param.Page, param.PageSize)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("GetCompetitionRankingList failed: %s", err.Error()),
		})
		h.log.ErrorContext(ctx, "GetCompetitionRankingList failed", logger.Error(err))
		return
	}
	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: model.GetCompetitionRankingListResponse{
			List:     rankingList,
			Total:    total,
			Page:     param.Page,
			PageSize: param.PageSize,
		},
	})
}
