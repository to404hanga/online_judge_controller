package web

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	ojmodel "github.com/to404hanga/online_judge_common/model"
	"github.com/to404hanga/online_judge_controller/constants"
	"github.com/to404hanga/online_judge_controller/model"
	"github.com/to404hanga/online_judge_controller/pkg/gintool"
	"github.com/to404hanga/online_judge_controller/service"
	"github.com/to404hanga/pkg404/gotools/transform"
	"github.com/to404hanga/pkg404/logger"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
)

type UserHandler struct {
	log            loggerv2.Logger
	userSvc        service.UserService
	competitionSvc service.CompetitionService
}

var _ Handler = (*UserHandler)(nil)

func NewUserHandler(log loggerv2.Logger, userSvc service.UserService, competitionSvc service.CompetitionService) *UserHandler {
	return &UserHandler{
		log:            log,
		userSvc:        userSvc,
		competitionSvc: competitionSvc,
	}
}

func (h *UserHandler) Register(r *gin.Engine) {
	r.GET(constants.GetUserListPath, gintool.WrapHandler(h.GetUserList, h.log))
	r.POST(constants.AddUsersToCompetitionPath, gintool.WrapHandler(h.AddUsersToCompetition, h.log))
	r.PUT(constants.EnableUsersInCompetitionPath, gintool.WrapHandler(h.EnableUsersInCompetition, h.log))
	r.PUT(constants.DisableUsersInCompetitionPath, gintool.WrapHandler(h.DisableUsersInCompetition, h.log))
	r.DELETE(constants.DeleteUserPath, gintool.WrapHandler(h.DeleteUser, h.log))
	r.PUT(constants.UpdateUserPath, gintool.WrapHandler(h.UpdateUser, h.log))
	r.PUT(constants.ResetPasswordPath, gintool.WrapHandler(h.ResetPassword, h.log))
	r.PUT(constants.UpdatePasswordPath, gintool.WrapHandler(h.UpdatePassword, h.log))
	r.GET(constants.GetCompetitionUserListPath, gintool.WrapHandler(h.GetCompetitionUserList, h.log))
	r.POST(constants.CreateUserPath, gintool.WrapHandler(h.CreateUser, h.log))
}

func (h *UserHandler) GetUserList(c *gin.Context, param *model.GetUserListParam) {
	if len(param.OrderBy) == 0 {
		param.OrderBy = "id"
	}
	fields := []logger.Field{
		logger.String("order_by", param.OrderBy),
		logger.Bool("desc", param.Desc),
		logger.Int("page", param.Page),
		logger.Int("page_size", param.PageSize),
	}
	if param.Username != "" {
		fields = append(fields, logger.String("username", param.Username))
	}
	if param.Realname != "" {
		fields = append(fields, logger.String("realname", param.Realname))
	}
	if param.Role != nil {
		fields = append(fields, logger.Int8("role", param.Role.Int8()))
	}
	if param.Status != nil {
		fields = append(fields, logger.Int8("status", param.Status.Int8()))
	}

	ctx := loggerv2.ContextWithFields(c.Request.Context(), fields...)
	h.log.DebugContext(ctx, "GetUserList param")

	users, err := h.userSvc.GetUserList(c.Request.Context(), param)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: "internal error",
		})
		h.log.ErrorContext(ctx, "GetUserList failed", logger.Error(err))
		return
	}

	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: model.GetUserListResponse{
			Total:    len(users),
			List:     users,
			Page:     param.Page,
			PageSize: param.PageSize,
		},
	})
}

