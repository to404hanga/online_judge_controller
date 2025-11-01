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
	UpdateUserScore(ctx context.Context, competitionID, problemID, userID uint64, isAccepted bool, submissionTime time.Time) error
	// InitCompetitionRanking 初始化比赛排行榜
	InitCompetitionRanking(ctx context.Context, competitionID uint64) error
	// GetFastestSolverList 获取最快通过每道题的用户
	GetFastestSolverList(ctx context.Context, competitionID uint64, problemIDs []uint64) []model.FastestSolver
	// Export 导出比赛排行榜
	Export(ctx context.Context, competitionID uint64, exporter factory.RankingExporterType) error
}

// RankingServiceImpl 排行榜服务实现, 实时排行榜强依赖 Redis, 暂无 Redis 重建数据功能
type RankingServiceImpl struct {
	db              *gorm.DB
	rdb             redis.Cmdable
	log             loggerv2.Logger
	exporterFactory *factory.RankingExporterFactory
}

var _ RankingService = (*RankingServiceImpl)(nil)

func NewRankingService(db *gorm.DB, rdb redis.Cmdable, log loggerv2.Logger) RankingService {
	return &RankingServiceImpl{
		db:              db,
		rdb:             rdb,
		log:             log,
		exporterFactory: factory.NewRankingExporterFactory(db, log),
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
	UserID        uint64                   `json:"user_id"`
	Username      string                   `json:"username" gorm:"column:username"`
	Realname      string                   `json:"realname" gorm:"column:realname"`
	TotalAccepted int                      `json:"total_accepted"`
	TotalTimeUsed int64                    `json:"total_time_used"`
	Problems      map[uint64]model.Problem `json:"problems"`
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
			Realname:      userData.Realname,
			TotalAccepted: userData.TotalAccepted,
			TotalTimeUsed: userData.TotalTimeUsed,
			Problems:      problems,
		})
	}

	return rankings, int(total), nil
}

// UpdateUserScore 更新用户分数
func (s *RankingServiceImpl) UpdateUserScore(ctx context.Context, competitionID, problemID, userID uint64, isAccepted bool, submissionTime time.Time) error {
	userIDStr := strconv.FormatUint(userID, 10)
	userDetailKey := fmt.Sprintf(UserDetailKey, userIDStr, competitionID)
	rankingKey := fmt.Sprintf(RankingKey, competitionID)

	// 获取当前用户数据
	var userData UserRankingData
	userDataStr, err := s.rdb.Get(ctx, userDetailKey).Result()
	if err == redis.Nil {
		// 用户首次提交, 初始化数据
		userData = UserRankingData{
			UserID:        userID,
			TotalAccepted: 0,
			TotalTimeUsed: 0,
			Problems:      make(map[uint64]model.Problem),
		}
		err = s.db.WithContext(ctx).Model(&ojmodel.User{}).
			Where("id = ?", userID).
			Select("username", "realname").
			First(&userData).Error
		if err != nil {
			return fmt.Errorf("get user detail from db failed: %w", err)
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
		// 题目首次提交, 初始化数据
		problem = model.Problem{
			ProblemID: problemID,
			Result:    model.ProblemStatusNotAttempted,
			Retrys:    0,
		}
	}

	// 如果题目已经通过, 不再更新
	if problem.Result == model.ProblemStatusAccepted {
		return nil
	}

	if isAccepted {
		if problem.Result != model.ProblemStatusAccepted {
			userData.TotalAccepted++
			submissionTimeMs := submissionTime.UnixMilli()
			penaltyTime := int64(problem.Retrys * PenaltyTime)
			userData.TotalTimeUsed += submissionTimeMs + penaltyTime
			problem.Result = model.ProblemStatusAccepted
			problem.AcceptedAt = submissionTimeMs
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

// InitCompetitionRanking 初始化比赛排行榜
func (s *RankingServiceImpl) InitCompetitionRanking(ctx context.Context, competitionID uint64) error {
	rankingKey := fmt.Sprintf(RankingKey, competitionID)

	// 清空现有排行榜
	err := s.rdb.ZRemRangeByRank(ctx, rankingKey, 0, -1).Err()
	if err != nil {
		return fmt.Errorf("zremrangebyrank ranking to redis failed: %w", err)
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

// Export 导出排行榜
func (s *RankingServiceImpl) Export(ctx context.Context, competitionID uint64, exporter factory.RankingExporterType) error {
	exp := s.exporterFactory.GetRankingExporter(exporter)
	if exp == nil {
		return fmt.Errorf("get ranking exporter failed: exporter not found")
	}
	file, err := os.Create(fmt.Sprintf("%d.%s", competitionID, exporter))
	if err != nil {
		return fmt.Errorf("create file failed: %w", err)
	}
	defer file.Close()
	return exp.Export(ctx, competitionID, file)
}
