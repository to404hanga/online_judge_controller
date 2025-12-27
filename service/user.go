package service

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	ojmodel "github.com/to404hanga/online_judge_common/model"
	"github.com/to404hanga/online_judge_controller/model"
	"github.com/to404hanga/online_judge_controller/pkg/pointer"
	"github.com/to404hanga/pkg404/gotools/retry"
	"github.com/to404hanga/pkg404/gotools/transform"
	"github.com/to404hanga/pkg404/logger"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const tokenVersionKey = "users:token_version:%d"

var defaultPassword = []byte("123456")

//go:embed lua/incr_token_version.lua
var incrTokenVersionScript string

type UserService interface {
	// GetRoleByID 获取用户角色
	GetRoleByID(ctx context.Context, userID uint64) (ojmodel.UserRole, error)
	// GetUserList 获取用户列表
	GetUserList(ctx context.Context, param *model.GetUserListParam) ([]ojmodel.User, int, error)
	// AddUsersToCompetition 添加用户到比赛名单
	AddUsersToCompetition(ctx context.Context, competitionID uint64, userMap map[uint64]*ojmodel.User, startTime time.Time) (int64, error)
	// GetUserListByUsernameList 获取用户列表, 根据学号全匹配, 仅返回正常用户
	GetUserListByUsernameList(ctx context.Context, usernameList []string) ([]ojmodel.User, error)
	// GetUserListByIDList 获取用户列表, 根据ID列表, 仅返回正常用户
	GetUserListByIDList(ctx context.Context, idList []uint64) ([]ojmodel.User, error)
	// UpdateCompetitionUserStatus 更新比赛用户状态
	UpdateCompetitionUserStatus(ctx context.Context, param *model.CompetitionUserListParam, status ojmodel.CompetitionUserStatus) error
	// GetAdminByID 获取管理员用户信息
	GetAdminByID(ctx context.Context, adminID uint64) (*ojmodel.User, error)
	// DeleteUserByID 删除用户
	DeleteUserByID(ctx context.Context, userID uint64) error
	// UpdateUser 更新用户
	UpdateUser(ctx context.Context, param *model.UpdateUserParam) error
	// ResetUserPassword 重置用户密码
	ResetUserPassword(ctx context.Context, userID uint64) error
	// UpdateUserPassword 更新用户密码
	UpdateUserPassword(ctx context.Context, userID uint64, password string) (bool, error)
	// GetCompetitionUserList 获取比赛用户列表
	GetCompetitionUserList(ctx context.Context, param *model.GetCompetitionUserListParam) ([]ojmodel.CompetitionUser, int, error)
	// CreateUser 创建用户
	CreateUser(ctx context.Context, username, realname string, role *ojmodel.UserRole) error
}

type UserServiceImpl struct {
	db  *gorm.DB
	rdb redis.Cmdable
	log loggerv2.Logger
}