func (h *UserHandler) getUserMapFromFile(c *gin.Context) map[uint64]*ojmodel.User {
	ctx := c.Request.Context()

	filerHeader, err := c.FormFile("file")
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "AddUsersToCompetition get file failed", logger.Error(err))
		return nil
	}

	tmpDir, err := os.MkdirTemp("", "oj_cu_*")
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "AddUsersToCompetition create tmp dir failed", logger.Error(err))
		return nil
	}
	defer os.RemoveAll(tmpDir)

	tmpPath := tmpDir + "/upload.csv"
	if err = c.SaveUploadedFile(filerHeader, tmpPath); err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "AddUsersToCompetition save file failed", logger.Error(err))
		return nil
	}

	b, err := os.ReadFile(tmpPath)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "AddUsersToCompetition read file failed", logger.Error(err))
		return nil
	}

	reader := csv.NewReader(bytes.NewReader(b))
	records, err := reader.ReadAll()
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "AddUsersToCompetition read file failed", logger.Error(err))
		return nil
	}
	if len(records) <= 1 {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusBadRequest,
			Message: "file is empty",
		})
		h.log.ErrorContext(ctx, "AddUsersToCompetition file is empty", logger.Error(err))
		return nil
	}
	records = records[1:] // 丢弃表头行

	usernameList := transform.SliceFromSlice(records, func(i int, record []string) string {
		if len(record) > 0 {
			return record[0]
		}
		return ""
	})

	userList, err := h.userSvc.GetUserListByUsernameList(ctx, usernameList)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "AddUsersToCompetition get user list failed", logger.Error(err))
		return nil
	}
	if len(userList) == 0 {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusBadRequest,
			Message: "no user found",
		})
		h.log.ErrorContext(ctx, "AddUsersToCompetition no user found",
			logger.Slice("username_list", usernameList))
		return nil
	}

	return transform.MapFromSlice(userList, func(i int, user ojmodel.User) (uint64, *ojmodel.User) {
		return user.ID, &user
	})
}

func (h *UserHandler) getUserMapFromUserIDList(ctx *gin.Context, userIDList []uint64) map[uint64]*ojmodel.User {
	userList, err := h.userSvc.GetUserListByIDList(ctx, userIDList)
	if err != nil {
		gintool.GinResponse(ctx, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "AddUsersToCompetition get user list failed", logger.Error(err))
		return nil
	}
	return transform.MapFromSlice(userList, func(i int, user ojmodel.User) (uint64, *ojmodel.User) {
		return user.ID, &user
	})
}

func (h *UserHandler) checkCompetitionExist(ctx context.Context, competitionID uint64) (bool, error) {
	competition, err := h.competitionSvc.GetCompetition(ctx, competitionID)
	if err != nil {
		return false, fmt.Errorf("check competition exist failed: %w", err)
	}
	return competition != nil && competition.ID != 0, nil
}

func (h *UserHandler) AddUsersToCompetition(c *gin.Context, param *model.AddUsersToCompetition) {
	ctx := loggerv2.WithFieldsToContext(c.Request.Context(), logger.Uint64("competition_id", param.CompetitionID))

	exist, err := h.checkCompetitionExist(ctx, param.CompetitionID)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "AddUsersToCompetition check competition exist failed", logger.Error(err))
		return
	}
	if !exist {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusBadRequest,
			Message: "competition not found",
		})
		h.log.ErrorContext(ctx, "AddUsersToCompetition competition not found")
		return
	}

	var userMap map[uint64]*ojmodel.User
	if len(param.UserIDList) == 0 && c.ContentType() == binding.MIMEMultipartPOSTForm {
		// 如果前端没有传入 UserIDList, 则证明是从文件上传的用户
		userMap = h.getUserMapFromFile(c)
	} else if len(param.UserIDList) > 0 {
		// 如果前端传入了 UserIDList, 则证明是从界面勾选的用户
		userMap = h.getUserMapFromUserIDList(c, param.UserIDList)
	} else {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusBadRequest,
			Message: "unsupported content type",
		})
		h.log.ErrorContext(ctx, "AddUsersToCompetition unsupported content type")
		return
	}
	if len(userMap) == 0 {
		h.log.InfoContext(ctx, "user len is 0")
		return
	}

	rowsAffected, err := h.userSvc.AddUsersToCompetition(ctx, param.CompetitionID, userMap)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "AddUsersToCompetition failed", logger.Error(err))
		return
	}

	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: &model.AddUsersToCompetitionResponse{
			InsertSuccess: rowsAffected,
		},
	})
}

