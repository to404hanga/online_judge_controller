package web

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	ojmodel "github.com/to404hanga/online_judge_common/model"
	"github.com/to404hanga/online_judge_controller/constants"
	"github.com/to404hanga/online_judge_controller/model"
	"github.com/to404hanga/online_judge_controller/pkg/gintool"
	"github.com/to404hanga/online_judge_controller/service"
	"github.com/to404hanga/online_judge_controller/service/exporter/factory"
	"github.com/to404hanga/online_judge_controller/web/jwt"
	"github.com/to404hanga/pkg404/gotools/transform"
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
	r.GET(constants.GetCompetitionRankingListPath, gintool.WrapCompetitionHandler(h.GetCompetitionRankingList, h.log))
	r.GET(constants.GetCompetitionFastestSolverListPath, gintool.WrapCompetitionHandler(h.GetCompetitionFastestSolverList, h.log))
	r.GET(constants.ExportCompetitionDataPath, gintool.WrapHandler(h.ExportCompetitionData, h.log))
	r.POST(constants.InitRankingPath, gintool.WrapHandler(h.InitRanking, h.log))
	r.PUT(constants.UpdateScorePath, gintool.WrapHandler(h.UpdateScore, h.log)) // 仅内部测试用, 后续 release 版本移除
	r.GET(constants.GetCompetitionListPath, gintool.WrapHandler(h.GetCompetitionList, h.log))
	// TODO: 增加用户获取比赛题目列表以及获取比赛题目详情接口
	// TODO: 增加用户获取比赛列表接口
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

	competition, err := h.competitionSvc.GetCompetition(ctx, param.ID)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("UpdateCompetition failed: %s", err.Error()),
		})
		h.log.ErrorContext(ctx, "UpdateCompetition failed", logger.Error(err))
		return
	}
	if competition == nil || competition.ID == 0 {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusNotFound,
			Message: fmt.Sprintf("Competition %d not found", param.ID),
		})
		h.log.ErrorContext(ctx, "UpdateCompetition Competition not found",
			logger.Uint64("competition_id", param.ID))
		return
	}

	newStartTime, newEndTime := competition.StartTime, competition.EndTime
	if param.StartTime != nil {
		newStartTime = *param.StartTime
	}
	if param.EndTime != nil {
		newEndTime = *param.EndTime
	}
	if !newEndTime.After(newStartTime) {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusBadRequest,
			Message: "EndTime must be after StartTime",
		})
		h.log.ErrorContext(ctx, "UpdateCompetition EndTime must be after StartTime",
			logger.String("start_time", newStartTime.Format(time.RFC3339)),
			logger.String("end_time", newEndTime.Format(time.RFC3339)))
		return
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

	err = h.competitionSvc.UpdateCompetition(ctx, param)
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

func (h *CompetitionHandler) GetCompetitionFastestSolverList(c *gin.Context, param *model.GetCompetitionFastestSolverListParam) {
	ctx := loggerv2.ContextWithFields(c.Request.Context(),
		logger.Uint64("competition_id", param.CompetitionID))

	if len(param.ProblemIDs) == 0 {
		problemList, err := h.competitionSvc.GetCompetitionProblemList(ctx, param.CompetitionID)
		if err != nil {
			gintool.GinResponse(c, &gintool.Response{
				Code:    http.StatusInternalServerError,
				Message: fmt.Sprintf("GetCompetitionProblemList failed: %s", err.Error()),
			})
			h.log.ErrorContext(ctx, "GetCompetitionProblemList failed", logger.Error(err))
			return
		}
		param.ProblemIDs = transform.SliceFromSlice(problemList, func(i int, problem ojmodel.CompetitionProblem) uint64 {
			return problem.ID
		})
	}

	ctx = loggerv2.ContextWithFields(ctx, logger.Slice("problem_id_list", param.ProblemIDs))

	// 不关心查询成功与否, Redis 由 xxx 负责维护
	// TODO 将 xxx 改为具体的服务
	fastestSolverList := h.rankingSvc.GetFastestSolverList(ctx, param.CompetitionID, param.ProblemIDs)
	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: model.GetCompetitionFastestSolverListResponse{
			List:  fastestSolverList,
			Total: len(fastestSolverList),
		},
	})
}

func (h *CompetitionHandler) ExportCompetitionData(c *gin.Context, param *model.ExportCompetitionDataParam) {
	ctx := loggerv2.ContextWithFields(c.Request.Context(),
		logger.Uint64("competition_id", param.CompetitionID))

	exporterType := param.ExportType.ToFactoryType()
	if exporterType == factory.UnknownExporter {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("Unknown exporter type: %d", param.ExportType),
		})
		h.log.ErrorContext(ctx, "Unknown exporter type", logger.Int8("export_type", int8(param.ExportType)))
		return
	}
	ctx = loggerv2.ContextWithFields(ctx, logger.String("export_type", string(exporterType)))

	filepath, err := h.rankingSvc.Export(ctx, param.CompetitionID, exporterType)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Export failed: %s", err.Error()),
		})
		h.log.ErrorContext(ctx, "Export failed", logger.Error(err))
		return
	}

	// 将文件内容响应给前端
	c.File(filepath)
}

func (h *CompetitionHandler) InitRanking(c *gin.Context, param *model.InitRankingParam) {
	ctx := loggerv2.ContextWithFields(c.Request.Context(),
		logger.Uint64("competition_id", param.CompetitionID))

	err := h.rankingSvc.InitCompetitionRanking(ctx, param.CompetitionID)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("InitCompetitionRanking failed: %s", err.Error()),
		})
		h.log.ErrorContext(ctx, "InitCompetitionRanking failed", logger.Error(err))
		return
	}
	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
	})
}

// 仅内部测试用, 后续 release 版本移除
func (h *CompetitionHandler) UpdateScore(c *gin.Context, param *model.UpdateScoreParam) {
	ctx := loggerv2.ContextWithFields(c.Request.Context(),
		logger.Uint64("competition_id", param.CompetitionID))

	err := h.rankingSvc.UpdateUserScore(ctx, param.CompetitionID, param.ProblemID, param.UserID, *param.IsAccepted, param.SubmissionTime, param.StartTime)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("UpdateUserScore failed: %s", err.Error()),
		})
		h.log.ErrorContext(ctx, "UpdateUserScore failed", logger.Error(err))
		return
	}
	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
	})
}

func (h *CompetitionHandler) GetCompetitionList(c *gin.Context, param *model.GetCompetitionListParam) {
	fields := []logger.Field{
		logger.Bool("desc", param.Desc),
		logger.String("order_by", param.OrderBy),
		logger.Int("page", param.Page),
		logger.Int("page_size", param.PageSize),
	}
	if param.Status != nil {
		fields = append(fields, logger.Int8("status", param.Status.Int8()))
	}
	if param.Name != "" {
		fields = append(fields, logger.String("name", param.Name))
	}
	ctx := loggerv2.ContextWithFields(c.Request.Context(), fields...)
	h.log.DebugContext(ctx, "GetCompetitionList param")

	competitionList, total, err := h.competitionSvc.GetCompetitionList(ctx, param)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("GetCompetitionList failed: %s", err.Error()),
		})
		h.log.ErrorContext(ctx, "GetCompetitionList failed", logger.Error(err))
		return
	}
	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: model.GetCompetitionListResponse{
			List:     competitionList,
			Total:    total,
			Page:     param.Page,
			PageSize: param.PageSize,
		},
	})
}
