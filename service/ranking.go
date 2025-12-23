package service

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
	ojmodel "github.com/to404hanga/online_judge_common/model"
	"github.com/to404hanga/online_judge_controller/model"
	"github.com/to404hanga/online_judge_controller/service/exporter/factory"
	"github.com/to404hanga/pkg404/gotools/transform"
	"github.com/to404hanga/pkg404/logger"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
	"gorm.io/gorm"
)

type RankingService interface {
	// GetCompetitionRankingList 获取比赛排行榜
	GetCompetitionRankingList(ctx context.Context, competitionID uint64, page, pageSize int) ([]model.Ranking, int, error)
	// UpdateUserScore 更新用户分数
	UpdateUserScore(ctx context.Context, competitionID, problemID, userID uint64, isAccepted bool, submissionTime time.Time, startTime time.Time) error
	// InitCompetitionRanking 初始化比赛排行榜
	InitCompetitionRanking(ctx context.Context, competitionID uint64) error
	// GetFastestSolverList 获取最快通过每道题的用户
	GetFastestSolverList(ctx context.Context, competitionID uint64, problemIDs []uint64) []model.FastestSolver
	// Export 导出数据
	Export(ctx context.Context, competitionID uint64, exporter factory.ExporterType) (string, error)
}

// RankingServiceImpl 排行榜服务实现, 实时排行榜强依赖 Redis, 暂无 Redis 重建数据功能
type RankingServiceImpl struct {
	db              *gorm.DB
	rdb             redis.Cmdable
	log             loggerv2.Logger
	exporterFactory *factory.ExporterFactory
	exportDir       string
}

var _ RankingService = (*RankingServiceImpl)(nil)

func NewRankingService(db *gorm.DB, rdb redis.Cmdable, log loggerv2.Logger, exportDir string) RankingService {
	return &RankingServiceImpl{
		db:              db,
		rdb:             rdb,
		log:             log,
		exporterFactory: factory.NewExporterFactory(db, log),
		exportDir:       exportDir,
	}
}

const (
	RankingKey              = "ranking:competition:%d"
	UserDetailKey           = "ranking:user:%s:competition:%d"
	ProblemFastestSolverKey = "ranking:problem:%d:competition:%d"
	PenaltyTime             = 20 * 60 * 1000 // 20分钟
	ScoreMultiplier         = 1000000000000
)

// UserRankingData 用户排行榜数据
type UserRankingData struct {
	UserID        uint64                   `json:"user_id" gorm:"-"`
	Username      string                   `json:"username" gorm:"column:username"`
	Realname      string                   `json:"realname" gorm:"column:realname"`
	TotalAccepted int                      `json:"total_accepted" gorm:"-"`
	TotalTimeUsed int64                    `json:"total_time_used" gorm:"-"`
	Problems      map[uint64]model.Problem `json:"problems" gorm:"-"`
}

// GetCompetitionRankingList 获取比赛排行榜
func (s *RankingServiceImpl) GetCompetitionRankingList(ctx context.Context, competitionID uint64, page, pageSize int) ([]model.Ranking, int, error) {
	rankingKey := fmt.Sprintf(RankingKey, competitionID)

	start := int64((page - 1) * pageSize)
	stop := start + int64(pageSize) - 1

	// 获取排行榜(按分数降序)
	userIDs, err := s.rdb.ZRevRange(ctx, rankingKey, start, stop).Result()
	if err != nil {
		return nil, 0, fmt.Errorf("get ranking from redis failed: %w", err)
	}

	// 获取总数
	total, err := s.rdb.ZCard(ctx, rankingKey).Result()
	if err != nil {
		return nil, 0, fmt.Errorf("get total from redis failed: %w", err)
	}

	// 获取用户详细信息
	rankings := make([]model.Ranking, 0, len(userIDs))
	for _, userIDStr := range userIDs {
		userDetailKey := fmt.Sprintf(UserDetailKey, userIDStr, competitionID)
		userDataStr, err := s.rdb.Get(ctx, userDetailKey).Result()
		if err != nil {
			s.log.ErrorContext(ctx, "get user detail from redis failed",
				logger.Error(err),
				logger.String("user_id", userIDStr))
			continue
		}
		var userData UserRankingData
		err = json.Unmarshal([]byte(userDataStr), &userData)
		if err != nil {
			s.log.ErrorContext(ctx, "unmarshal user detail from redis failed",
				logger.Error(err),
				logger.String("user_id", userIDStr))
			continue
		}

		problems := transform.SliceFromMap(userData.Problems, func(k uint64, v model.Problem) model.Problem {
			return v
		})

		rankings = append(rankings, model.Ranking{
			UserID:        userData.UserID,
			Username:      userData.Username,
			Realname:      userData.Realname,
			TotalAccepted: userData.TotalAccepted,
			TotalTimeUsed: userData.TotalTimeUsed,
			Problems:      problems,
		})
	}

	return rankings, int(total), nil
}

