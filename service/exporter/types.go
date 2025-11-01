package exporter

import (
	"context"
	"io"
)

type Exporter interface {
	Export(ctx context.Context, competitionID uint64, writer io.Writer) error
}
