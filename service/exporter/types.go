package exporter

import (
	"context"
	"io"
)

type RankingExporter interface {
	Export(ctx context.Context, competitionID uint64, writer io.Writer) error
}
