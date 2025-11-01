package factory

import (
	"github.com/to404hanga/online_judge_controller/service/exporter"
	"github.com/to404hanga/online_judge_controller/service/exporter/csv"
	"github.com/to404hanga/online_judge_controller/service/exporter/xlsx"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
	"gorm.io/gorm"
)

type ExporterType string

const (
	CSVRankingExporter  ExporterType = "csv-ranking"
	XLSXRankingExporter ExporterType = "xlsx-ranking"
	CSVDetailExporter   ExporterType = "csv-detail"
)

var ExporterSuffixMap = map[ExporterType]string{
	CSVRankingExporter:  ".csv",
	XLSXRankingExporter: ".xlsx",
	CSVDetailExporter:   ".csv",
}

type ExporterFactory struct {
	factory map[ExporterType]exporter.Exporter
	db      *gorm.DB
	log     loggerv2.Logger
}

func NewExporterFactory(db *gorm.DB, log loggerv2.Logger) *ExporterFactory {
	return &ExporterFactory{
		factory: make(map[ExporterType]exporter.Exporter), // 延迟创建
		db:      db,
		log:     log,
	}
}

func (f *ExporterFactory) GetExporter(exporterType ExporterType) exporter.Exporter {
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
	case CSVDetailExporter:
		f.factory[CSVDetailExporter] = csv.NewCSVDetailExporter(f.db, f.log)
		return f.factory[CSVDetailExporter]
	}

	return nil
}
