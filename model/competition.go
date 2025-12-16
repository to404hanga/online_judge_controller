package model

import (
	"time"

	ojmodel "github.com/to404hanga/online_judge_common/model"
	"github.com/to404hanga/online_judge_controller/service/exporter/factory"
)

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
	Status    *int8      `json:"status" binding:"omitempty,oneof=0 1 2"`
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

type GetCompetitionFastestSolverListParam struct {
	CompetitionCommonParam `json:"-"`

	ProblemIDs []uint64 `json:"problem_ids"`
}

type FastestSolver struct {
	ProblemID uint64 `json:"problem_id"`
	UserID    uint64 `json:"user_id"`
}

type GetCompetitionFastestSolverListResponse struct {
	List  []FastestSolver `json:"list"`
	Total int             `json:"total"`
}

type ModelExportType int8

const (
	ModelExportTypeCSVRanking ModelExportType = iota + 1
	ModelExportTypeXLSXRanking
	ModelExportTypeCSVDetail
)

func (t ModelExportType) ToFactoryType() factory.ExporterType {
	switch t {
	case ModelExportTypeCSVRanking:
		return factory.CSVRankingExporter
	case ModelExportTypeXLSXRanking:
		return factory.XLSXRankingExporter
	case ModelExportTypeCSVDetail:
		return factory.CSVDetailExporter
	default:
		return factory.UnknownExporter
	}
}

type ExportCompetitionDataParam struct {
	CommonParam `json:"-"`

	CompetitionID uint64          `json:"competition_id" binding:"required"`
	ExportType    ModelExportType `json:"export_type" binding:"required,oneof=1 2 3"`
}

type GetCompetitionListParam struct {
	CommonParam `json:"-"`

	Desc    bool                       `form:"desc"`
	OrderBy string                     `form:"order_by" binding:"omitempty,oneof=id start_time end_time"`
	Name    string                     `form:"name"`
	Status  *ojmodel.CompetitionStatus `form:"status" binding:"omitempty,oneof=0 1 2"`
	Phase   *CompetitionPhase          `form:"phase" binding:"omitempty,oneof=0 1 2"` // 比赛进行阶段，0：未开始，1：进行中，2：已结束

	Page     int `form:"page" binding:"required,min=1"`
	PageSize int `form:"page_size" binding:"required,min=10,max=100"`
}

type GetCompetitionListResponse struct {
	List     []ojmodel.Competition `json:"list"`
	Total    int                   `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"page_size"`
}

type UserGetCompetitionListParam struct {
	CommonParam `json:"-"`

	Desc    bool              `form:"desc"`
	OrderBy string            `form:"order_by" binding:"omitempty,oneof=id start_time end_time"`
	Name    string            `form:"name"`
	Phase   *CompetitionPhase `form:"phase" binding:"omitempty,oneof=0 1 2"` // 比赛进行阶段，0：未开始，1：进行中，2：已结束

	Page     int `form:"page" binding:"required,min=1"`
	PageSize int `form:"page_size" binding:"required,min=10,max=100"`
}

type CompetitionPhase int8

const (
	CompetitionPhaseNotStarted CompetitionPhase = iota // 未开始
	CompetitionPhaseOngoing                            // 进行中
	CompetitionPhaseEnded                              // 已结束
)

func (p *CompetitionPhase) Int8() int8 {
	switch *p {
	case CompetitionPhaseNotStarted:
		return 0
	case CompetitionPhaseOngoing:
		return 1
	case CompetitionPhaseEnded:
		return 2
	default:
		return 0
	}
}

type GetCompetitionProblemListParam struct {
	CommonParam `json:"-"`

	CompetitionID uint64 `form:"competition_id" binding:"required"`
}
