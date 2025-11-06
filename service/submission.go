package service

import (
	"context"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/redis/go-redis/v9"
	ojmodel "github.com/to404hanga/online_judge_common/model"
	"github.com/to404hanga/online_judge_controller/event"
	"github.com/to404hanga/online_judge_controller/model"
	"github.com/to404hanga/online_judge_controller/pkg/pointer"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
	"gorm.io/gorm"
)

type SubmissionService interface {
	// SubmitCompetitionProblem 提交比赛题目
	SubmitCompetitionProblem(ctx context.Context, param *model.SubmitCompetitionProblemParam) error
	// GetLatestSubmission 获取最新提交记录
	GetLatestSubmission(ctx context.Context, competitionID, problemID, userID uint64) (*ojmodel.Submission, error)
	// GetSubmissionByID 获取提交记录
	GetSubmissionByID(ctx context.Context, submissionID uint64) (*ojmodel.Submission, error)
	// CleanUserFailedSubmission 清理给定截止时间之前所有用户的失败提交记录(仅清理提交代码)
	CleanUserFailedSubmission(ctx context.Context, timeDeadline time.Time) error
}

type SubmissionServiceImpl struct {
	db    *gorm.DB
	rdb   redis.Cmdable
	kafka event.Producer
	log   loggerv2.Logger
}

var _ SubmissionService = (*SubmissionServiceImpl)(nil)

func NewSubmissionService(db *gorm.DB, rdb redis.Cmdable, kafka event.Producer, log loggerv2.Logger) SubmissionService {
	return &SubmissionServiceImpl{
		db:    db,
		rdb:   rdb,
		kafka: kafka,
		log:   log,
	}
}

// SubmitCompetitionProblem 提交比赛题目
func (s *SubmissionServiceImpl) SubmitCompetitionProblem(ctx context.Context, param *model.SubmitCompetitionProblemParam) error {
	submission := ojmodel.Submission{
		CompetitionID: param.CompetitionID,
		ProblemID:     param.ProblemID,
		UserID:        param.Operator,
		Code:          param.Code,
		Language:      pointer.ToPtr(ojmodel.SubmissionLanguage(param.Language)),
	}

	err := s.db.WithContext(ctx).Create(&submission).Error
	if err != nil {
		return fmt.Errorf("SubmitCompetitionProblem failed at create submission: %w", err)
	}

	// 通过 kafka 发布提交任务
	msg := event.SubmissionMessage{SubmissionID: int64(submission.ID)}
	val, err := msg.Marshal()
	if err != nil {
		return fmt.Errorf("SubmitCompetitionProblem failed at marshal message: %w", err)
	}
	_, _, err = s.kafka.Produce(ctx, &sarama.ProducerMessage{
		Topic: event.SubmissionTopic,
		Value: sarama.ByteEncoder(val),
	})
	if err != nil {
		return fmt.Errorf("SubmitCompetitionProblem failed at produce message: %w", err)
	}

	return nil
}

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

func (s *SubmissionServiceImpl) CleanUserFailedSubmission(ctx context.Context, timeDeadline time.Time) error {
	err := s.db.WithContext(ctx).Model(&ojmodel.Submission{}).
		Where("result != ?", ojmodel.SubmissionResultAccepted).
		Where("created_at < ?", timeDeadline).
		UpdateColumn("code", "**代码已被清理**").Error
	if err != nil {
		return fmt.Errorf("CleanUserFailedSubmission failed at update submission: %w", err)
	}
	return nil
}
