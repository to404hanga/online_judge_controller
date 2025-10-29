package service

import (
	"context"
	"fmt"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
	ojmodel "github.com/to404hanga/online_judge_common/model"
	"github.com/to404hanga/online_judge_controller/model"
	"github.com/to404hanga/pkg404/gotools/retry"
	"github.com/to404hanga/pkg404/logger"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
	"gorm.io/gorm"
)

type CompetitionService interface {
	// CreateCompetition 创建比赛
	CreateCompetition(ctx context.Context, param *model.CreateCompetitionParam) error
	// UpdateCompetition 更新比赛
	UpdateCompetition(ctx context.Context, param *model.UpdateCompetitionParam) error
	// AddCompetitionProblem 添加比赛题目
	AddCompetitionProblem(ctx context.Context, param *model.CompetitionProblemParam) error
	// RemoveCompetitionProblem 删除比赛题目
	RemoveCompetitionProblem(ctx context.Context, param *model.CompetitionProblemParam) error
	// EnableCompetitionProblem 启用比赛题目
	EnableCompetitionProblem(ctx context.Context, param *model.CompetitionProblemParam) error
	// DisableCompetitionProblem 禁用比赛题目
	DisableCompetitionProblem(ctx context.Context, param *model.CompetitionProblemParam) error
	// GetCompetitionProblemList 获取比赛题目列表
	GetCompetitionProblemList(ctx context.Context, competitionID uint64) ([]ojmodel.CompetitionProblem, error)
	// CheckUserInCompetition 检查用户是否在比赛名单中
	CheckUserInCompetition(ctx context.Context, competitionID, userID uint64) (bool, error)
	// CheckCompetitionTime 检查比赛时间是否在范围内
	CheckCompetitionTime(ctx context.Context, competitionID uint64) (bool, error)
	// GenerateCompetitionProblemPresignedURLList 生成比赛题目预签名 URL 列表
	GenerateCompetitionProblemPresignedURLList(ctx context.Context, competitionID uint64) ([]model.PresignedURL, error)
	// GetCompetitionProblemListWithPresignedURL 获取比赛题目列表（包含预签名 URL）
	GetCompetitionProblemListWithPresignedURL(ctx context.Context, competitionID uint64) ([]model.ProblemWithPresignedURL, error)
	// CheckCompetitionProblemExists 检查比赛题目是否存在
	CheckCompetitionProblemExists(ctx context.Context, competitionID, problemID uint64) (bool, error)
}

type CompetitionServiceImpl struct {
	db  *gorm.DB
	rdb redis.Cmdable
	log loggerv2.Logger
}

var _ CompetitionService = (*CompetitionServiceImpl)(nil)

func NewCompetitionService(db *gorm.DB, rdb redis.Cmdable, log loggerv2.Logger) CompetitionService {
	return &CompetitionServiceImpl{
		db:  db,
		rdb: rdb,
		log: log,
	}
}

// CreateCompetition 创建比赛
func (s *CompetitionServiceImpl) CreateCompetition(ctx context.Context, param *model.CreateCompetitionParam) error {
	tx := s.db.WithContext(ctx).Begin()
	competition := &ojmodel.Competition{
		Name:      param.Name,
		StartTime: param.StartTime,
		EndTime:   param.EndTime,
		CreatorID: param.Operator,
		UpdaterID: param.Operator,
	}
	err := tx.Create(&competition).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("CreateCompetition transaction failed at insert into competition: %w", err)
	}

	if len(param.Problems) != 0 {
		competitionProblems := make([]ojmodel.CompetitionProblem, 0, len(param.Problems))
		for _, problem := range param.Problems {
			competitionProblems = append(competitionProblems, ojmodel.CompetitionProblem{
				CompetitionID: competition.ID,
				ProblemID:     problem,
			})
		}
		err = tx.Create(&competitionProblems).Error
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("CreateCompetition transaction failed at insert into competition_problem: %w", err)
		}
	}

	// 提交事务
	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("CreateCompetition transaction failed at commit: %w", err)
	}
	return nil
}

// UpdateCompetition 更新比赛
func (s *CompetitionServiceImpl) UpdateCompetition(ctx context.Context, param *model.UpdateCompetitionParam) error {
	updates := map[string]any{
		"updater_id": param.Operator,
	}
	if param.Name != nil {
		updates["name"] = *param.Name
	}
	if param.StartTime != nil {
		updates["start_time"] = *param.StartTime
	}
	if param.EndTime != nil {
		updates["end_time"] = *param.EndTime
	}
	if param.Status != nil {
		updates["status"] = *param.Status
	}

	// 检查是否有更新
	if len(updates) == 1 {
		return nil
	}

	// 更新比赛
	err := s.db.WithContext(ctx).Model(&ojmodel.Competition{}).
		Where("id = ?", param.ID).
		Updates(updates).Error
	if err != nil {
		return fmt.Errorf("UpdateCompetition failed at update competition: %w", err)
	}
	return nil
}