func (h *UserHandler) EnableUsersInCompetition(c *gin.Context, param *model.CompetitionUserListParam) {
	ctx := loggerv2.WithFieldsToContext(c.Request.Context(), logger.Uint64("competition_id", param.CompetitionID))

	exist, err := h.checkCompetitionExist(ctx, param.CompetitionID)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "EnableUsersInCompetition check competition exist failed", logger.Error(err))
		return
	}
	if !exist {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusBadRequest,
			Message: "competition not found",
		})
		h.log.ErrorContext(ctx, "EnableUsersInCompetition competition not found")
		return
	}

	userList, err := h.userSvc.GetUserListByIDList(ctx, param.UserIDList)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "EnableUsersInCompetition get user list failed", logger.Error(err))
		return
	}
	if len(userList) != len(param.UserIDList) {
		notExistUserIDList := make([]uint64, 0, len(param.UserIDList)-len(userList))
		for _, userID := range param.UserIDList {
			exist := false
			for _, user := range userList {
				if user.ID == userID {
					exist = true
					break
				}
			}
			if !exist {
				notExistUserIDList = append(notExistUserIDList, userID)
			}
		}
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("user id %v not exist", notExistUserIDList),
		})
		h.log.ErrorContext(ctx, "EnableUsersInCompetition user id not exist",
			logger.Slice("user_id_list", notExistUserIDList))
		return
	}

	err = h.userSvc.UpdateCompetitionUserStatus(ctx, param, ojmodel.CompetitionUserStatusNormal)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: "internal error",
		})
		h.log.ErrorContext(ctx, "EnableUsersInCompetition failed", logger.Error(err))
		return
	}

	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
	})
}

func (h *UserHandler) DisableUsersInCompetition(c *gin.Context, param *model.CompetitionUserListParam) {
	ctx := loggerv2.WithFieldsToContext(c.Request.Context(), logger.Uint64("competition_id", param.CompetitionID))

	exist, err := h.checkCompetitionExist(ctx, param.CompetitionID)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "DisableUsersInCompetition check competition exist failed", logger.Error(err))
		return
	}
	if !exist {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusBadRequest,
			Message: "competition not found",
		})
		h.log.ErrorContext(ctx, "DisableUsersInCompetition competition not found")
		return
	}

	userList, err := h.userSvc.GetUserListByIDList(ctx, param.UserIDList)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		h.log.ErrorContext(ctx, "DisableUsersInCompetition get user list failed", logger.Error(err))
		return
	}
	if len(userList) != len(param.UserIDList) {
		notExistUserIDList := make([]uint64, 0, len(param.UserIDList)-len(userList))
		for _, userID := range param.UserIDList {
			exist := false
			for _, user := range userList {
				if user.ID == userID {
					exist = true
					break
				}
			}
			if !exist {
				notExistUserIDList = append(notExistUserIDList, userID)
			}
		}
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusBadRequest,
			Message: fmt.Sprintf("user id %v not exist", notExistUserIDList),
		})
		h.log.ErrorContext(ctx, "EnableUsersInCompetition user id not exist",
			logger.Slice("user_id_list", notExistUserIDList))
		return
	}

	err = h.userSvc.UpdateCompetitionUserStatus(ctx, param, ojmodel.CompetitionUserStatusDisabled)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: "internal error",
		})
		h.log.ErrorContext(ctx, "DisableUsersInCompetition failed", logger.Error(err))
		return
	}

	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
	})
}

func (h *UserHandler) DeleteUser(c *gin.Context, param *model.DeleteUserParam) {
	ctx := loggerv2.WithFieldsToContext(c.Request.Context(), logger.Uint64("user_id", param.UserID))

	err := h.userSvc.DeleteUserByID(ctx, param.UserID)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: "internal error",
		})
		h.log.ErrorContext(ctx, "DeleteUser failed", logger.Error(err))
		return
	}

	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
	})
}

