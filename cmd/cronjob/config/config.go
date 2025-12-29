package config

type BaseCronJobConfig struct {
	CronExpr string `yaml:"cronExpr" mapstructure:"cronExpr"`
	Enabled  bool   `yaml:"enabled" mapstructure:"enabled"`
	Timeout  int    `yaml:"timeout" mapstructure:"timeout"` // 单位: 毫秒
}

type SubmissionCleanerConfig struct {
	BaseCronJobConfig `yaml:",inline" mapstructure:",squash"`

	TimeRange int `yaml:"timeRange" mapstructure:"timeRange"` // 单位: 天
}

func (SubmissionCleanerConfig) Key() string {
	return "submissionCleaner"
}