// AddCompetitionProblem 添加比赛题目
func (s *CompetitionServiceImpl) AddCompetitionProblem(ctx context.Context, param *model.CompetitionProblemParam) error {
	competitionProblems := make([]ojmodel.CompetitionProblem, 0, len(param.ProblemIDs))
	for _, problemID := range param.ProblemIDs {
		competitionProblems = append(competitionProblems, ojmodel.CompetitionProblem{
			CompetitionID: param.CompetitionID,
			ProblemID:     problemID,
		})
	}
	err := s.db.WithContext(ctx).Create(&competitionProblems).Error
	if err != nil {
		return fmt.Errorf("AddCompetitionProblem failed at insert into competition_problem: %w", err)
	}
	return nil
}

// RemoveCompetitionProblem 删除比赛题目
func (s *CompetitionServiceImpl) RemoveCompetitionProblem(ctx context.Context, param *model.CompetitionProblemParam) error {
	err := s.db.WithContext(ctx).
		Where("competition_id = ?", param.CompetitionID).
		Where("problem_id IN ?", param.ProblemIDs).
		Delete(&ojmodel.CompetitionProblem{}).Error
	if err != nil {
		return fmt.Errorf("RemoveCompetitionProblem failed at delete from competition_problem: %w", err)
	}
	return nil
}

// EnableCompetitionProblem 启用比赛题目
func (s *CompetitionServiceImpl) EnableCompetitionProblem(ctx context.Context, param *model.CompetitionProblemParam) error {
	err := s.db.WithContext(ctx).
		Where("competition_id = ?", param.CompetitionID).
		Where("problem_id IN ?", param.ProblemIDs).
		Updates(map[string]any{
			"status": ojmodel.CompetitionProblemStatusEnabled,
		}).Error
	if err != nil {
		return fmt.Errorf("EnableCompetitionProblem failed at update competition_problem: %w", err)
	}
	return nil
}

// DisableCompetitionProblem 禁用比赛题目
func (s *CompetitionServiceImpl) DisableCompetitionProblem(ctx context.Context, param *model.CompetitionProblemParam) error {
	err := s.db.WithContext(ctx).
		Where("competition_id = ?", param.CompetitionID).
		Where("problem_id IN ?", param.ProblemIDs).
		Updates(map[string]any{
			"status": ojmodel.CompetitionProblemStatusDisabled,
		}).Error
	if err != nil {
		return fmt.Errorf("DisableCompetitionProblem failed at update competition_problem: %w", err)
	}
	return nil
}

// GetCompetitionProblemList 获取比赛题目列表
func (s *CompetitionServiceImpl) GetCompetitionProblemList(ctx context.Context, competitionID uint64) ([]ojmodel.CompetitionProblem, error) {
	competitionProblems := make([]ojmodel.CompetitionProblem, 0)
	err := s.db.WithContext(ctx).
		Where("competition_id = ?", competitionID).
		Find(&competitionProblems).Error
	if err != nil {
		return nil, fmt.Errorf("GetCompetitionProblemList failed at select from competition_problem: %w", err)
	}
	return competitionProblems, nil
}

// CheckUserInCompetition 检查用户是否在比赛名单中
func (s *CompetitionServiceImpl) CheckUserInCompetition(ctx context.Context, competitionID, userID uint64) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Model(&ojmodel.CompetitionUser{}).
		Where("competition_id = ?", competitionID).
		Where("user_id = ?", userID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("CheckUserInCompetition failed at select from competition_user: %w", err)
	}
	return count > 0, nil
}

// CheckCompetitionTime 检查比赛时间是否在范围内
func (s *CompetitionServiceImpl) CheckCompetitionTime(ctx context.Context, competitionID uint64) (bool, error) {
	var competition ojmodel.Competition
	err := s.db.WithContext(ctx).
		Where("id = ?", competitionID).
		Select("start_time, end_time").
		First(&competition).Error
	if err != nil {
		return false, fmt.Errorf("CheckCompetitionTime failed at select from competition: %w", err)
	}
	return !time.Now().Before(competition.StartTime) && time.Now().Before(competition.EndTime), nil
}

// GenerateCompetitionProblemPresignedURLList 生成比赛题目预签名 URL 列表
func (s *CompetitionServiceImpl) GenerateCompetitionProblemPresignedURLList(ctx context.Context, competitionID uint64) ([]model.PresignedURL, error) {
	var urls []model.PresignedURL
	err := s.db.WithContext(ctx).
		Table(fmt.Sprintf("%s competition_problem", ojmodel.CompetitionProblem{}.TableName())).
		Select("p.id as id, p.description_url as url").
		Joins(fmt.Sprintf("JOIN %s p ON cp.problem_id = p.id", ojmodel.Problem{}.TableName())).
		Where("cp.competition_id = ?", competitionID).
		Where("cp.status = ?", ojmodel.CompetitionProblemStatusEnabled).
		Where("p.status = ?", ojmodel.ProblemStatusPublished).
		Scan(&urls).Error
	if err != nil {
		return nil, fmt.Errorf("GenerateCompetitionProblemPresignedURLList failed at select from competition_problem: %w", err)
	}

	for _, url := range urls {
		err = retry.Do(ctx, func() error {
			_, errInternal := s.rdb.Set(ctx, fmt.Sprintf("competition_problem:%d:%d", competitionID, url.ID), true, 8*time.Hour).Result()
			return errInternal
		})
		if err != nil {
			s.log.WarnContext(ctx, "GenerateCompetitionProblemPresignedURLList failed at set redis", logger.Error(err))
		}
	}

	urlsByte, _ := json.Marshal(urls)
	err = s.rdb.Set(ctx, fmt.Sprintf("competition_problem_presigned_url_list:%d", competitionID), string(urlsByte), 8*time.Hour).Err()
	if err != nil {
		return nil, fmt.Errorf("GenerateCompetitionProblemPresignedURLList failed at set redis: %w", err)
	}

	return urls, nil
}