func NewUserService(db *gorm.DB, rdb redis.Cmdable, log loggerv2.Logger) UserService {
	return &UserServiceImpl{
		db:  db,
		rdb: rdb,
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
func (s *UserServiceImpl) GetUserList(ctx context.Context, param *model.GetUserListParam) ([]ojmodel.User, int, error) {
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

	var total int64
	err := query.Count(&total).Error
	if err != nil {
		s.log.ErrorContext(ctx, "GetUserList failed", logger.Error(err))
		return nil, 0, err
	}

	orderBy := param.OrderBy
	if param.Desc {
		orderBy += " desc"
	}

	err = query.Order(orderBy).
		Offset((param.Page-1)*param.PageSize).
		Limit(param.PageSize).
		Select("id", "username", "realname", "role", "status", "created_at", "updated_at").
		Find(&users).Error
	if err != nil {
		s.log.ErrorContext(ctx, "GetUserList failed", logger.Error(err))
		return nil, int(total), err
	}
	return users, int(total), nil
}

// AddUsersToCompetition 添加用户到比赛名单
func (s *UserServiceImpl) AddUsersToCompetition(ctx context.Context, competitionID uint64, userMap map[uint64]*ojmodel.User, startTime time.Time) (int64, error) {
	competitionUser := transform.SliceFromMap(userMap, func(userID uint64, user *ojmodel.User) ojmodel.CompetitionUser {
		return ojmodel.CompetitionUser{
			CompetitionID: competitionID,
			UserID:        userID,
			Username:      user.Username,
			Realname:      user.Realname,
			Status:        pointer.ToPtr(ojmodel.CompetitionUserStatusNormal), // 默认正常状态
			StartTime:     startTime,
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

// GetAdminByID 获取管理员用户信息
func (s *UserServiceImpl) GetAdminByID(ctx context.Context, adminID uint64) (*ojmodel.User, error) {
	var admin ojmodel.User
	err := s.db.WithContext(ctx).
		Where("id = ?", adminID).
		Where("role = ?", ojmodel.UserRoleAdmin).
		Select("id", "username", "realname", "role", "status", "created_at", "updated_at").
		First(&admin).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("GetAdminByID failed: %w", err)
	}
	return &admin, nil
}

func (s *UserServiceImpl) DeleteUserByID(ctx context.Context, userID uint64) error {
	err := s.db.WithContext(ctx).
		Where("id = ?", userID).
		Delete(&ojmodel.User{}).Error
	if err != nil {
		return fmt.Errorf("DeleteUserByID failed: %w", err)
	}
	revokeCtx := context.WithValue(context.Background(), loggerv2.FieldsKey, ctx.Value(loggerv2.FieldsKey))
	s.revokeUserToken(revokeCtx, userID)
	return nil
}

func (s *UserServiceImpl) UpdateUser(ctx context.Context, param *model.UpdateUserParam) error {
	updates := map[string]any{}
	revoke := false
	if param.Realname != "" {
		updates["realname"] = param.Realname
	}
	if param.Status != nil {
		if *param.Status == ojmodel.UserStatusDisabled {
			revoke = true
		} else {
			retryCtx := context.WithValue(context.Background(), loggerv2.FieldsKey, ctx.Value(loggerv2.FieldsKey))
			cutoffTime := time.Now().Add(-8 * time.Hour) // 当前时间 - 8 小时，晚于这个时间开始的比赛，该用户都会被允许参赛
			retry.Do(retryCtx, func() error {
				errInternal := s.db.WithContext(retryCtx).Model(&ojmodel.CompetitionUser{}).
					Where("user_id = ?", param.UserID).
					Where("start_time >= ?", cutoffTime).
					Update("status", ojmodel.CompetitionUserStatusDisabled).Error
				if errInternal != nil {
					return fmt.Errorf("UpdateCompetitionUserStatus failed: %w", errInternal)
				}
				return nil
			}, retry.WithAsync(true), retry.WithCallback(func(err error) {
				if err != nil {
					s.log.ErrorContext(ctx, "UpdateCompetitionUserStatus failed", logger.Error(err))
				}
			}))
		}
		updates["status"] = param.Status.Int8()
	}

	err := s.db.WithContext(ctx).Model(&ojmodel.User{}).
		Where("id = ?", param.UserID).
		Updates(updates).Error
	if err != nil {
		return fmt.Errorf("UpdateUser failed: %w", err)
	}

	if revoke {
		revokeCtx := context.WithValue(context.Background(), loggerv2.FieldsKey, ctx.Value(loggerv2.FieldsKey))
		s.revokeUserToken(revokeCtx, param.UserID)
	}

	return nil
}

func (s *UserServiceImpl) revokeUserToken(ctx context.Context, userID uint64) {
	key := fmt.Sprintf(tokenVersionKey, userID)
	cutoffTime := time.Now().Add(-8 * time.Hour) // 当前时间 - 8 小时，晚于这个时间开始的比赛，该用户都会被禁用
	retry.Do(ctx, func() error {
		err := s.rdb.Eval(ctx, incrTokenVersionScript, []string{key}).Err()
		if err != nil {
			return fmt.Errorf("IncrTokenVersion failed: %w", err)
		}
		err = s.db.WithContext(ctx).Model(&ojmodel.CompetitionUser{}).
			Where("user_id = ?", userID).
			Where("start_time >= ?", cutoffTime).
			Update("status", ojmodel.CompetitionUserStatusDisabled).Error
		if err != nil {
			return fmt.Errorf("UpdateCompetitionUserStatus failed: %w", err)
		}
		return nil
	}, retry.WithAsync(true), retry.WithCallback(func(err error) {
		if err != nil {
			s.log.ErrorContext(ctx, "RevokeUserToken failed", logger.Error(err))
		}
	}))
}

func (s *UserServiceImpl) ResetUserPassword(ctx context.Context, userID uint64) error {
	hash, err := bcrypt.GenerateFromPassword(defaultPassword, bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("ResetUserPassword failed: %w", err)
	}
	err = s.db.WithContext(ctx).Model(&ojmodel.User{}).
		Where("id = ?", userID).
		Update("password", string(hash)).Error
	if err != nil {
		return fmt.Errorf("ResetUserPassword failed: %w", err)
	}
	return nil
}

func (s *UserServiceImpl) UpdateUserPassword(ctx context.Context, userID uint64, password string) (bool, error) {
	var user ojmodel.User
	err := s.db.WithContext(ctx).
		Where("id = ?", userID).
		Select("password").
		First(&user).Error
	if err != nil {
		return false, fmt.Errorf("CheckUserPassword failed: %w", err)
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return false, nil // 旧密码不匹配
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return true, fmt.Errorf("UpdateUserPassword failed: %w", err)
	}
	err = s.db.WithContext(ctx).Model(&ojmodel.User{}).
		Where("id = ?", userID).
		Update("password", string(hash)).Error
	if err != nil {
		return true, fmt.Errorf("UpdateUserPassword failed: %w", err)
	}
	return true, nil
}

func (s *UserServiceImpl) GetCompetitionUserList(ctx context.Context, param *model.GetCompetitionUserListParam) ([]ojmodel.CompetitionUser, int, error) {
	var userList []ojmodel.CompetitionUser
	var total int64

	query := s.db.WithContext(ctx).Model(&ojmodel.CompetitionUser{}).
		Where("competition_id = ?", param.CompetitionID)

	if param.Username != "" {
		query = query.Where("username LIKE ?", param.Username+"%") // 前缀匹配查询
	}
	if param.Realname != "" {
		query = query.Where("realname LIKE ?", "%"+param.Realname+"%") // 模糊查询
	}
	if param.Status != nil {
		query = query.Where("status = ?", param.Status.Int8())
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, fmt.Errorf("GetCompetitionUserList failed: %w", err)
	}

	if param.Desc {
		param.OrderBy += " desc"
	}

	err = query.Order(param.OrderBy).
		Limit(param.PageSize).
		Offset((param.Page - 1) * param.PageSize).
		Find(&userList).Error
	if err != nil {
		return nil, 0, fmt.Errorf("GetCompetitionUserList failed: %w", err)
	}

	return userList, int(total), nil
}

func (s *UserServiceImpl) CreateUser(ctx context.Context, username, realname string, role *ojmodel.UserRole) error {
	hash, err := bcrypt.GenerateFromPassword(defaultPassword, bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("CreateUser failed: %w", err)
	}

	err = s.db.WithContext(ctx).Model(&ojmodel.User{}).
		Create(&ojmodel.User{
			Username: username,
			Password: string(hash),
			Realname: realname,
			Role:     role,
			Status:   pointer.ToPtr(ojmodel.UserStatusNormal),
		}).Error
	if err != nil {
		return fmt.Errorf("CreateUser failed: %w", err)
	}
	return nil
}
