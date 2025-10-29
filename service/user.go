package service

import (
	"context"

	ojmodel "github.com/to404hanga/online_judge_common/model"
	"github.com/to404hanga/pkg404/logger"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
	"gorm.io/gorm"
)

type UserService interface {
	// GetRoleByID 获取用户角色
	GetRoleByID(ctx context.Context, userID uint64) (ojmodel.UserRole, error)
}

type UserServiceImpl struct {
	db  *gorm.DB
	log loggerv2.Logger
}

func NewUserService(db *gorm.DB, log loggerv2.Logger) UserService {
	return &UserServiceImpl{
		db:  db,
		log: log,
	}
}

// GetRoleByID 获取用户角色
func (s *UserServiceImpl) GetRoleByID(ctx context.Context, userID uint64) (ojmodel.UserRole, error) {
	var user ojmodel.User
	err := s.db.WithContext(ctx).
		Where("id = ?", userID).
		Where("status = ?", ojmodel.UserStatusNormal).
		Select("role").
		First(&user).Error
	if err != nil {
		s.log.ErrorContext(ctx, "GetRoleByID failed", logger.Error(err))
		return ojmodel.UserRoleNormal, err
	}
	return *user.Role, nil
}
