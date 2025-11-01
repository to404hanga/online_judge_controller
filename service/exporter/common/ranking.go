package common

import (
	"context"
	"fmt"

	ojmodel "github.com/to404hanga/online_judge_common/model"
	"gorm.io/gorm"
)

// FetchRanking 从数据库中获取排名数据
func FetchRanking(db *gorm.DB, ctx context.Context, competitionID uint64, page, limit int) ([]ojmodel.CompetitionUser, error) {
	var ranks []ojmodel.CompetitionUser
	if err := db.WithContext(ctx).
		Model(&ojmodel.CompetitionUser{}).
		Where("competition_id = ?", competitionID).
		Order("pass_count DESC, total_time ASC").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&ranks).Error; err != nil {
		return nil, fmt.Errorf("fetch ranking failed: %w", err)
	}
	return ranks, nil
}
