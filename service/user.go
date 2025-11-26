package service

import (
	"context"
	"fmt"

	ojmodel "github.com/to404hanga/online_judge_common/model"
	"github.com/to404hanga/online_judge_controller/model"
	"github.com/to404hanga/online_judge_controller/pkg/pointer"
	"github.com/to404hanga/pkg404/gotools/transform"
	"github.com/to404hanga/pkg404/logger"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
	"gorm.io/gorm"
)

type UserService interface {
	// GetRoleByID 获取用户角色
	GetRoleByID(ctx context.Context, userID uint64) (ojmodel.UserRole, error)
	// GetUserList 获取用户列表
	GetUserList(ctx context.Context, param *model.GetUserListParam) ([]ojmodel.User, error)
	// AddUsersToCompetition 添加用户到比赛名单
	AddUsersToCompetition(ctx context.Context, competitionID uint64, userMap map[uint64]*ojmodel.User) (int64, error)
	// GetUserListByUsernameList 获取用户列表, 根据学号全匹配, 仅返回正常用户
	GetUserListByUsernameList(ctx context.Context, usernameList []string) ([]ojmodel.User, error)
	// GetUserListByIDList 获取用户列表, 根据ID列表, 仅返回正常用户
	GetUserListByIDList(ctx context.Context, idList []uint64) ([]ojmodel.User, error)
	// UpdateCompetitionUserStatus 更新比赛用户状态
	UpdateCompetitionUserStatus(ctx context.Context, param *model.CompetitionUserListParam, status ojmodel.CompetitionUserStatus) error
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

// GetUserList 获取用户列表
func (s *UserServiceImpl) GetUserList(ctx context.Context, param *model.GetUserListParam) ([]ojmodel.User, error) {
	var users []ojmodel.User
	query := s.db.WithContext(ctx)

	if len(param.Username) != 0 {
		query = query.Where("username like ?", param.Username+"%")
	}
	if len(param.Realname) != 0 {
		query = query.Where("realname like ?", "%"+param.Realname+"%")
	}
	if param.Role != nil {
		query = query.Where("role = ?", *param.Role)
	}
	if param.Status != nil {
		query = query.Where("status = ?", *param.Status)
	}

	orderBy := "id"
	if len(param.OrderBy) != 0 {
		orderBy = param.OrderBy
	}
	if param.Desc {
		query = query.Order(orderBy + " desc")
	} else {
		query = query.Order(orderBy + " asc")
	}

	err := query.Offset((param.Page-1)*param.PageSize).
		Limit(param.PageSize).
		Select("id", "username", "realname", "role", "status", "created_at", "updated_at").
		Find(&users).Error
	if err != nil {
		s.log.ErrorContext(ctx, "GetUserList failed", logger.Error(err))
		return nil, err
	}
	return users, nil
}

// AddUsersToCompetition 添加用户到比赛名单
func (s *UserServiceImpl) AddUsersToCompetition(ctx context.Context, competitionID uint64, userMap map[uint64]*ojmodel.User) (int64, error) {
	competitionUser := transform.SliceFromMap(userMap, func(userID uint64, user *ojmodel.User) ojmodel.CompetitionUser {
		return ojmodel.CompetitionUser{
			CompetitionID: competitionID,
			UserID:        userID,
			Username:      user.Username,
			Realname:      user.Realname,
			Status:        pointer.ToPtr(ojmodel.CompetitionUserStatusNormal), // 默认正常状态
		}
	})
	res := s.db.WithContext(ctx).Create(&competitionUser)
	if res.Error != nil {
		return 0, fmt.Errorf("AddUsersToCompetition failed: %w", res.Error)
	}
	return res.RowsAffected, nil
}

// GetUserListByUsernameList 获取用户列表, 根据学号全匹配, 仅返回正常用户
func (s *UserServiceImpl) GetUserListByUsernameList(ctx context.Context, usernameList []string) ([]ojmodel.User, error) {
	var users []ojmodel.User
	err := s.db.WithContext(ctx).
		Where("username in ?", usernameList).
		Where("status = ?", ojmodel.UserStatusNormal). // 被禁用的不返回
		Where("role = ?", ojmodel.UserRoleNormal).     // 不返回管理员用户
		Select("id", "username", "realname", "role", "status", "created_at", "updated_at").
		Find(&users).Error
	if err != nil {
		s.log.ErrorContext(ctx, "GetUserListByUsernameList failed", logger.Error(err))
		return nil, err
	}
	return users, nil
}

// GetUserListByIDList 获取用户列表, 根据ID列表, 仅返回正常用户
func (s *UserServiceImpl) GetUserListByIDList(ctx context.Context, idList []uint64) ([]ojmodel.User, error) {
	var users []ojmodel.User
	err := s.db.WithContext(ctx).
		Where("id in ?", idList).
		Where("status = ?", ojmodel.UserStatusNormal). // 被禁用的不返回
		Where("role = ?", ojmodel.UserRoleNormal).     // 不返回管理员用户
		Select("id", "username", "realname", "role", "status", "created_at", "updated_at").
		Find(&users).Error
	if err != nil {
		s.log.ErrorContext(ctx, "GetUserListByIDList failed", logger.Error(err))
		return nil, err
	}
	return users, nil
}

// UpdateCompetitionUserStatus 更新比赛用户状态
func (s *UserServiceImpl) UpdateCompetitionUserStatus(ctx context.Context, param *model.CompetitionUserListParam, status ojmodel.CompetitionUserStatus) error {
	err := s.db.WithContext(ctx).
		Model(&ojmodel.CompetitionUser{}).
		Where("competition_id = ?", param.CompetitionID).
		Where("user_id in ?", param.UserIDList).
		Update("status", status).Error
	if err != nil {
		return fmt.Errorf("UpdateCompetitionUserStatus failed: %w", err)
	}
	return nil
}
