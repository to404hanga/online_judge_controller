package csv

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	ojmodel "github.com/to404hanga/online_judge_common/model"
	"github.com/to404hanga/online_judge_controller/service/exporter"
	"github.com/to404hanga/online_judge_controller/service/exporter/common"
	"github.com/to404hanga/pkg404/gotools/transform"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
	"gorm.io/gorm"
)

type StreamableCSVRankingExporter struct {
	log loggerv2.Logger
	db  *gorm.DB
}

var _ exporter.RankingExporter = (*StreamableCSVRankingExporter)(nil)

func NewStreamableCSVRankingExporter(db *gorm.DB, log loggerv2.Logger) *StreamableCSVRankingExporter {
	return &StreamableCSVRankingExporter{
		db:  db,
		log: log,
	}
}

func (e *StreamableCSVRankingExporter) Export(ctx context.Context, competitionID uint64, writer io.Writer) error {
	ectx, cancel := context.WithCancel(ctx)
	defer cancel()

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

	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	err := e.writeHeader(csvWriter)
	if err != nil {
		return fmt.Errorf("write header failed: %w", err)
	}

	timeBuilder := &strings.Builder{}
	timeBuilder.Grow(12) // %02d:%02d:%02d.%03d 最小长度为 12 字节

	var goroutineErr error
	for {
		select {
		case ranks, ok := <-rankCh:
			if !ok {
				if goroutineErr != nil {
					return fmt.Errorf("sub goroutine fetch ranking failed: %w", goroutineErr)
				}
				return nil
			}
			if err = e.processRanks(timeBuilder, csvWriter, ranks); err != nil {
				return fmt.Errorf("process ranks failed: %w", err)
			}
		case err = <-errCh:
			if err != nil {
				goroutineErr = err
			}
		}
	}
}

// processRanks 处理排名数据，将其转换为 CSV 记录
func (e *StreamableCSVRankingExporter) processRanks(timeBuilder *strings.Builder, csvWriter *csv.Writer, ranks []ojmodel.CompetitionUser) error {
	records := transform.SliceFromSlice(ranks, func(idx int, rank ojmodel.CompetitionUser) []string {
		timeBuilder.Reset()
		fmt.Fprintf(timeBuilder, "%02d:%02d:%02d.%03d",
			rank.TotalTime/3600000,
			(rank.TotalTime%3600000)/60000,
			(rank.TotalTime%60000)/1000,
			rank.TotalTime%1000)
		return []string{
			rank.Username,                // 学号
			rank.Realname,                // 姓名
			strconv.Itoa(rank.PassCount), // 通过题目数
			timeBuilder.String(),         // 总耗时
		}
	})
	return csvWriter.WriteAll(records)
}

// writeHeader 写入 CSV 头部
func (e *StreamableCSVRankingExporter) writeHeader(csvWriter *csv.Writer) error {
	headers := []string{
		"学号",
		"姓名",
		"通过题目数",
		"总耗时",
	}
	return csvWriter.Write(headers)
}
