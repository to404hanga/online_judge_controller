package factory

import (
	"github.com/to404hanga/online_judge_controller/service/exporter"
	"github.com/to404hanga/online_judge_controller/service/exporter/csv"
	"github.com/to404hanga/online_judge_controller/service/exporter/xlsx"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
	"gorm.io/gorm"
)

type RankingExporterType string

const (
	CSVRankingExporter  RankingExporterType = "csv"
	XLSXRankingExporter RankingExporterType = "xlsx"
)

type RankingExporterFactory struct {
	factory map[RankingExporterType]exporter.RankingExporter
	db      *gorm.DB
	log     loggerv2.Logger
}

func NewRankingExporterFactory(db *gorm.DB, log loggerv2.Logger) *RankingExporterFactory {
	return &RankingExporterFactory{
		factory: map[RankingExporterType]exporter.RankingExporter{
			CSVRankingExporter:  csv.NewStreamableCSVRankingExporter(db, log),
			XLSXRankingExporter: xlsx.NewStreamableXLSXRankingExporter(db, log),
		},
		db:  db,
		log: log,
	}
}

func (f *RankingExporterFactory) GetRankingExporter(exporterType RankingExporterType) exporter.RankingExporter {
	if exp, exists := f.factory[exporterType]; exists {
		return exp
	}

	switch exporterType {
	case CSVRankingExporter:
		f.factory[CSVRankingExporter] = csv.NewStreamableCSVRankingExporter(f.db, f.log)
		return f.factory[CSVRankingExporter]
	case XLSXRankingExporter:
		f.factory[XLSXRankingExporter] = xlsx.NewStreamableXLSXRankingExporter(f.db, f.log)
		return f.factory[XLSXRankingExporter]
	}

	return nil
}