// UpdateUserScore 更新用户分数
func (s *RankingServiceImpl) UpdateUserScore(ctx context.Context, competitionID, problemID, userID uint64, isAccepted bool, submissionTime time.Time, startTime time.Time) error {
	userIDStr := strconv.FormatUint(userID, 10)
	userDetailKey := fmt.Sprintf(UserDetailKey, userIDStr, competitionID)
	rankingKey := fmt.Sprintf(RankingKey, competitionID)

	// 获取当前用户数据
	var userData UserRankingData
	userDataStr, err := s.rdb.Get(ctx, userDetailKey).Result()
	if err == redis.Nil {
		// 用户首次提交 or Redis 缓存过期
		var cu ojmodel.CompetitionUser
		err = s.db.WithContext(ctx).Model(&ojmodel.CompetitionUser{}).
			Where("competition_id = ?", competitionID).
			Where("user_id = ?", userID).
			Select("username", "realname", "pass_count", "total_time").
			First(&cu).Error
		if err != nil {
			return fmt.Errorf("get user detail from db failed: %w", err)
		}
		userData = UserRankingData{
			UserID:        userID,
			Username:      cu.Username,
			Realname:      cu.Realname,
			TotalAccepted: int(cu.PassCount),
			TotalTimeUsed: int64(cu.TotalTime),
			Problems:      make(map[uint64]model.Problem),
		}
	} else if err != nil {
		return fmt.Errorf("get user detail from redis failed: %w", err)
	} else {
		if err = json.Unmarshal([]byte(userDataStr), &userData); err != nil {
			return fmt.Errorf("unmarshal user detail from redis failed: %w", err)
		}
	}

	// 获取题目当前状态
	problem, exists := userData.Problems[problemID]
	if !exists {
		// 题目首次提交 or Redis 缓存过期

		// 获取最早的通过记录 ID
		var latestAcceptedID int64
		err = s.db.WithContext(ctx).Model(&ojmodel.Submission{}).
			Where("competition_id = ?", competitionID).
			Where("user_id = ?", userID).
			Where("problem_id = ?", problemID).
			Where("result = ?", model.ProblemStatusAccepted).
			Select("id").
			Limit(1).
			Scan(&latestAcceptedID).Error
		if err != nil {
			return fmt.Errorf("get latest accepted submission id from db failed: %w", err)
		}

		var retryCount int64
		result := model.ProblemStatusNotAttempted
		query := s.db.WithContext(ctx).Model(&ojmodel.Submission{}).
			Where("competition_id = ?", competitionID).
			Where("user_id = ?", userID).
			Where("problem_id = ?", problemID).
			Where("result != ?", model.ProblemStatusAccepted)
		if latestAcceptedID != 0 {
			result = model.ProblemStatusAccepted
			query = query.Where("id < ?", latestAcceptedID)
		}
		err = query.Count(&retryCount).Error
		if err != nil {
			return fmt.Errorf("get retry count from db failed: %w", err)
		}
		problem = model.Problem{
			ProblemID: problemID,
			Result:    result,
			Retrys:    int(retryCount),
		}
	}

	// 如果题目已经通过, 不再更新
	if problem.Result == model.ProblemStatusAccepted {
		return nil
	}

	submissionTimeMs := submissionTime.UnixMilli()
	startTimeMs := startTime.UnixMilli()
	if submissionTimeMs < startTimeMs {
		submissionTimeMs = startTimeMs
	}
	offsetMs := submissionTimeMs - startTimeMs
	if isAccepted {
		if problem.Result != model.ProblemStatusAccepted {
			userData.TotalAccepted++
			penaltyTime := int64(problem.Retrys * PenaltyTime)
			userData.TotalTimeUsed = offsetMs + penaltyTime
			problem.Result = model.ProblemStatusAccepted
			problem.AcceptedAt = offsetMs
			// 最快通过标记更新
			fastKey := fmt.Sprintf(ProblemFastestSolverKey, problemID, competitionID)
			// 读取当前最快
			var prev struct {
				ProblemID  uint64 `json:"problem_id"`
				UserID     uint64 `json:"user_id"`
				AcceptedAt int64  `json:"accepted_at"`
			}
			prevStr, err := s.rdb.Get(ctx, fastKey).Result()
			if err == redis.Nil || err != nil {
				// 无记录或读取失败，直接设置为当前最快
				problem.IsFastest = true
				fastBytes, _ := json.Marshal(struct {
					ProblemID  uint64 `json:"problem_id"`
					UserID     uint64 `json:"user_id"`
					AcceptedAt int64  `json:"accepted_at"`
				}{ProblemID: problemID, UserID: userID, AcceptedAt: userData.TotalTimeUsed})
				_ = s.rdb.Set(ctx, fastKey, fastBytes, 8*time.Hour).Err()
			} else {
				_ = json.Unmarshal([]byte(prevStr), &prev)
				if prev.AcceptedAt == 0 || userData.TotalTimeUsed < prev.AcceptedAt {
					// 更新最快用户
					if prev.UserID != 0 && prev.UserID != userID {
						// 取消之前用户的最快标记
						prevUserKey := fmt.Sprintf(UserDetailKey, strconv.FormatUint(prev.UserID, 10), competitionID)
						prevUserStr, e2 := s.rdb.Get(ctx, prevUserKey).Result()
						if e2 == nil {
							var prevUser UserRankingData
							if json.Unmarshal([]byte(prevUserStr), &prevUser) == nil {
								p := prevUser.Problems[problemID]
								p.IsFastest = false
								prevUser.Problems[problemID] = p
								b, _ := json.Marshal(prevUser)
								_ = s.rdb.Set(ctx, prevUserKey, b, 8*time.Hour).Err()
							}
						}
					}
					problem.IsFastest = true
					fastBytes, _ := json.Marshal(struct {
						ProblemID  uint64 `json:"problem_id"`
						UserID     uint64 `json:"user_id"`
						AcceptedAt int64  `json:"accepted_at"`
					}{ProblemID: problemID, UserID: userID, AcceptedAt: userData.TotalTimeUsed})
					_ = s.rdb.Set(ctx, fastKey, fastBytes, 8*time.Hour).Err()
				}
			}
		}
	} else {
		problem.Retrys++
		problem.Result = model.ProblemStatusAttempting
	}

	userData.Problems[problemID] = problem

	// 保存用户数据
	userDataBytes, err := json.Marshal(userData)
	if err != nil {
		return fmt.Errorf("marshal user detail to redis failed: %w", err)
	}

	err = s.rdb.Set(ctx, userDetailKey, userDataBytes, 8*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("set user detail to redis failed: %w", err)
	}

	score := s.calculateScore(userData.TotalAccepted, userData.TotalTimeUsed)

	// 更新排行榜
	err = s.rdb.ZAdd(ctx, rankingKey, redis.Z{
		Score:  score,
		Member: userIDStr,
	}).Err()
	if err != nil {
		return fmt.Errorf("zadd ranking to redis failed: %w", err)
	}

	return nil
}

