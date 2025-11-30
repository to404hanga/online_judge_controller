package web

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/to404hanga/online_judge_controller/constants"
	"github.com/to404hanga/online_judge_controller/model"
	"github.com/to404hanga/online_judge_controller/pkg/gintool"
	"github.com/to404hanga/online_judge_controller/service"
	"github.com/to404hanga/online_judge_controller/web/jwt"
	"github.com/to404hanga/pkg404/logger"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
)

type ProblemHandler struct {
	problemSvc service.ProblemService
	//minioSvc                *minio.MinIOService
	log loggerv2.Logger
	// problemBucket           string
	// testcaseBucket          string
	// uploadDurationSeconds   int
	// downloadDurationSeconds int
}

var _ Handler = (*ProblemHandler)(nil)

func NewProblemHandler(problemSvc service.ProblemService, log loggerv2.Logger) *ProblemHandler {
	return &ProblemHandler{
		problemSvc: problemSvc,
		log:        log,
	}
}

func (h *ProblemHandler) Register(r *gin.Engine) {
	r.POST(constants.CreateProblemPath, gintool.WrapHandler(h.CreateProblem, h.log))
	r.PUT(constants.UpdateProblemPath, gintool.WrapHandler(h.UpdateProblem, h.log))
	r.GET(constants.GetProblemListPath, gintool.WrapHandler(h.GetProblemList, h.log))
	r.POST(constants.UploadProblemTestcasePath, gintool.WrapWithoutBodyHandler(h.UploadProblemTestcase, h.log))
	r.GET(constants.GetProblemPath, gintool.WrapHandler(h.GetProblem, h.log))
}

func (h *ProblemHandler) CreateProblem(c *gin.Context, param *model.CreateProblemParam) {
	// 计算 Description 的 SHA256 哈希
	hash := sha256.Sum256([]byte(param.Description))
	hashStr := hex.EncodeToString(hash[:])

	// 与传入的 DescriptionHash 进行比对
	if hashStr != param.DescriptionHash {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusBadRequest,
			Message: "description hash mismatch",
		})
		return
	}

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
	if param.Description != nil {
		if param.DescriptionHash == nil {
			gintool.GinResponse(c, &gintool.Response{
				Code:    http.StatusBadRequest,
				Message: "description hash is required",
			})
			return
		}

		// 计算 Description 的 SHA256 哈希
		hash := sha256.Sum256([]byte(*param.Description))
		hashStr := hex.EncodeToString(hash[:])

		// 与传入的 DescriptionHash 进行比对
		if hashStr != *param.DescriptionHash {
			gintool.GinResponse(c, &gintool.Response{
				Code:    http.StatusBadRequest,
				Message: "description hash mismatch",
			})
			return
		}
	}

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

