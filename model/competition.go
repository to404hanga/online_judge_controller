package model

import (
	"time"

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
