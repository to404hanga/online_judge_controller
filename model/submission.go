package model

import "time"

type GetSubmissionUploadPresignedURLParam struct {
	CompetitionCommonParam `json:"-"`

	ProblemID uint64 `json:"problem_id" binding:"required"`
	Hash      string `json:"hash" binding:"required"`
}

type GetSubmissionUploadPresignedURLResponse struct {
	PresignedURL string `json:"presigned_url"`
}

type SubmitCompetitionProblemParam struct {
	CompetitionCommonParam `json:"-"`

	URL       string `json:"url" binding:"required"`
	Language  int8   `json:"language" binding:"required,oneof=0 1 2 3 4"`
	ProblemID uint64 `json:"problem_id" binding:"required"`
}

// type GetSubmissionListParam struct {
// 	CommonParam `json:"-"`

// 	CompetitionID uint64 `json:"competition_id"`
// 	ProblemID     uint64 `json:"problem_id"`
// }

type Submission struct {
	ID         uint64    `json:"id"`
	Language   int8      `json:"language"`
	Status     int8      `json:"status"`
	Result     int8      `json:"result"`
	TimeUsed   int       `json:"time_used"`
	MemoryUsed int       `json:"memory_used"`
	CreatedAt  time.Time `json:"created_at"`
}

// type GetSubmissionListResponse struct {
// 	List  []Submission `json:"list"`
// 	Total int          `json:"total"`
// }

type GetSubmissionDownloadPresignedURLParam struct {
	CommonParam `json:"-"`

	SubmissionID uint64 `json:"submission_id" binding:"required"`
}

type GetSubmissionDownloadPresignedURLResponse struct {
	PresignedURL string `json:"presigned_url"`
}

type GetLatestSubmissionParam struct {
	CompetitionCommonParam `json:"-"`

	ProblemID uint64 `json:"problem_id" binding:"required"`
}

type GetLatestSubmissionResponse struct {
	Submission
}