// GetCompetitionProblemListWithPresignedURL 获取比赛题目列表（包含预签名 URL）
func (s *CompetitionServiceImpl) GetCompetitionProblemListWithPresignedURL(ctx context.Context, competitionID uint64) ([]model.ProblemWithPresignedURL, error) {
	var problems []model.ProblemWithPresignedURL
	err := s.db.WithContext(ctx).
		Table(fmt.Sprintf("%s competition_problem", ojmodel.CompetitionProblem{}.TableName())).
		Select("p.id as id, p.title as title, p.time_limit as time_limit, p.memory_limit as memory_limit").
		Joins(fmt.Sprintf("JOIN %s p ON cp.problem_id = p.id", ojmodel.Problem{}.TableName())).
		Where("cp.competition_id = ?", competitionID).
		Where("cp.status = ?", ojmodel.CompetitionProblemStatusEnabled).
		Where("p.status = ?", ojmodel.ProblemStatusPublished).
		Scan(&problems).Error
	if err != nil {
		return nil, fmt.Errorf("GetCompetitionProblemListWithPresignedURL failed at select from competition_problem: %w", err)
	}

	var urls []model.PresignedURL
	urlsByte, err := s.rdb.Get(ctx, fmt.Sprintf("competition_problem_presigned_url_list:%d", competitionID)).Bytes()
	if err != nil {
		s.log.ErrorContext(ctx, "GetCompetitionProblemListWithPresignedURL failed at get redis", logger.Error(err))
		urls, err = s.GenerateCompetitionProblemPresignedURLList(ctx, competitionID)
		if err != nil {
			return nil, fmt.Errorf("GetCompetitionProblemListWithPresignedURL failed at generate presigned url list: %w", err)
		}
	} else {
		err = json.Unmarshal(urlsByte, &urls)
		if err != nil {
			return nil, fmt.Errorf("GetCompetitionProblemListWithPresignedURL failed at unmarshal redis: %w", err)
		}
	}

	for idx := range problems {
		for _, url := range urls {
			if problems[idx].ID == url.ID {
				problems[idx].PresignedURL = url.URL
				break
			}
		}
	}

	return problems, nil
}

// CheckCompetitionProblemExists 检查比赛题目是否存在
func (s *CompetitionServiceImpl) CheckCompetitionProblemExists(ctx context.Context, competitionID, problemID uint64) (bool, error) {
	// 检查比赛题目是否存在
	exists, err := s.rdb.Exists(ctx, fmt.Sprintf("competition_problem:%d:%d", competitionID, problemID)).Result()
	if err != nil {
		return false, fmt.Errorf("CheckCompetitionProblemExists failed at check competition problem: %w", err)
	}
	if exists > 0 {
		return true, nil
	}

	ok, err := s.rdb.SetNX(ctx, fmt.Sprintf("competition_problem_mutex:%d:%d", competitionID, problemID), 1, 2*time.Second).Result()
	if err != nil {
		return false, fmt.Errorf("CheckCompetitionProblemExists failed at check competition problem: %w", err)
	}
	if !ok {
		time.Sleep(3 * time.Second)
		return s.CheckCompetitionProblemExists(ctx, competitionID, problemID)
	}
	defer func() {
		err = retry.Do(ctx, func() error {
			return s.rdb.Del(ctx, fmt.Sprintf("competition_problem_mutex:%d:%d", competitionID, problemID)).Err()
		})
		if err != nil {
			s.log.ErrorContext(ctx, "CheckCompetitionProblemExists failed at del competition problem mutex")
		}
	}()

	var cnt int64
	err = s.db.WithContext(ctx).Model(&ojmodel.CompetitionProblem{}).
		Where("competition_id = ?", competitionID).
		Where("problem_id = ?", problemID).
		Where("status = ?", ojmodel.CompetitionProblemStatusEnabled).
		Count(&cnt).Error
	if err != nil {
		return false, fmt.Errorf("CheckCompetitionProblemExists failed at check competition problem: %w", err)
	}
	if cnt > 0 {
		s.rdb.Set(ctx, fmt.Sprintf("competition_problem:%d:%d", competitionID, problemID), true, 8*time.Hour)
		return true, nil
	}
	return false, nil
}
