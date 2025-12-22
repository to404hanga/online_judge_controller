package model

import "time"

type SubmitCompetitionProblemParam struct {
	CompetitionCommonParam `json:"-"`

	Code      string `json:"code" binding:"required"`
	Language  int8   `json:"language" binding:"required,oneof=0 1 2 3 4"`
	ProblemID uint64 `json:"problem_id" binding:"required"`
}

type Submission struct {
	ID         uint64    `json:"id"`
	Code       string    `json:"code"`
	Stderr     string    `json:"stderr"`
	Language   int8      `json:"language"`
	Status     int8      `json:"status"`
	Result     int8      `json:"result"`
	TimeUsed   int       `json:"time_used"`
	MemoryUsed int       `json:"memory_used"`
	CreatedAt  time.Time `json:"created_at"`
}

type GetLatestSubmissionParam struct {
	CompetitionCommonParam `json:"-"`

	ProblemID uint64 `form:"problem_id" binding:"required"`
}

type GetLatestSubmissionResponse struct {
	Submission
}
