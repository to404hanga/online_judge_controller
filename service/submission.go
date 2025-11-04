package service

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	ojmodel "github.com/to404hanga/online_judge_common/model"
	"github.com/to404hanga/online_judge_controller/model"
	"github.com/to404hanga/online_judge_controller/pkg/pointer"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
	"gorm.io/gorm"
)

type SubmissionService interface {
	// SubmitCompetitionProblem 提交比赛题目
	SubmitCompetitionProblem(ctx context.Context, param *model.SubmitCompetitionProblemParam) error
	// // GetSubmissionList 获取比赛题目提交列表
	// GetSubmissionList(ctx context.Context, param *model.GetSubmissionListParam) ([]model.Submission, error)
	// // PublishJudgeTask 发布判题任务到 Redis Stream
	// PublishJudgeTask(ctx context.Context, task *JudgeTask) error
	// GetLatestSubmission 获取最新提交记录
	GetLatestSubmission(ctx context.Context, competitionID, problemID, userID uint64) (*ojmodel.Submission, error)
	// GetSubmissionByID 获取提交记录
	GetSubmissionByID(ctx context.Context, submissionID uint64) (*ojmodel.Submission, error)
}

type SubmissionServiceImpl struct {
	db  *gorm.DB
	rdb redis.Cmdable
	log loggerv2.Logger
}

var _ SubmissionService = (*SubmissionServiceImpl)(nil)

func NewSubmissionService(db *gorm.DB, rdb redis.Cmdable, log loggerv2.Logger) SubmissionService {
	return &SubmissionServiceImpl{
		db:  db,
		rdb: rdb,
		log: log,
	}
}

// SubmitCompetitionProblem 提交比赛题目
func (s *SubmissionServiceImpl) SubmitCompetitionProblem(ctx context.Context, param *model.SubmitCompetitionProblemParam) error {
	submission := ojmodel.Submission{
		CompetitionID: param.CompetitionID,
		ProblemID:     param.ProblemID,
		UserID:        param.Operator,
		CodeURL:       param.URL,
		Language:      pointer.ToPtr(ojmodel.SubmissionLanguage(param.Language)),
	}

	err := s.db.WithContext(ctx).Create(&submission).Error
	if err != nil {
		return fmt.Errorf("SubmitCompetitionProblem failed at create submission: %w", err)
	}

	// // 后台异步发布判题任务, 失败时重试
	// // 重试到失败也不关心, 判题集群 master 会定时扫描仍未开始判题的任务
	// redisCtx := loggerv2.ContextWithFields(ctx, logger.Uint64("submission_id", submission.ID))
	// go func() {
	// 	err = retry.Do(redisCtx, func() error {
	// 		return s.PublishJudgeTask(redisCtx, &JudgeTask{
	// 			SubmiisionID: submission.ID,
	// 			CodeURL:      submission.CodeURL,
	// 			Language:     submission.Language.Int8(),
	// 			CreatedAt:    submission.CreatedAt.UnixNano(),
	// 		})
	// 	})
	// 	if err != nil {
	// 		s.log.WarnContext(redisCtx, "PublishJudgeTask failed", logger.Error(err))
	// 	}
	// }()

	return nil
}

// // GetSubmissionList 获取比赛题目提交列表
// func (s *SubmissionServiceImpl) GetSubmissionList(ctx context.Context, param *model.GetSubmissionListParam) ([]model.Submission, error) {
// 	var models []ojmodel.Submission
// 	err := s.db.WithContext(ctx).Model(&ojmodel.Submission{}).
// 		Where("competition_id = ?", param.CompetitionID).
// 		Where("user_id = ?", param.Operator).
// 		Where("problem_id = ?", param.ProblemID).
// 		Find(&models).Error
// 	if err != nil {
// 		return nil, fmt.Errorf("GetSubmissionList failed at find submission: %w", err)
// 	}

// 	domains := transform.SliceFromSlice(models, func(idx int, m ojmodel.Submission) model.Submission {
// 		return model.Submission{
// 			ID:         m.ID,
// 			Language:   m.Language.Int8(),
// 			Status:     m.Status.Int8(),
// 			Result:     m.Result.Int8(),
// 			TimeUsed:   *m.TimeUsed,
// 			MemoryUsed: *m.MemoryUsed,
// 			CreatedAt:  m.CreatedAt,
// 		}
// 	})

// 	return domains, nil
// }

type JudgeTask struct {
	SubmiisionID uint64 `json:"submission_id"`
	CodeURL      string `json:"code_url"`
	Language     int8   `json:"language"`
	CreatedAt    int64  `json:"created_at"` // 纳秒
}

const (
	JudgeTaskStream = "judge:tasks"    // Redis Stream 名称
	JudgeTaskGroup  = "judge:consumer" // 消费者组名称
)

// // PublishJudgeTask 发布判题任务到 Redis Stream
// func (s *SubmissionServiceImpl) PublishJudgeTask(ctx context.Context, task *JudgeTask) error {
// 	taskData, err := json.Marshal(task)
// 	if err != nil {
// 		return fmt.Errorf("PublishJudgeTask failed at marshal task: %w", err)
// 	}

// 	args := &redis.XAddArgs{
// 		Stream: JudgeTaskStream,
// 		Values: map[string]any{
// 			"task": string(taskData),
// 		},
// 	}

// 	messageID, err := s.rdb.XAdd(ctx, args).Result()
// 	if err != nil {
// 		return fmt.Errorf("PublishJudgeTask failed at xadd task: %w", err)
// 	}

// 	s.log.InfoContext(ctx, "PublishJudgeTask success",
// 		logger.String("message_id", messageID),
// 		logger.String("stream", JudgeTaskStream),
// 	)
// 	return nil
// }

func (s *SubmissionServiceImpl) GetLatestSubmission(ctx context.Context, competitionID, problemID, userID uint64) (*ojmodel.Submission, error) {
	var submission ojmodel.Submission
	err := s.db.WithContext(ctx).Model(&ojmodel.Submission{}).
		Where("competition_id = ?", competitionID).
		Where("user_id = ?", userID).
		Where("problem_id = ?", problemID).
		Order("created_at desc").
		First(&submission).Error
	if err != nil {
		return nil, fmt.Errorf("GetLatestSubmission failed at find submission: %w", err)
	}
	return &submission, nil
}

// GetSubmissionByID 获取提交记录
func (s *SubmissionServiceImpl) GetSubmissionByID(ctx context.Context, submissionID uint64) (*ojmodel.Submission, error) {
	var submission ojmodel.Submission
	err := s.db.WithContext(ctx).Model(&ojmodel.Submission{}).
		Where("id = ?", submissionID).
		First(&submission).Error
	if err != nil {
		return nil, fmt.Errorf("GetSubmissionByID failed at find submission: %w", err)
	}
	return &submission, nil
}
