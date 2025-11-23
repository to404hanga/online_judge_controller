package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
	ojmodel "github.com/to404hanga/online_judge_common/model"
	"github.com/to404hanga/online_judge_controller/constants"
	"github.com/to404hanga/online_judge_controller/model"
	"github.com/to404hanga/online_judge_controller/pkg/gintool"
	"github.com/to404hanga/online_judge_controller/service"
	"github.com/to404hanga/pkg404/logger"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
)

const SubmissionBucket = "submission"

type SubmissionHandler struct {
	// minioSvc                *minio.MinIOService
	submissionSvc  service.SubmissionService
	competitionSvc service.CompetitionService
	log            loggerv2.Logger
	// bucket                  string
	// uploadDurationSeconds   int
	// downloadDurationSeconds int
}

var _ Handler = (*SubmissionHandler)(nil)

func NewSubmissionHandler(submissionSvc service.SubmissionService, competitionSvc service.CompetitionService, log loggerv2.Logger) *SubmissionHandler {
	return &SubmissionHandler{
		submissionSvc:  submissionSvc,
		competitionSvc: competitionSvc,
		log:            log,
	}
}

func (h *SubmissionHandler) Register(r *gin.Engine) {
	r.POST(constants.SubmitCompetitionProblemPath, gintool.WrapCompetitionHandler(h.SubmitCompetitionProblem, h.log))
	r.GET(constants.GetLatestSubmissionPath, gintool.WrapCompetitionHandler(h.GetLatestSubmission, h.log))
}

func (h *SubmissionHandler) SubmitCompetitionProblem(c *gin.Context, param *model.SubmitCompetitionProblemParam) {
	ctx := loggerv2.ContextWithFields(c.Request.Context(),
		logger.Uint64("competition_id", param.CompetitionID),
		logger.Uint64("problem_id", param.ProblemID),
		logger.Int8("language", param.Language))

	ok, err := h.competitionSvc.CheckCompetitionTime(ctx, param.CompetitionID)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "SubmitCompetitionProblem failed", logger.Error(err))
		return
	}
	if !ok {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusForbidden,
			Message: "不在比赛时间内, 禁止提交",
		})
		return
	}

	latestSubmission, err := h.submissionSvc.GetLatestSubmission(ctx, param.CompetitionID, param.ProblemID, param.Operator)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "SubmitCompetitionProblem failed", logger.Error(err))
		return
	}
	// 如果最近的一次提交还没判题完毕, 禁止提交
	if latestSubmission != nil && *latestSubmission.Status != ojmodel.SubmissionStatusJudged {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusForbidden,
			Message: "You have submitted this problem, please wait for the result",
		})
		return
	}

	err = h.submissionSvc.SubmitCompetitionProblem(ctx, param)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "SubmitCompetitionProblem failed", logger.Error(err))
		return
	}

	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
	})
}

func (h *SubmissionHandler) GetLatestSubmission(c *gin.Context, param *model.GetLatestSubmissionParam) {
	ctx := loggerv2.ContextWithFields(c.Request.Context(),
		logger.Uint64("competition_id", param.CompetitionID),
		logger.Uint64("problem_id", param.ProblemID))

	submission, err := h.submissionSvc.GetLatestSubmission(ctx, param.CompetitionID, param.ProblemID, param.Operator)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "GetLatestSubmission failed", logger.Error(err))
		return
	}

	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: model.GetLatestSubmissionResponse{
			Submission: model.Submission{
				ID:         submission.ID,
				Language:   submission.Language.Int8(),
				Status:     submission.Status.Int8(),
				Result:     submission.Result.Int8(),
				TimeUsed:   *submission.TimeUsed,
				MemoryUsed: *submission.MemoryUsed,
				CreatedAt:  submission.CreatedAt,
			},
		},
	})
}