// InitCompetitionRanking 初始化比赛排行榜 (从 MySQL 重建 Redis 数据)
func (s *RankingServiceImpl) InitCompetitionRanking(ctx context.Context, competitionID uint64) error {
	rankingKey := fmt.Sprintf(RankingKey, competitionID)
	ctx = loggerv2.ContextWithFields(ctx, logger.Uint64("competition_id", competitionID))

	// 1. 清理现有 Redis 数据
	// 获取所有榜单用户，删除其详情 Key
	userIDs, err := s.rdb.ZRange(ctx, rankingKey, 0, -1).Result()
	if err != nil {
		s.log.WarnContext(ctx, "InitCompetitionRanking: failed to get existing members", logger.Error(err))
	}

	pipeline := s.rdb.Pipeline()
	for _, uid := range userIDs {
		pipeline.Del(ctx, fmt.Sprintf(UserDetailKey, uid, competitionID))
	}
	// 删除排行榜 ZSet
	pipeline.Del(ctx, rankingKey)

	// 删除题目最快解题者记录
	var problemIDList []uint64
	if err = s.db.WithContext(ctx).Model(&ojmodel.CompetitionProblem{}).
		Where("competition_id = ?", competitionID).
		Pluck("problem_id", &problemIDList).Error; err != nil {
		s.log.WarnContext(ctx, "InitCompetitionRanking: failed to pluck problem_id",
			logger.Error(err))
	}
	for _, pid := range problemIDList {
		pipeline.Del(ctx, fmt.Sprintf(ProblemFastestSolverKey, pid, competitionID))
	}

	if _, err = pipeline.Exec(ctx); err != nil {
		return fmt.Errorf("clean redis failed: %w", err)
	}

	// 2. 从 MySQL 加载该比赛所有有效提交 (按 ID 升序/时间升序)
	var submissions []ojmodel.Submission
	err = s.db.WithContext(ctx).
		Model(&ojmodel.Submission{}).
		Where("competition_id = ?", competitionID).
		Where("result IS NOT NULL"). // 过滤未判题的
		Where("result != ?", -1).    // 假设 -1 是未判题
		Order("id ASC").             // 保证回放顺序
		Find(&submissions).Error
	if err != nil {
		return fmt.Errorf("load submissions from db failed: %w", err)
	}

	// 先获取比赛开始时间
	var comp ojmodel.Competition
	if err := s.db.WithContext(ctx).Model(&ojmodel.Competition{}).
		Where("id = ?", competitionID).
		Select("start_time").
		First(&comp).Error; err != nil {
		return fmt.Errorf("load competition start_time failed: %w", err)
	}

	// 3. 重放提交记录重建排行榜
	for _, sub := range submissions {
		if sub.Result == nil {
			continue
		}
		isAccepted := *sub.Result == ojmodel.SubmissionResultAccepted
		err := s.UpdateUserScore(ctx, competitionID, sub.ProblemID, sub.UserID, isAccepted, sub.CreatedAt, comp.StartTime)
		if err != nil {
			s.log.ErrorContext(ctx, "InitCompetitionRanking: replay submission failed",
				logger.Error(err),
				logger.Uint64("submission_id", sub.ID))
		}
	}

	return nil
}

