package common

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

const detailSql = `
WITH ranked_submissions AS (
    SELECT
        id,
        competition_id,
        user_id,
        problem_id,
        result,
        created_at,
        -- 为每个用户每个题目的所有提交按时间排序
        ROW_NUMBER() OVER (
            PARTITION BY competition_id, user_id, problem_id
            ORDER BY created_at, id
        ) AS submission_order,
        -- 为每个用户每个题目的Accepted提交按时间排序
        SUM(CASE WHEN result = 1 THEN 1 ELSE 0 END) OVER (
            PARTITION BY competition_id, user_id, problem_id
            ORDER BY created_at, id
            ROWS UNBOUNDED PRECEDING
        ) AS accepted_order
    FROM submission
    WHERE competition_id = ?
),
first_accepted AS (
    SELECT
        competition_id,
        user_id,
        problem_id,
        created_at AS accepted_time,
        submission_order - 1 AS attempts_before_accepted
    FROM ranked_submissions
    WHERE result = 1 AND accepted_order = 1
)
SELECT
    fa.competition_id AS competition_id,
    fa.user_id AS user_id,
    fa.problem_id AS problem_id,
    u.username AS username,
    u.realname AS realname,
    fa.accepted_time AS accepted_time,
    fa.attempts_before_accepted AS attempts_before_accepted
FROM first_accepted fa
LEFT JOIN user u ON fa.user_id = u.id
ORDER BY fa.user_id, fa.problem_id
`

type AcceptedDetail struct {
	CompetitionID          uint64    `gorm:"competition_id" json:"competition_id"`
	UserID                 uint64    `gorm:"user_id" json:"user_id"`
	ProblemID              uint64    `gorm:"problem_id" json:"problem_id"`
	Username               string    `gorm:"username" json:"username"`
	Realname               string    `gorm:"realname" json:"realname"`
	AcceptedTime           time.Time `gorm:"accepted_time" json:"accepted_time"`
	AttemptsBeforeAccepted int       `gorm:"attempts_before_accepted" json:"attempts_before_accepted"`
}

func FetchDetail(db *gorm.DB, ctx context.Context, competitionID uint64) ([]AcceptedDetail, error) {
	var details []AcceptedDetail
	err := db.WithContext(ctx).Raw(fmt.Sprintf(detailSql), competitionID).Scan(&details).Error
	if err != nil {
		return nil, fmt.Errorf("fetch detail failed: %w", err)
	}
	return details, nil
}

func (d *AcceptedDetail) GetAcceptTime() string {
	return d.AcceptedTime.Format("15:04:05.000")
}
