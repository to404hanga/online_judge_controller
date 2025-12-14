package service

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	ojmodel "github.com/to404hanga/online_judge_common/model"
	"github.com/to404hanga/online_judge_controller/model"
	"github.com/to404hanga/online_judge_controller/pkg/pointer"
	"github.com/to404hanga/pkg404/gotools/retry"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
	"gorm.io/gorm"
)

type ProblemService interface {
	// CreateProblem 创建题目
	CreateProblem(ctx context.Context, param *model.CreateProblemParam) error
	// UpdateProblem 更新题目
	UpdateProblem(ctx context.Context, param *model.UpdateProblemParam) error
	// GetProblemByID 获取题目
	GetProblemByID(ctx context.Context, problemID uint64) (*ojmodel.Problem, error)
	// GetProblemList 获取题目列表
	GetProblemList(ctx context.Context, param *model.GetProblemListParam) ([]ojmodel.Problem, int, error)
}

const (
	problemKey            = "problem:%d"
	competitionProblemKey = "problem:%d:competition:%d"
)

//go:embed lua/unlink_problem.lua
var unlinkProblemScript string

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

		// 同步删除 redis 缓存, 带重试
		err = retry.Do(ctx, func() error {
			pattern := fmt.Sprintf(problemKey+"*", param.ProblemID)
			return s.rdb.Eval(ctx, unlinkProblemScript, []string{}, pattern).Err()
		})
		if err != nil {
			return fmt.Errorf("UpdateProblem failed: %w", err)
		}
	}

	return nil
}

// GetProblemByID 获取题目(管理员使用)
func (s *ProblemServiceImpl) GetProblemByID(ctx context.Context, problemID uint64) (*ojmodel.Problem, error) {
	var problem ojmodel.Problem
	err := s.db.WithContext(ctx).Model(&ojmodel.Problem{}).
		Where("id = ?", problemID).First(&problem).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("GetProblemByID failed: %w", err)
	}
	return &problem, nil
}

// GetProblemList 获取题目列表
func (s *ProblemServiceImpl) GetProblemList(ctx context.Context, param *model.GetProblemListParam) ([]ojmodel.Problem, int, error) {
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

	var count int64
	err := query.Count(&count).Error
	if err != nil {
		return nil, 0, fmt.Errorf("GetProblemList failed: %w", err)
	}

	err = query.Limit(param.PageSize).
		Offset((param.Page - 1) * param.PageSize).
		Find(&problems).Error
	if err != nil {
		return nil, 0, fmt.Errorf("GetProblemList failed: %w", err)
	}

	return problems, int(count), nil
}
