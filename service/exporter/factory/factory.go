package factory

import (
	"sync"

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
	mux     sync.RWMutex
}

func NewExporterFactory(db *gorm.DB, log loggerv2.Logger) *ExporterFactory {
	return &ExporterFactory{
		factory: make(map[ExporterType]exporter.Exporter), // 延迟创建
		db:      db,
		log:     log,
	}
}

func (f *ExporterFactory) GetExporter(exporterType ExporterType) exporter.Exporter {
	f.mux.RLock()
	if exp, exists := f.factory[exporterType]; exists {
		f.mux.RUnlock()
		return exp
	}
	f.mux.RUnlock()

	f.mux.Lock()
	defer f.mux.Unlock()

	// 双重检查，避免重复创建
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
