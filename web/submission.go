package web

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	ojmodel "github.com/to404hanga/online_judge_common/model"
	"github.com/to404hanga/online_judge_controller/constants"
	"github.com/to404hanga/online_judge_controller/model"
	"github.com/to404hanga/online_judge_controller/pkg/gintool"
	"github.com/to404hanga/online_judge_controller/pkg/pointer"
	"github.com/to404hanga/online_judge_controller/service"
	"github.com/to404hanga/pkg404/logger"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
)

const SubmissionBucket = "submission"

var (
	submitCompetitionProblemRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "online_judge_controller",
			Subsystem: "submission",
			Name:      "submit_competition_problem_requests_total",
			Help:      "SubmitCompetitionProblem requests total.",
		},
		[]string{"code", "reason", "language"},
	)
	submitCompetitionProblemDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "online_judge_controller",
			Subsystem: "submission",
			Name:      "submit_competition_problem_duration_seconds",
			Help:      "SubmitCompetitionProblem duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"code", "reason", "language"},
	)
)

func init() {
	prometheus.MustRegister(submitCompetitionProblemRequestsTotal, submitCompetitionProblemDurationSeconds)
}

type SubmissionHandler struct {
	submissionSvc  service.SubmissionService
	competitionSvc service.CompetitionService
	log            loggerv2.Logger
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
	start := time.Now()
	code := http.StatusOK
	reason := "ok"
	languageLabel := strconv.FormatInt(int64(param.Language), 10)
	defer func() {
		codeLabel := strconv.Itoa(code)
		submitCompetitionProblemRequestsTotal.WithLabelValues(codeLabel, reason, languageLabel).Inc()
		submitCompetitionProblemDurationSeconds.WithLabelValues(codeLabel, reason, languageLabel).Observe(time.Since(start).Seconds())
	}()

	ctx := loggerv2.ContextWithFields(c.Request.Context(),
		logger.Uint64("competition_id", param.CompetitionID),
		logger.Uint64("problem_id", param.ProblemID),
		logger.Int8("language", param.Language))

	ok, err := h.competitionSvc.CheckCompetitionTime(ctx, param.CompetitionID)
	if err != nil {
		code = http.StatusInternalServerError
		reason = "check_competition_time_error"
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "SubmitCompetitionProblem failed", logger.Error(err))
		return
	}
	if !ok {
		code = http.StatusForbidden
		reason = "not_in_competition_time"
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusForbidden,
			Message: "不在比赛时间内, 禁止提交",
		})
		return
	}

	latestSubmission, err := h.submissionSvc.GetLatestSubmission(ctx, param.CompetitionID, param.ProblemID, param.Operator)
	if err != nil {
		code = http.StatusInternalServerError
		reason = "get_latest_submission_error"
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "SubmitCompetitionProblem failed", logger.Error(err))
		return
	}
	// 如果最近的一次提交还没判题完毕, 禁止提交
	if latestSubmission != nil && latestSubmission.ID != 0 && *latestSubmission.Status != ojmodel.SubmissionStatusJudged {
		code = http.StatusForbidden
		reason = "latest_submission_not_judged"
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusForbidden,
			Message: "You have submitted this problem, please wait for the result",
		})
		return
	}

	// 接下来需要调用其他服务，ctx 携带 request_id 进行传递
	ctx = context.WithValue(ctx, "request_id", c.GetHeader(constants.HeaderRequestIDKey))
	err = h.submissionSvc.SubmitCompetitionProblem(ctx, param)
	if err != nil {
		code = http.StatusInternalServerError
		reason = "submit_competition_problem_error"
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
	if submission == nil || submission.ID == 0 {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusNotFound,
			Message: "No submission found",
		})
		return
	}

	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: model.GetLatestSubmissionResponse{
			Submission: model.Submission{
				ID:         submission.ID,
				Code:       submission.Code,
				Stderr:     pointer.FromPtr(submission.Stderr),
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