// 场景1：解题数不同
// 选手A: 3题, 1小时 = 3 × 10^12 - 3600000 = 2999999996400000
// 选手B: 2题, 10分钟 = 2 × 10^12 - 600000 = 1999999999400000
// 结果：A > B ✓

// 场景2：解题数相同，时间不同
// 选手A: 2题, 30分钟 = 2 × 10^12 - 1800000 = 1999999998200000
// 选手B: 2题, 45分钟 = 2 × 10^12 - 2700000 = 1999999997300000
// 结果：A > B ✓

// 场景3：极端情况
// 选手A: 10题, 5小时 = 10 × 10^12 - 18000000 = 9999999982000000
// 选手B: 9题, 1分钟 = 9 × 10^12 - 60000 = 8999999999940000
// 结果：A > B ✓
func (s *RankingServiceImpl) calculateScore(totalAccepted int, totalTimeUsed int64) float64 {
	return float64(int64(totalAccepted)*ScoreMultiplier - totalTimeUsed)
}

// GetFastestSolverList 获取最快通过每道题的用户
func (s *RankingServiceImpl) GetFastestSolverList(ctx context.Context, competitionID uint64, problemIDs []uint64) []model.FastestSolver {
	res := make([]model.FastestSolver, 0, len(problemIDs))
	for _, problemID := range problemIDs {
		problemFastestSolverKey := fmt.Sprintf(ProblemFastestSolverKey, problemID, competitionID)

		solverDataStr, err := s.rdb.Get(ctx, problemFastestSolverKey).Result()
		if err != nil {
			s.log.ErrorContext(ctx, "get problem fastest solver from redis failed", logger.Error(err))
			continue
		}
		if len(solverDataStr) == 0 {
			// 没人通过这道题
			continue
		}

		var solverData model.FastestSolver
		if err = json.Unmarshal([]byte(solverDataStr), &solverData); err != nil {
			s.log.ErrorContext(ctx, "unmarshal problem fastest solver from redis failed", logger.Error(err))
			continue
		}
		res = append(res, solverData)
	}

	return res
}

// Export 导出数据
func (s *RankingServiceImpl) Export(ctx context.Context, competitionID uint64, exporter factory.ExporterType) (string, error) {
	exp := s.exporterFactory.GetExporter(exporter)
	if exp == nil {
		return "", fmt.Errorf("get exporter failed: exporter not found")
	}
	filepath := fmt.Sprintf("%s/%d%s", s.exportDir, competitionID, factory.ExporterSuffixMap[exporter])
	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("create file failed: %w", err)
	}
	defer file.Close()
	return filepath, exp.Export(ctx, competitionID, file)
}
