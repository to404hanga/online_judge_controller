package model

import (
	"time"

	ojmodel "github.com/to404hanga/online_judge_common/model"
)

type GetCompetitionRankingListParam struct {
	CompetitionCommonParam `json:"-"`

	Page     int `form:"page" binding:"required,min=1"`
	PageSize int `form:"page_size" binding:"required,min=10,max=100"`
}

type Problem struct {
	ProblemID  uint64        `json:"problem_id"`
	Result     ProblemStatut `json:"result"`      // 题目状态: 0-未尝试, 1-尝试中, 2-通过
	AcceptedAt int64         `json:"accepted_at"` // 通过时间(不含罚时, 单位: 毫秒)
	Retrys     int           `json:"retries"`     // 重试次数
	IsFastest  bool          `json:"is_fastest"`
}

type ProblemStatut int8

const (
	ProblemStatusNotAttempted ProblemStatut = 0 // 未尝试
	ProblemStatusAttempting   ProblemStatut = 1 // 尝试中
	ProblemStatusAccepted     ProblemStatut = 2 // 通过
)

type Ranking struct {
	UserID        uint64    `json:"user_id"`
	Username      string    `json:"username"`        // 学号
	Realname      string    `json:"realname"`        // 真实姓名
	TotalAccepted int       `json:"total_accepted"`  // 通过数
	TotalTimeUsed int64     `json:"total_time_used"` // 总耗时(包括罚时, 单位: 毫秒)
	Problems      []Problem `json:"problems"`        // 题目通过情况
}

type GetCompetitionRankingListResponse struct {
	List     []Ranking `json:"list"`
	Total    int       `json:"total"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
}

type InitRankingParam struct {
	CommonParam `json:"-"`

	CompetitionID uint64 `json:"competition_id" binding:"required"`
}

// 仅内部测试用, 后续 release 版本移除
type UpdateScoreParam struct {
	CommonParam `json:"-"`

	CompetitionID  uint64    `json:"competition_id" binding:"required"`
	UserID         uint64    `json:"user_id" binding:"required"`
	ProblemID      uint64    `json:"problem_id" binding:"required"`
	IsAccepted     *bool     `json:"is_accepted" binding:"required"`
	SubmissionTime time.Time `json:"submission_time" binding:"required"`
	StartTime      time.Time `json:"start_time" binding:"required"`
}

type GetCompetitionListParam struct {
	CommonParam `json:"-"`

	Page     int                        `form:"page" binding:"required,min=1"`
	PageSize int                        `form:"page_size" binding:"required,min=10,max=100"`
	Status   *ojmodel.CompetitionStatus `form:"status" binding:"omitempty,oneof=0 1 2"`
}

type GetCompetitionListResponse struct {
	List     []ojmodel.Competition `json:"list"`
	Total    int                   `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}
