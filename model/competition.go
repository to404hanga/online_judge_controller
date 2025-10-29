package model

import "time"

type CreateCompetitionParam struct {
	CommonParam `json:"-"`

	Name      string    `json:"name" binding:"required"`
	StartTime time.Time `json:"start_time" binding:"required"`
	EndTime   time.Time `json:"end_time" binding:"required"`

	Problems []uint64 `json:"problem_ids"`
}

type UpdateCompetitionParam struct {
	CommonParam `json:"-"`

	ID uint64 `json:"id" binding:"required"`

	Name      *string    `json:"name"`
	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
	Status    *int8      `json:"status" binding:"omitempty,oneof=0 1"`
}

type CompetitionProblemParam struct {
	CommonParam `json:"-"`

	CompetitionID uint64   `json:"competition_id" binding:"required"`
	ProblemIDs    []uint64 `json:"problem_ids" binding:"required"`
}

type StartCompetitionParam struct {
	CommonParam `json:"-"`

	CompetitionID uint64 `json:"competition_id" binding:"required"`
}

type ProblemWithPresignedURL struct {
	ID           uint64 `json:"id" gorm:"id"`
	Title        string `json:"title" gorm:"title"`
	PresignedURL string `json:"presigned_url"`
	TimeLimit    int    `json:"time_limit" gorm:"time_limit"`
	MemoryLimit  int    `json:"memory_limit" gorm:"memory_limit"`
}

type GetCompetitionProblemListWithPresignedURLParam struct {
	CompetitionCommonParam `json:"-"`
}

type GetCompetitionProblemListWithPresignedURLResponse struct {
	Problems []ProblemWithPresignedURL `json:"problems"`
}
