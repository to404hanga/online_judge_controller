package csv

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	ojmodel "github.com/to404hanga/online_judge_common/model"
	"github.com/to404hanga/online_judge_controller/service/exporter"
	"github.com/to404hanga/online_judge_controller/service/exporter/common"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
	"gorm.io/gorm"
)

type CSVDetailExporter struct {
	log loggerv2.Logger
	db  *gorm.DB
}

var _ exporter.Exporter = (*CSVDetailExporter)(nil)

func NewCSVDetailExporter(db *gorm.DB, log loggerv2.Logger) exporter.Exporter {
	return &CSVDetailExporter{
		db:  db,
		log: log,
	}
}

func (e *CSVDetailExporter) Export(ctx context.Context, competitionID uint64, writer io.Writer) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	var problemIDList []uint64
	err := e.db.WithContext(ctx).Model(&ojmodel.CompetitionProblem{}).
		Where("competition_id = ?", competitionID).
		Where("status = ?", ojmodel.CompetitionProblemStatusEnabled).
		Order("problem_id ASC").
		Pluck("problem_id", &problemIDList).Error
	if err != nil {
		return fmt.Errorf("get problem id list failed: %w", err)
	}
	headers := make([]string, 0, (len(problemIDList)+1)*2)
	headers = append(headers, "学号", "姓名")
	for _, problemID := range problemIDList {
		headers = append(headers,
			fmt.Sprintf("%d题-通过时间", problemID),
			fmt.Sprintf("%d题-尝试次数", problemID))
	}

	err = csvWriter.Write(headers)
	if err != nil {
		return fmt.Errorf("write header failed: %w", err)
	}

	details, err := common.FetchDetail(e.db, ctx, competitionID)
	if err != nil {
		return fmt.Errorf("csv exporter fetch detail failed: %w", err)
	}
	record := make([]string, 0, len(headers))
	for _, detail := range details {
		record = record[:0] // 清空记录
		record = append(record, detail.Username, detail.Realname)
		for _, problemID := range problemIDList {
			if detail.ProblemID == problemID {
				record = append(record, detail.GetAcceptTime(), strconv.Itoa(detail.AttemptsBeforeAccepted))
			} else {
				record = append(record, "", "")
			}
		}
		err = csvWriter.Write(record)
		if err != nil {
			return fmt.Errorf("write record failed: %w", err)
		}
	}
	return nil
}
