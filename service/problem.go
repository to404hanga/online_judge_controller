package service

import (
	"context"
	"fmt"

	ojmodel "github.com/to404hanga/online_judge_common/model"
	"github.com/to404hanga/online_judge_controller/model"
	"github.com/to404hanga/online_judge_controller/pkg/pointer"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
	"gorm.io/gorm"
)

type ProblemService interface {
	// CreateProblem 创建题目
	CreateProblem(ctx context.Context, param *model.CreateProblemParam) error
	// UpdateProblem 更新题目
	UpdateProblem(ctx context.Context, param *model.UpdateProblemParam) error
	// CheckExistByDescriptionURL 检查题目描述 URL 是否存在
	CheckExistByDescriptionURL(ctx context.Context, descriptionURL string) (bool, error)
	// CheckExistByTestcaseZipURL 检查测试用例压缩包 URL 是否存在
	CheckExistByTestcaseZipURL(ctx context.Context, testcaseZipURL string) (bool, error)
	// GetProblemByID 获取题目
	GetProblemByID(ctx context.Context, problemID uint64) (*ojmodel.Problem, error)
	// GetProblemList 获取题目列表
	GetProblemList(ctx context.Context, param *model.GetProblemListParam) ([]ojmodel.Problem, error)
}

type ProblemServiceImpl struct {
	db  *gorm.DB
	log loggerv2.Logger
}

var _ ProblemService = (*ProblemServiceImpl)(nil)

func NewProblemService(db *gorm.DB, log loggerv2.Logger) ProblemService {
	return &ProblemServiceImpl{
		db:  db,
		log: log,
	}
}

// CreateProblem 创建题目
func (s *ProblemServiceImpl) CreateProblem(ctx context.Context, param *model.CreateProblemParam) error {
	problem := ojmodel.Problem{
		Title:          param.Title,
		DescriptionURL: param.DescriptionURL,
		TestcaseZipURL: param.TestcaseZipURL,
		Visible:        pointer.ToPtr(ojmodel.ProblemVisible(param.Visible)),
		TimeLimit:      param.TimeLimit,
		MemoryLimit:    param.MemoryLimit,
		CreatorID:      param.Operator,
		UpdaterID:      param.Operator,
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
	if param.DescriptionURL != nil {
		updates["description_url"] = *param.DescriptionURL
	}
	if param.TestcaseZipURL != nil {
		updates["testcase_zip_url"] = *param.TestcaseZipURL
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
	}

	return nil
}

// CheckExistByDescriptionURL 检查题目描述 URL 是否存在
func (s *ProblemServiceImpl) CheckExistByDescriptionURL(ctx context.Context, descriptionURL string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&ojmodel.Problem{}).
		Where("description_url = ?", descriptionURL).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("CheckExistByDescriptionURL failed: %w", err)
	}
	return count > 0, nil
}

// CheckExistByTestcaseZipURL 检查测试用例压缩包 URL 是否存在
func (s *ProblemServiceImpl) CheckExistByTestcaseZipURL(ctx context.Context, testcaseZipURL string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&ojmodel.Problem{}).
		Where("testcase_zip_url = ?", testcaseZipURL).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("CheckExistByTestcaseZipURL failed: %w", err)
	}
	return count > 0, nil
}

// GetProblemByID 获取题目
func (s *ProblemServiceImpl) GetProblemByID(ctx context.Context, problemID uint64) (*ojmodel.Problem, error) {
	var problem ojmodel.Problem
	err := s.db.WithContext(ctx).Model(&ojmodel.Problem{}).
		Where("id = ?", problemID).
		First(&problem).Error
	if err != nil {
		return nil, fmt.Errorf("GetProblemByID failed: %w", err)
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
	if param.Desc {
		query = query.Order("id DESC")
	} else {
		query = query.Order("id ASC")
	}
	if param.PageSize > 0 {
		query = query.Limit(param.PageSize)
	}
	if param.Page > 0 {
		query = query.Offset((param.Page - 1) * param.PageSize)
	}

	err := query.Find(&problems).Error
	if err != nil {
		return nil, fmt.Errorf("GetProblemList failed: %w", err)
	}

	return problems, nil
}
