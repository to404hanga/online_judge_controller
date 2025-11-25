package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
	ojmodel "github.com/to404hanga/online_judge_common/model"
	"github.com/to404hanga/online_judge_controller/model"
	"github.com/to404hanga/online_judge_controller/pkg/pointer"
	"github.com/to404hanga/pkg404/gotools/retry"
	"github.com/to404hanga/pkg404/logger"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
	"gorm.io/gorm"
)

type ProblemService interface {
	// CreateProblem 创建题目
	CreateProblem(ctx context.Context, param *model.CreateProblemParam) error
	// UpdateProblem 更新题目
	UpdateProblem(ctx context.Context, param *model.UpdateProblemParam) error
	// // CheckExistByTestcaseZipURL 检查测试用例压缩包 URL 是否存在
	// CheckExistByTestcaseZipURL(ctx context.Context, testcaseZipURL string) (bool, error)
	// GetProblemByID 获取题目
	GetProblemByID(ctx context.Context, problemID, competitionID uint64) (*ojmodel.Problem, error)
	// GetProblemList 获取题目列表
	GetProblemList(ctx context.Context, param *model.GetProblemListParam) ([]ojmodel.Problem, error)
}

const (
	problemKey            = "problem:%d"
	competitionProblemKey = "problem:%d:competition:%d"
)

type ProblemServiceImpl struct {
	db  *gorm.DB
	rdb redis.Cmdable
	log loggerv2.Logger
}

var _ ProblemService = (*ProblemServiceImpl)(nil)

func NewProblemService(db *gorm.DB, rdb redis.Cmdable, log loggerv2.Logger) ProblemService {
	return &ProblemServiceImpl{
		db:  db,
		rdb: rdb,
		log: log,
	}
}

// CreateProblem 创建题目
func (s *ProblemServiceImpl) CreateProblem(ctx context.Context, param *model.CreateProblemParam) error {
	problem := ojmodel.Problem{
		Title:       param.Title,
		Description: param.Description,
		Visible:     pointer.ToPtr(ojmodel.ProblemVisible(*param.Visible)),
		TimeLimit:   param.TimeLimit,
		MemoryLimit: param.MemoryLimit,
		Status:      pointer.ToPtr(ojmodel.ProblemStatusUnpublished),
		CreatorID:   param.Operator,
		UpdaterID:   param.Operator,
	}
	err := s.db.WithContext(ctx).Create(&problem).Error
	if err != nil {
		return fmt.Errorf("CreateProblem failed: %w", err)
	}
	return nil
}

// UpdateProblem 更新题目
func (s *ProblemServiceImpl) UpdateProblem(ctx context.Context, param *model.UpdateProblemParam) error {
	updates := map[string]any{
		"updater_id": param.Operator,
	}
	if param.Title != nil {
		updates["title"] = *param.Title
	}
	if param.Description != nil {
		updates["description"] = *param.Description
	}
	if param.Status != nil {
		updates["status"] = *param.Status
	}
	if param.TimeLimit != nil {
		updates["time_limit"] = *param.TimeLimit
	}
	if param.MemoryLimit != nil {
		updates["memory_limit"] = *param.MemoryLimit
	}
	if param.Visible != nil {
		updates["visible"] = *param.Visible
	}

	// 如果有修改内容再执行, 否则直接返回
	if len(updates) > 1 {
		err := s.db.WithContext(ctx).Model(&ojmodel.Problem{}).
			Where("id = ?", param.ProblemID).
			Updates(updates).Error
		if err != nil {
			return fmt.Errorf("UpdateProblem failed: %w", err)
		}
		// 异步删除 redis 缓存, 带重试
		retryCtx := context.WithValue(context.Background(), loggerv2.FieldsKey, ctx.Value(loggerv2.FieldsKey))
		retry.Do(retryCtx, func() error {
			// 使用 Unlink 命令删除缓存, 可以异步执行, 不阻塞主流程
			return s.rdb.Unlink(retryCtx, fmt.Sprintf(problemKey, param.ProblemID)).Err()
		}, retry.WithAsync(true), retry.WithCallback(func(err error) {
			s.log.ErrorContext(retryCtx, "UpdateProblem: failed to delete cache", logger.Error(err))
		}))
	}

	return nil
}

// // CheckExistByTestcaseZipURL 检查测试用例压缩包 URL 是否存在
// func (s *ProblemServiceImpl) CheckExistByTestcaseZipURL(ctx context.Context, testcaseZipURL string) (bool, error) {
// 	var count int64
// 	err := s.db.WithContext(ctx).Model(&ojmodel.Problem{}).
// 		Where("testcase_zip_url = ?", testcaseZipURL).
// 		Count(&count).Error
// 	if err != nil {
// 		return false, fmt.Errorf("CheckExistByTestcaseZipURL failed: %w", err)
// 	}
// 	return count > 0, nil
// }

