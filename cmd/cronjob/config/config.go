package config

type BaseCronJobConfig struct {
	CronExpr string `yaml:"cronExpr"`
	Enabled  bool   `yaml:"enabled"`
	Timeout  int    `yaml:"timeout"` // 单位: 毫秒
}

type MinIOCleanerConfig struct {
	BaseCronJobConfig `yaml:",inline"`

	Bucket              string `yaml:"bucket"`
	OrphanFileCheckDays int    `yaml:"orphanFileCheckDays"` // 单位: 天
}

func (MinIOCleanerConfig) Key() string {
	return "minIOCleaner"
}

type SubmissionCleanerConfig struct {
	BaseCronJobConfig `yaml:",inline"`

	TimeRange int `yaml:"timeRange"` // 单位: 天
}

func (SubmissionCleanerConfig) Key() string {
	return "submissionCleaner"
}