func (h *UserHandler) UpdateUser(c *gin.Context, param *model.UpdateUserParam) {
	ctx := loggerv2.WithFieldsToContext(c.Request.Context(), logger.Uint64("user_id", param.UserID))

	err := h.userSvc.UpdateUser(ctx, param)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: "internal error",
		})
		h.log.ErrorContext(ctx, "UpdateUser failed", logger.Error(err))
		return
	}

	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
	})
}

func (h *UserHandler) ResetPassword(c *gin.Context, param *model.ResetPasswordParam) {
	ctx := loggerv2.WithFieldsToContext(c.Request.Context(), logger.Uint64("user_id", param.UserID))

	err := h.userSvc.ResetUserPassword(ctx, param.UserID)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: "internal error",
		})
		h.log.ErrorContext(ctx, "ResetPassword failed", logger.Error(err))
		return
	}

	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
	})
}

func (h *UserHandler) UpdatePassword(c *gin.Context, param *model.UpdatePasswordParam) {
	ctx := loggerv2.WithFieldsToContext(c.Request.Context(), logger.Uint64("user_id", param.UserID))

	oldPasswordMatch, err := h.userSvc.UpdateUserPassword(ctx, param.UserID, param.NewPassword)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: "internal error",
		})
		h.log.ErrorContext(ctx, "UpdatePassword update user password failed", logger.Error(err))
		return
	}
	if !oldPasswordMatch {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusBadRequest,
			Message: "old password not match",
		})
		h.log.ErrorContext(ctx, "UpdatePassword old password not match")
		return
	}

	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
	})
}

func (h *UserHandler) GetCompetitionUserList(c *gin.Context, param *model.GetCompetitionUserListParam) {
	if param.OrderBy == "" {
		param.OrderBy = "id"
	}

	fields := []logger.Field{
		logger.Uint64("competition_id", param.CompetitionID),
		logger.String("order_by", param.OrderBy),
		logger.Bool("desc", param.Desc),
		logger.Int("page", param.Page),
		logger.Int("page_size", param.PageSize),
	}
	if param.Username != "" {
		fields = append(fields, logger.String("username", param.Username))
	}
	if param.Realname != "" {
		fields = append(fields, logger.String("realname", param.Realname))
	}
	if param.Status != nil {
		fields = append(fields, logger.Int8("status", param.Status.Int8()))
	}

	ctx := loggerv2.WithFieldsToContext(c.Request.Context(), fields...)
	h.log.DebugContext(ctx, "GetCompetitionUserList")

	userList, total, err := h.userSvc.GetCompetitionUserList(ctx, param)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: "internal error",
		})
		h.log.ErrorContext(ctx, "GetCompetitionUserList failed", logger.Error(err))
		return
	}

	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
		Data: model.GetCompetitionUserListResponse{
			Total:    total,
			List:     userList,
			Page:     param.Page,
			PageSize: param.PageSize,
		},
	})
}

func (h *UserHandler) CreateUser(c *gin.Context, param *model.CreateUserParam) {
	ctx := loggerv2.WithFieldsToContext(c.Request.Context(),
		logger.String("username", param.Username),
		logger.String("realname", param.Realname),
		logger.Int8("role", param.Role.Int8()),
	)

	err := h.userSvc.CreateUser(ctx, param.Username, param.Realname, param.Role)
	if err != nil {
		gintool.GinResponse(c, &gintool.Response{
			Code:    http.StatusInternalServerError,
			Message: "internal error",
		})
		h.log.ErrorContext(ctx, "CreateUser failed", logger.Error(err))
		return
	}

	gintool.GinResponse(c, &gintool.Response{
		Code:    http.StatusOK,
		Message: "success",
	})
}