// GetProblemByID 获取题目
func (s *ProblemServiceImpl) GetProblemByID(ctx context.Context, problemID, competitionID uint64) (*ojmodel.Problem, error) {
	var problem ojmodel.Problem

	// 如果是非赛时可见的题目, 其键为 problem:%d:0
	key := fmt.Sprintf(competitionProblemKey, problemID, competitionID)
	problemBytes, err := s.rdb.Get(ctx, key).Bytes()
	if err == nil {
		if err = json.Unmarshal(problemBytes, &problem); err == nil {
			return &problem, nil
		}
	}

	s.log.WarnContext(ctx, "GetProblemByID from redis failed", logger.Error(err))

	// 分布式锁，防止缓存击穿
	lockKey := fmt.Sprintf("lock:problem:%d", problemID)
	// 过期时间设置为 10s
	ok, err := s.rdb.SetNX(ctx, lockKey, "locked", 10*time.Second).Result()
	if err != nil {
		return nil, fmt.Errorf("GetProblemByID failed to set lock: %w", err)
	}

	// 没有获得锁，说明有其他人在操作
	if !ok {
		// 等待 1s 后重试
		time.Sleep(1 * time.Second)
		return s.GetProblemByID(ctx, problemID, competitionID)
	}
	defer func() {
		retryCtx := context.WithValue(context.Background(), loggerv2.FieldsKey, ctx.Value(loggerv2.FieldsKey))
		retry.Do(retryCtx, func() error {
			return s.rdb.Del(retryCtx, lockKey).Err()
		}, retry.WithAsync(true), retry.WithCallback(func(err error) {
			s.log.ErrorContext(retryCtx, "GetProblemByID: failed to delete lock", logger.Error(err))
		}))
	}()

	// 获得锁后，再次尝试从 redis 中获取
	problemBytes, err = s.rdb.Get(ctx, key).Bytes()
	if err == nil {
		if err = json.Unmarshal(problemBytes, &problem); err == nil {
			return &problem, nil
		}
	}
	s.log.WarnContext(ctx, "GetProblemByID from redis recheck failed", logger.Error(err))

	// 校验题目是否在比赛中启用
	if competitionID != 0 {
		var cnt int64
		err = s.db.WithContext(ctx).Model(&ojmodel.CompetitionProblem{}).
			Where("competition_id = ?", competitionID).
			Where("problem_id = ?", problemID).
			Where("status = ?", ojmodel.CompetitionProblemStatusEnabled).
			Count(&cnt).Error
		if err != nil {
			return nil, fmt.Errorf("GetProblemByID failed: %w", err)
		}
		if cnt == 0 {
			return nil, fmt.Errorf("GetProblemByID failed: problem %d not enabled in competition %d", problemID, competitionID)
		}
	}

	query := s.db.WithContext(ctx).Model(&ojmodel.Problem{}).
		Where("id = ?", problemID)
	if competitionID == 0 {
		// 没有传入 competitionID, 只允许查看非比赛期间可见的题目
		query = query.Where("visible = ?", ojmodel.ProblemVisibleTrue)
	}

	err = query.First(&problem).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("GetProblemByID failed: %w", err)
	}

	// 存入 redis, 找不到数据存入一份空数据, 防止缓存击穿
	problemBytes, err = json.Marshal(problem)
	if err != nil {
		s.log.ErrorContext(ctx, "GetProblemByID: failed to marshal problem", logger.Error(err))
	} else {
		// 过期时间设置为 8h
		s.rdb.Set(ctx, key, problemBytes, 8*time.Hour)
	}
	return &problem, nil
}

// GetProblemList 获取题目列表
func (s *ProblemServiceImpl) GetProblemList(ctx context.Context, param *model.GetProblemListParam) ([]ojmodel.Problem, error) {
	var problems []ojmodel.Problem
	query := s.db.WithContext(ctx).Model(&ojmodel.Problem{})
	if param.Title != "" {
		query = query.Where("title LIKE ?", "%"+param.Title+"%")
	}
	if param.Status != nil {
		query = query.Where("status = ?", *param.Status)
	}
	if param.Visible != nil {
		query = query.Where("visible = ?", *param.Visible)
	}
	if param.TimeLimit != nil {
		query = query.Where("time_limit = ?", *param.TimeLimit)
	}
	if param.MemoryLimit != nil {
		query = query.Where("memory_limit = ?", *param.MemoryLimit)
	}
	orderBy := "id"
	if param.OrderBy != "" {
		orderBy = param.OrderBy
	}
	if param.Desc {
		query = query.Order(orderBy + " DESC")
	} else {
		query = query.Order(orderBy + " ASC")
	}

	err := query.Limit(param.PageSize).
		Offset((param.Page - 1) * param.PageSize).
		Find(&problems).Error
	if err != nil {
		return nil, fmt.Errorf("GetProblemList failed: %w", err)
	}

	return problems, nil
}
