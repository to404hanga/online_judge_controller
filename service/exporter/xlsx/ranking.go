package xlsx

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	ojmodel "github.com/to404hanga/online_judge_common/model"
	"github.com/to404hanga/online_judge_controller/service/exporter"
	"github.com/to404hanga/online_judge_controller/service/exporter/common"
	"github.com/to404hanga/pkg404/logger"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type StreamableXLSXRankingExporter struct {
	log loggerv2.Logger
	db  *gorm.DB
}

var _ exporter.Exporter = (*StreamableXLSXRankingExporter)(nil)

func NewStreamableXLSXRankingExporter(db *gorm.DB, log loggerv2.Logger) exporter.Exporter {
	return &StreamableXLSXRankingExporter{
		db:  db,
		log: log,
	}
}

func (e *StreamableXLSXRankingExporter) Export(ctx context.Context, competitionID uint64, writer io.Writer) error {
	ectx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 创建新的Excel文件
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			e.log.ErrorContext(ctx, "close excel file failed", logger.Error(err))
		}
	}()

	sheetName := "排名"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("create sheet failed: %w", err)
	}
	f.SetActiveSheet(index)

	if err = e.writeHeader(f, sheetName); err != nil {
		return fmt.Errorf("write header failed: %w", err)
	}

	batchSize := 1000
	page := 1
	rankCh := make(chan []ojmodel.CompetitionUser, 3)
	errCh := make(chan error, 1)

	go func() {
		defer close(rankCh)
		defer close(errCh)
		for {
			select {
			case <-ectx.Done():
				errCh <- ectx.Err()
				return
			default:
				ranks, errGoroutine := common.FetchRanking(e.db, ectx, competitionID, page, batchSize)
				if errGoroutine != nil {
					errCh <- errGoroutine
					return
				}
				if len(ranks) == 0 {
					return
				}
				rankCh <- ranks
				page++
			}
		}
	}()

	timeBuilder := &strings.Builder{}
	timeBuilder.Grow(12) // %02d:%02d:%02d.%03d 最小长度为 12 字节

	currentRow := 2 // 从第二行开始写入数据（第一行是表头）
	var goroutineErr error

	for {
		select {
		case ranks, ok := <-rankCh:
			if !ok {
				if goroutineErr != nil {
					return fmt.Errorf("sub goroutine fetch ranking failed: %w", goroutineErr)
				}
				// 所有数据处理完成，写入到writer
				if err = f.Write(writer); err != nil {
					return fmt.Errorf("write excel file failed: %w", err)
				}
				return nil
			}
			if err = e.processRanks(timeBuilder, f, sheetName, ranks, &currentRow); err != nil {

				return fmt.Errorf("process ranks failed: %w", err)
			}
		case err = <-errCh:
			if err != nil {
				goroutineErr = err
			}
		}
	}
}

// processRanks 处理排名数据，将其写 Excel 文件
func (e *StreamableXLSXRankingExporter) processRanks(timeBuilder *strings.Builder, f *excelize.File, sheetName string, ranks []ojmodel.CompetitionUser, currentRow *int) error {
	for _, rank := range ranks {
		timeBuilder.Reset()
		fmt.Fprintf(timeBuilder, "%02d:%02d:%02d.%03d",
			rank.TotalTime/3600000,
			(rank.TotalTime%3600000)/60000,
			(rank.TotalTime%60000)/1000,
			rank.TotalTime%1000)

		// 写入每一行数据
		rowData := []interface{}{
			rank.Username,                // 学号
			rank.Realname,                // 姓名
			strconv.Itoa(rank.PassCount), // 通过题目数
			timeBuilder.String(),         // 总耗时
		}

		for col, value := range rowData {
			cell, err := excelize.CoordinatesToCellName(col+1, *currentRow)
			if err != nil {
				return fmt.Errorf("get cell name failed: %w", err)
			}
			if err := f.SetCellValue(sheetName, cell, value); err != nil {
				return fmt.Errorf("set cell value failed: %w", err)
			}
		}
		*currentRow++
	}
	return nil
}

// writeHeader 写入Excel表头
func (e *StreamableXLSXRankingExporter) writeHeader(f *excelize.File, sheetName string) error {
	headers := []string{
		"学号",
		"姓名",
		"通过题目数",
		"总耗时",
	}

	// 设置表头样式
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E0E0E0"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	if err != nil {
		return fmt.Errorf("create header style failed: %w", err)
	}

	for col, header := range headers {
		cell, err := excelize.CoordinatesToCellName(col+1, 1)
		if err != nil {
			return fmt.Errorf("get cell name failed: %w", err)
		}
		if err := f.SetCellValue(sheetName, cell, header); err != nil {
			return fmt.Errorf("set header value failed: %w", err)
		}
		if err := f.SetCellStyle(sheetName, cell, cell, headerStyle); err != nil {
			return fmt.Errorf("set header style failed: %w", err)
		}
	}

	// 设置列宽
	columnWidths := map[string]float64{
		"A": 20, // 学号
		"B": 15, // 姓名
		"C": 15, // 通过题目数
		"D": 20, // 总耗时
	}

	for col, width := range columnWidths {
		if err := f.SetColWidth(sheetName, col, col, width); err != nil {
			return fmt.Errorf("set column width failed: %w", err)
		}
	}

	return nil
}
