package service

import (
	"context"
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
}

const (
	competitionProblemListKey   = "competition:%d:problem:list"
	competitionUserSetKey       = "competition:%d:user:set"
	competitionUserSetLoadedKey = "competition:%d:user:set:loaded"
	competitionMetaKey          = "competition:%d:meta"
)

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
		Status:    pointer.ToPtr(ojmodel.CompetitionStatusUnpublished),
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
			problemTitle := ""
			err = tx.Model(&ojmodel.Problem{}).
				Where("id = ?", problem).
				Select("title").
				Scan(&problemTitle).Error
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("CreateCompetition transaction failed at query problem title: %w", err)
			}
			if problemTitle == "" {
				tx.Rollback()
				return fmt.Errorf("CreateCompetition transaction failed: problem %d not found", problem)
			}
			competitionProblems = append(competitionProblems, ojmodel.CompetitionProblem{
				CompetitionID: competition.ID,
				ProblemID:     problem,
				ProblemTitle:  problemTitle,
				Status:        pointer.ToPtr(ojmodel.CompetitionProblemStatusEnabled),
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
		problemTitle := ""
		err := s.db.WithContext(ctx).
			Model(&ojmodel.Problem{}).
			Where("id = ?", problemID).
			Select("title").
			Scan(&problemTitle).Error
		if err != nil {
			return fmt.Errorf("AddCompetitionProblem failed at query problem title: %w", err)
		}
		if problemTitle == "" {
			return fmt.Errorf("AddCompetitionProblem failed: problem %d not found", problemID)
		}
		competitionProblems = append(competitionProblems, ojmodel.CompetitionProblem{
			CompetitionID: param.CompetitionID,
			ProblemID:     problemID,
			ProblemTitle:  problemTitle,
			Status:        pointer.ToPtr(ojmodel.CompetitionProblemStatusEnabled),
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
	competitionProblems := make([]ojmodel.CompetitionProblem, 0, 10)

	competitionProblemsBytes, err := s.rdb.Get(ctx, fmt.Sprintf(competitionProblemListKey, competitionID)).Bytes()
	if err == nil {
		if err = json.Unmarshal(competitionProblemsBytes, &competitionProblems); err == nil {
			return competitionProblems, nil
		}
	}
	s.log.WarnContext(ctx, "GetCompetitionProblemList from redis failed", logger.Error(err))

	// 分布式锁，防止缓存击穿
	lockKey := fmt.Sprintf("lock:competition:%d:problem:list", competitionID)
	// 过期时间设置为 10s
	ok, err := s.rdb.SetNX(ctx, lockKey, "locked", 10*time.Second).Result()
	if err != nil {
		return nil, fmt.Errorf("GetCompetitionProblemList failed to set lock: %w", err)
	}

	// 没有获得锁，说明有其他人在操作
	if !ok {
		// 等待 1s 后重试
		time.Sleep(1 * time.Second)
		return s.GetCompetitionProblemList(ctx, competitionID)
	}
	defer retry.Do(ctx, func() error {
		return s.rdb.Del(ctx, lockKey).Err()
	}, retry.WithAsync(true), retry.WithCallback(func(err error) {
		s.log.ErrorContext(ctx, "GetCompetitionProblemList: failed to delete lock", logger.Error(err))
	}))

	// 获得锁后, 再次尝试从 redis 中获取
	competitionProblemsBytes, err = s.rdb.Get(ctx, fmt.Sprintf(competitionProblemListKey, competitionID)).Bytes()
	if err == nil {
		if err = json.Unmarshal(competitionProblemsBytes, &competitionProblems); err == nil {
			return competitionProblems, nil
		}
	}
	s.log.WarnContext(ctx, "GetCompetitionProblemList from redis recheck failed", logger.Error(err))

	err = s.db.WithContext(ctx).
		Where("competition_id = ?", competitionID).
		Find(&competitionProblems).Error
	if err != nil {
		return nil, fmt.Errorf("GetCompetitionProblemList failed at select from competition_problem: %w", err)
	}

	// 存入 redis
	competitionProblemsBytes, err = json.Marshal(competitionProblems)
	if err == nil {
		// 过期时间设置为 8h
		s.rdb.Set(ctx, fmt.Sprintf(competitionProblemListKey, competitionID), competitionProblemsBytes, 8*time.Hour)
	}
	return competitionProblems, nil
}

// CheckUserInCompetition 检查用户是否在比赛名单中
func (s *CompetitionServiceImpl) CheckUserInCompetition(ctx context.Context, competitionID, userID uint64) (bool, error) {
	userSetKey := fmt.Sprintf(competitionUserSetKey, competitionID)

	// 1. 检查用户是否在 Set 缓存中
	isMember, err := s.rdb.SIsMember(ctx, userSetKey, userID).Result()
	if err != nil && err != redis.Nil {
		s.log.WarnContext(ctx, "CheckUserInCompetition: failed to check user in redis set", logger.Error(err))
		// Redis 查询失败，降级到数据库查询
	} else if isMember {
		// 缓存命中，用户在白名单内
		return true, nil
	}

	// 2. 检查 'loaded' 标记是否存在，防止缓存穿透
	loadedKey := fmt.Sprintf(competitionUserSetLoadedKey, competitionID)
	loaded, err := s.rdb.Exists(ctx, loadedKey).Result()
	if err != nil && err != redis.Nil {
		s.log.WarnContext(ctx, "CheckUserInCompetition: failed to check loaded flag", logger.Error(err))
	}
	if loaded > 0 {
		// 'loaded' 标记存在，但用户不在 Set 中，说明用户确实不在白名单内
		return false, nil
	}

	// 3. 缓存未命中且 'loaded' 标记不存在（可能预热失败），启动分布式锁进行缓存重建
	lockKey := fmt.Sprintf("lock:competition:%d:user:set:load", competitionID)
	ok, err := s.rdb.SetNX(ctx, lockKey, "locked", 10*time.Second).Result()
	if err != nil {
		return false, fmt.Errorf("CheckUserInCompetition: failed to set lock: %w", err)
	}

	if !ok {
		// 未获取到锁，休眠后重试，让持有锁的协程完成缓存加载
		time.Sleep(1 * time.Second)
		return s.CheckUserInCompetition(ctx, competitionID, userID)
	}
	defer retry.Do(ctx, func() error {
		return s.rdb.Del(ctx, lockKey).Err()
	}, retry.WithAsync(true), retry.WithCallback(func(err error) {
		s.log.ErrorContext(ctx, "CheckUserInCompetition: failed to delete lock", logger.Error(err))
	}))

	// 4. 获取锁后，再次检查缓存，防止在等待锁期间缓存已被其他协程加载
	isMember, _ = s.rdb.SIsMember(ctx, userSetKey, userID).Result()
	if isMember {
		return true, nil
	}
	loaded, _ = s.rdb.Exists(ctx, loadedKey).Result()
	if loaded > 0 {
		return false, nil
	}

	// 5. 从数据库加载所有比赛用户
	var users []ojmodel.CompetitionUser
	err = s.db.WithContext(ctx).Model(&ojmodel.CompetitionUser{}).
		Where("competition_id = ?", competitionID).
		Find(&users).Error
	if err != nil {
		return false, fmt.Errorf("CheckUserInCompetition: failed to load users from db: %w", err)
	}

	// 6. 将用户批量写入 Redis Set 并设置 'loaded' 标记
	pipe := s.rdb.Pipeline()
	userInDB := false
	if len(users) > 0 {
		for _, user := range users {
			pipe.SAdd(ctx, userSetKey, user.UserID)
			if user.UserID == userID {
				userInDB = true
			}
		}
	}
	// 写入 'loaded' 标记，即使比赛没有用户（空集合），也要标记为已加载，防止后续请求穿透
	pipe.Set(ctx, loadedKey, "1", 8*time.Hour)
	_, err = pipe.Exec(ctx)
	if err != nil {
		s.log.ErrorContext(ctx, "CheckUserInCompetition: failed to execute redis pipeline", logger.Error(err))
		// 即使缓存写入失败，本次检查的正确性由 userInDB 保证
	}

	return userInDB, nil
}

// CheckCompetitionTime 检查比赛时间是否在范围内
func (s *CompetitionServiceImpl) CheckCompetitionTime(ctx context.Context, competitionID uint64) (bool, error) {
	var competition ojmodel.Competition
	metaKey := fmt.Sprintf(competitionMetaKey, competitionID)

	// 1. 优先从 Redis 获取比赛元数据
	metaBytes, err := s.rdb.Get(ctx, metaKey).Bytes()
	if err == nil {
		if err = json.Unmarshal(metaBytes, &competition); err == nil {
			return !time.Now().Before(competition.StartTime) && time.Now().Before(competition.EndTime), nil
		}
		s.log.WarnContext(ctx, "CheckCompetitionTime: failed to unmarshal competition meta from redis", logger.Error(err))
	}

	// 2. 缓存未命中，使用分布式锁进行数据库回源
	lockKey := fmt.Sprintf("lock:competition:%d:meta:load", competitionID)
	ok, err := s.rdb.SetNX(ctx, lockKey, "locked", 10*time.Second).Result()
	if err != nil {
		return false, fmt.Errorf("CheckCompetitionTime: failed to set lock: %w", err)
	}

	if !ok {
		// 未获取到锁，休眠后重试
		time.Sleep(1 * time.Second)
		return s.CheckCompetitionTime(ctx, competitionID)
	}
	defer retry.Do(ctx, func() error {
		return s.rdb.Del(ctx, lockKey).Err()
	}, retry.WithAsync(true), retry.WithCallback(func(err error) {
		s.log.ErrorContext(ctx, "CheckCompetitionTime: failed to delete lock", logger.Error(err))
	}))

	// 3. 获取锁后，再次检查缓存
	metaBytes, err = s.rdb.Get(ctx, metaKey).Bytes()
	if err == nil {
		if err = json.Unmarshal(metaBytes, &competition); err == nil {
			return !time.Now().Before(competition.StartTime) && time.Now().Before(competition.EndTime), nil
		}
	}

	// 4. 从数据库加载比赛元数据
	err = s.db.WithContext(ctx).
		Where("id = ?", competitionID).
		First(&competition).Error
	if err != nil {
		return false, fmt.Errorf("CheckCompetitionTime: failed to select from competition: %w", err)
	}

	// 5. 将元数据写入 Redis 缓存
	metaBytes, err = json.Marshal(competition)
	if err == nil {
		s.rdb.Set(ctx, metaKey, metaBytes, 8*time.Hour)
	} else {
		s.log.ErrorContext(ctx, "CheckCompetitionTime: failed to marshal competition meta", logger.Error(err))
	}

	return !time.Now().Before(competition.StartTime) && time.Now().Before(competition.EndTime), nil
}
