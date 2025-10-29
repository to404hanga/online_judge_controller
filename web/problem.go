package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/to404hanga/online_judge_controller/constants"
	"github.com/to404hanga/online_judge_controller/model"
	"github.com/to404hanga/online_judge_controller/pkg/gintool"
	"github.com/to404hanga/online_judge_controller/pkg/minio"
	"github.com/to404hanga/online_judge_controller/service"
	"github.com/to404hanga/pkg404/logger"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
)

type ProblemHandler struct {
	problemSvc              service.ProblemService
	minioSvc                *minio.MinIOService
	log                     loggerv2.Logger
	problemBucket           string
	testcaseBucket          string
	uploadDurationSeconds   int
	downloadDurationSeconds int
}

var _ Handler = (*ProblemHandler)(nil)

func NewProblemHandler(problemSvc service.ProblemService, minioSvc *minio.MinIOService, log loggerv2.Logger, problemBucket, testcaseBucket string, uploadDurationSeconds, downloadDurationSeconds int) *ProblemHandler {
	return &ProblemHandler{
		problemSvc:              problemSvc,
		minioSvc:                minioSvc,
		log:                     log,
		problemBucket:           problemBucket,
		testcaseBucket:          testcaseBucket,
		uploadDurationSeconds:   uploadDurationSeconds,
		downloadDurationSeconds: downloadDurationSeconds,
	}
}

func (h *ProblemHandler) Register(r *gin.Engine) {
	r.POST(constants.CreateProblemPath, gintool.WrapHandler(h.CreateProblem, h.log))
	r.PUT(constants.UpdateProblemPath, gintool.WrapHandler(h.UpdateProblem, h.log))
	r.GET(constants.GetProblemUploadPresignedURLPath, gintool.WrapHandler(h.GetProblemUploadPresignedURL, h.log))
	r.GET(constants.GetProblemDownloadPresignedURLPath, gintool.WrapHandler(h.GetProblemDownloadPresignedURL, h.log))
	r.GET(constants.GetProblemTestcaseUploadPresignedURLPath, gintool.WrapHandler(h.GetProblemTestcaseUploadPresignedURL, h.log))
	r.GET(constants.GetProblemTestcaseDownloadPresignedURLPath, gintool.WrapHandler(h.GetProblemTestcaseDownloadPresignedURL, h.log))
	r.GET(constants.GetProblemListPath, gintool.WrapHandler(h.GetProblemList, h.log))
}

func (h *ProblemHandler) CreateProblem(c *gin.Context, param *model.CreateProblemParam) {
	err := h.problemSvc.CreateProblem(c.Request.Context(), param)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(c.Request.Context(), "CreateProblem failed", logger.Error(err))
		return
	}

	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
	})
}

func (h *ProblemHandler) UpdateProblem(c *gin.Context, param *model.UpdateProblemParam) {
	err := h.problemSvc.UpdateProblem(c.Request.Context(), param)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(c.Request.Context(), "UpdateProblem failed", logger.Error(err))
		return
	}

	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
	})
}

func (h *ProblemHandler) GetProblemUploadPresignedURL(c *gin.Context, param *model.GetProblemUploadPresignedURLParam) {
	ctx := loggerv2.ContextWithFields(c.Request.Context(), logger.String("hash", param.Hash))

	presignedURL, err := h.minioSvc.GetPresignedUploadURL(ctx, h.problemBucket, param.Hash, h.uploadDurationSeconds)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "GetProblemUploadPresignedURL failed", logger.Error(err))
		return
	}

	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: &model.GetProblemUploadPresignedURLResponse{
			PresignedURL: presignedURL,
		},
	})
}

func (h *ProblemHandler) GetProblemDownloadPresignedURL(c *gin.Context, param *model.GetProblemDownloadPresignedURLParam) {
	ctx := loggerv2.ContextWithFields(c.Request.Context(), logger.Uint64("problem_id", param.ProblemID))

	problem, err := h.problemSvc.GetProblemByID(ctx, param.ProblemID)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "GetProblemDownloadPresignedURL failed", logger.Error(err))
		return
	}

	presignedURL, err := h.minioSvc.GetPresignedDownloadURL(ctx, h.problemBucket, problem.DescriptionURL, h.downloadDurationSeconds)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "GetProblemDownloadPresignedURL failed", logger.Error(err))
		return
	}

	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: &model.GetProblemDownloadPresignedURLResponse{
			PresignedURL: presignedURL,
		},
	})
}

func (h *ProblemHandler) GetProblemTestcaseUploadPresignedURL(c *gin.Context, param *model.GetProblemTestcaseUploadPresignedURLParam) {
	ctx := loggerv2.ContextWithFields(c.Request.Context(), logger.String("hash", param.Hash))

	presignedURL, err := h.minioSvc.GetPresignedUploadURL(ctx, h.testcaseBucket, param.Hash, h.uploadDurationSeconds)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "GetProblemTestcaseUploadPresignedURL failed", logger.Error(err))
		return
	}

	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: &model.GetProblemTestcaseUploadPresignedURLResponse{
			PresignedURL: presignedURL,
		},
	})
}

func (h *ProblemHandler) GetProblemTestcaseDownloadPresignedURL(c *gin.Context, param *model.GetProblemTestcaseDownloadPresignedURLParam) {
	ctx := loggerv2.ContextWithFields(c.Request.Context(), logger.Uint64("problem_id", param.ProblemID))

	problem, err := h.problemSvc.GetProblemByID(ctx, param.ProblemID)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "GetProblemTestcaseDownloadPresignedURL failed", logger.Error(err))
		return
	}

	presignedURL, err := h.minioSvc.GetPresignedDownloadURL(ctx, h.testcaseBucket, problem.TestcaseZipURL, h.downloadDurationSeconds)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "GetProblemTestcaseDownloadPresignedURL failed", logger.Error(err))
		return
	}

	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: &model.GetProblemTestcaseDownloadPresignedURLResponse{
			PresignedURL: presignedURL,
		},
	})
}

func (h *ProblemHandler) GetProblemList(c *gin.Context, param *model.GetProblemListParam) {
	fields := []logger.Field{
		logger.Bool("desc", param.Desc),
		logger.Int("page", param.Page),
		logger.Int("page_size", param.PageSize),
	}
	if param.Title != "" {
		fields = append(fields, logger.String("title", param.Title))
	}
	if param.Status != nil {
		fields = append(fields, logger.Int8("status", *param.Status))
	}
	if param.Visible != nil {
		fields = append(fields, logger.Int8("visible", *param.Visible))
	}
	if param.TimeLimit != nil {
		fields = append(fields, logger.Int("time_limit", *param.TimeLimit))
	}
	if param.MemoryLimit != nil {
		fields = append(fields, logger.Int("memory_limit", *param.MemoryLimit))
	}

	ctx := loggerv2.ContextWithFields(c.Request.Context(), fields...)

	problems, err := h.problemSvc.GetProblemList(ctx, param)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "GetProblemList failed", logger.Error(err))
		return
	}

	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: &model.GetProblemListResponse{
			List:  problems,
			Total: len(problems),
		},
	})
}