func (h *ProblemHandler) UploadProblemTestcase(c *gin.Context, param *model.UploadProblemTestcaseParam) {
	pid := c.Query("problem_id")
	if pid != "" {
		id, err := strconv.ParseUint(pid, 10, 64)
		if err != nil {
			gintool.GinResponse(c, &gintool.Response{
				Code:    http.StatusBadRequest,
				Message: "problem_id is invalid",
			})
			return
		}
		param.ProblemID = id
	}

	ctx := loggerv2.ContextWithFields(c.Request.Context(), logger.Uint64("problem_id", param.ProblemID))

	fileHeader, err := c.FormFile("file")
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{Code: http.StatusBadRequest, Message: "file is required"})
		h.log.ErrorContext(ctx, "UploadProblemTestcase get file failed", logger.Error(err))
		return
	}

	tmpDir, err := os.MkdirTemp("", "oj_tc_*")
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{Code: http.StatusInternalServerError, Message: "internal error"})
		h.log.ErrorContext(ctx, "UploadProblemTestcase create temp dir failed", logger.Error(err))
		return
	}
	defer os.RemoveAll(tmpDir)
	tmpPath := tmpDir + "/upload.zip"
	if err = c.SaveUploadedFile(fileHeader, tmpPath); err != nil {
		gintool.GinResponse(c, &gintool.Response{Code: http.StatusInternalServerError, Message: "failed to save upload"})
		h.log.ErrorContext(ctx, "UploadProblemTestcase save file failed", logger.Error(err))
		return
	}

	zr, err := zip.OpenReader(tmpPath)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{Code: http.StatusBadRequest, Message: "invalid zip file"})
		h.log.ErrorContext(ctx, "UploadProblemTestcase open zip failed", logger.Error(err))
		return
	}
	defer zr.Close()

	base := "/testcases"
	if _, err := os.Stat(base); os.IsNotExist(err) {
		base = "./testcases"
	}
	destRoot := base + "/" + strconv.FormatUint(param.ProblemID, 10)
	_ = os.RemoveAll(destRoot)
	if err := os.MkdirAll(destRoot, 0755); err != nil {
		gintool.GinResponse(c, &gintool.Response{Code: http.StatusInternalServerError, Message: "failed to prepare dest"})
		h.log.ErrorContext(ctx, "UploadProblemTestcase mkdir failed", logger.Error(err))
		return
	}

	for _, f := range zr.File {
		parts := strings.Split(f.Name, "/")
		name := parts[len(parts)-1]
		if name == "." || strings.HasPrefix(name, "..") {
			continue
		}
		target := destRoot + "/" + name
		if !strings.HasPrefix(target, destRoot+string(os.PathSeparator)) && target != destRoot {
			gintool.GinResponse(c, &gintool.Response{Code: http.StatusBadRequest, Message: "invalid zip entry"})
			return
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				gintool.GinResponse(c, &gintool.Response{Code: http.StatusInternalServerError, Message: "failed to create dir"})
				h.log.ErrorContext(ctx, "UploadProblemTestcase mkdir entry failed", logger.Error(err), logger.String("entry", name))
				return
			}
			continue
		}
		// 仅为文件创建父目录，避免将文件路径创建为目录导致“is a directory”
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			gintool.GinResponse(c, &gintool.Response{Code: http.StatusInternalServerError, Message: "failed to create dir"})
			h.log.ErrorContext(ctx, "UploadProblemTestcase mkdir parent failed", logger.Error(err))
			return
		}
		rc, err := f.Open()
		if err != nil {
			gintool.GinResponse(c, &gintool.Response{Code: http.StatusInternalServerError, Message: "failed to read zip entry"})
			h.log.ErrorContext(ctx, "UploadProblemTestcase open entry failed", logger.Error(err))
			return
		}
		defer rc.Close()
		destFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			gintool.GinResponse(c, &gintool.Response{Code: http.StatusInternalServerError, Message: "failed to write file"})
			h.log.ErrorContext(ctx, "UploadProblemTestcase open target failed", logger.Error(err), logger.String("target", target))
			return
		}
		destFile.Close()
		content, err := io.ReadAll(rc)
		if err != nil {
			gintool.GinResponse(c, &gintool.Response{
				Code:    http.StatusInternalServerError,
				Message: "failed to read zip entry",
			})
			h.log.ErrorContext(ctx, "UploadProblemTestcase read entry failed", logger.Error(err))
			return
		}
		content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
		if _, err = destFile.Write(content); err != nil {
			gintool.GinResponse(c, &gintool.Response{
				Code:    http.StatusInternalServerError,
				Message: "failed to write file",
			})
			h.log.ErrorContext(ctx, "UploadProblemTestcase write target failed", logger.Error(err), logger.String("target", target))
			return
		}
	}

	gintool.GinResponse(c, &gintool.Response{Code: http.StatusOK, Message: "success"})
}

func (h *ProblemHandler) GetProblem(c *gin.Context, param *model.GetProblemParam) {
	ctx := loggerv2.ContextWithFields(c.Request.Context(), logger.Uint64("problem_id", param.ProblemID))

	if userClaims, exists := c.Get(constants.ContextUserClaimsKey); exists {
		competitionUserClaims, ok := userClaims.(jwt.CompetitionUserClaims)
		if !ok {
			gintool.GinResponse(c, &gintool.Response{
				Code:    http.StatusBadRequest,
				Message: "competition user claims type assertion failed",
			})
			h.log.ErrorContext(c.Request.Context(), "WrapCompetitionHandler competition user claims type assertion failed")
			return
		}
		param.SetCompetitionID(competitionUserClaims.CompetitionID)
	}

	problem, err := h.problemSvc.GetProblemByID(ctx, param.ProblemID, param.CompetitionID)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: "internal error",
		})
		h.log.ErrorContext(ctx, "GetProblemByID failed", logger.Error(err))
		return
	}

	gintool.GinResponse(c, &gintool.Response{Code: http.StatusOK, Data: model.GetProblemResponse{
		Problem: problem,
	}})
}
