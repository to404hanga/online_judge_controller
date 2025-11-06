package job

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/to404hanga/pkg404/logger"
	loggerv2 "github.com/to404hanga/pkg404/logger/v2"
)

// JobFunc 定义任务执行函数类型
type JobFunc func(ctx context.Context) error

// JobConfig 任务配置
type JobConfig struct {
	Name        string        // 任务名称
	CronExpr    string        // cron表达式
	JobFunc     JobFunc       // 任务执行函数
	Description string        // 任务描述
	Enabled     bool          // 是否启用
	Timeout     time.Duration // 任务超时时间
}

// JobStatus 任务状态
type JobStatus struct {
	Name         string        `json:"name"`
	CronExpr     string        `json:"cron_expr"`
	Description  string        `json:"description"`
	Enabled      bool          `json:"enabled"`
	LastRun      *time.Time    `json:"last_run,omitempty"`
	NextRun      *time.Time    `json:"next_run,omitempty"`
	LastDuration time.Duration `json:"last_duration"`
	LastError    string        `json:"last_error,omitempty"`
	RunCount     int64         `json:"run_count"`
	ErrorCount   int64         `json:"error_count"`
}

// CronScheduler cron定时任务调度器
type CronScheduler struct {
	cron        *cron.Cron
	jobs        map[string]*JobConfig
	jobStatuses map[string]*JobStatus
	log         loggerv2.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
}

// NewCronScheduler 创建新的cron调度器
func NewCronScheduler(log loggerv2.Logger) *CronScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建cron实例，支持秒级精度
	c := cron.New(cron.WithSeconds())

	scheduler := &CronScheduler{
		cron:        c,
		jobs:        make(map[string]*JobConfig),
		jobStatuses: make(map[string]*JobStatus),
		log:         log,
		ctx:         ctx,
		cancel:      cancel,
	}

	return scheduler
}

// AddJob 添加任务
func (s *CronScheduler) AddJob(config *JobConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if config.Name == "" {
		return fmt.Errorf("job name cannot be empty")
	}

	if config.CronExpr == "" {
		return fmt.Errorf("cron expression cannot be empty")
	}

	if config.JobFunc == nil {
		return fmt.Errorf("job function cannot be nil")
	}

	if config.Timeout == 0 {
		config.Timeout = 10 * time.Minute // 默认超时时间
	}

	// 验证cron表达式
	_, err := s.cron.AddFunc(config.CronExpr, func() {})
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	s.jobs[config.Name] = config
	s.jobStatuses[config.Name] = &JobStatus{
		Name:        config.Name,
		CronExpr:    config.CronExpr,
		Description: config.Description,
		Enabled:     config.Enabled,
	}

	s.log.InfoContext(s.ctx, "Job added",
		logger.String("name", config.Name),
		logger.String("cronExpr", config.CronExpr),
		logger.Bool("enabled", config.Enabled),
	)

	return nil
}

// RemoveJob 移除任务
func (s *CronScheduler) RemoveJob(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[name]; !exists {
		return fmt.Errorf("job %s not found", name)
	}

	delete(s.jobs, name)
	delete(s.jobStatuses, name)

	s.log.InfoContext(s.ctx, "Job removed", logger.String("name", name))
	return nil
}

// EnableJob 启用任务
func (s *CronScheduler) EnableJob(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[name]
	if !exists {
		return fmt.Errorf("job %s not found", name)
	}

	job.Enabled = true
	if status, ok := s.jobStatuses[name]; ok {
		status.Enabled = true
	}

	s.log.InfoContext(s.ctx, "Job enabled", logger.String("name", name))
	return nil
}

// DisableJob 禁用任务
func (s *CronScheduler) DisableJob(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[name]
	if !exists {
		return fmt.Errorf("job %s not found", name)
	}

	job.Enabled = false
	if status, ok := s.jobStatuses[name]; ok {
		status.Enabled = false
	}

	s.log.InfoContext(s.ctx, "Job disabled", logger.String("name", name))
	return nil
}

// Start 启动调度器
func (s *CronScheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 清空现有任务
	s.cron.Stop()
	s.cron = cron.New(cron.WithSeconds())

	// 添加所有启用的任务
	for name, job := range s.jobs {
		if job.Enabled {
			_, err := s.cron.AddFunc(job.CronExpr, s.wrapJobFunc(name, job))
			if err != nil {
				s.log.ErrorContext(s.ctx, "Failed to add job to cron",
					logger.String("name", name),
					logger.Error(err),
				)
				continue
			}

			// 更新下次运行时间
			if status, ok := s.jobStatuses[name]; ok {
				entries := s.cron.Entries()
				for _, entry := range entries {
					nextRun := entry.Next
					status.NextRun = &nextRun
					break
				}
			}
		}
	}

	s.cron.Start()
	s.log.InfoContext(s.ctx, "Cron scheduler started")
	return nil
}

// Stop 停止调度器
func (s *CronScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cron.Stop()
	s.cancel()
	s.log.InfoContext(s.ctx, "Cron scheduler stopped")
}

// wrapJobFunc 包装任务函数，添加日志、超时、统计等功能
func (s *CronScheduler) wrapJobFunc(name string, job *JobConfig) func() {
	return func() {
		startTime := time.Now()

		s.mu.Lock()
		status := s.jobStatuses[name]
		status.LastRun = &startTime
		status.RunCount++
		s.mu.Unlock()

		s.log.InfoContext(s.ctx, "Job started", logger.String("name", name))

		// 创建带超时的上下文
		ctx, cancel := context.WithTimeout(s.ctx, job.Timeout)
		defer cancel()

		// 执行任务
		err := job.JobFunc(ctx)

		duration := time.Since(startTime)

		s.mu.Lock()
		status.LastDuration = duration
		if err != nil {
			status.ErrorCount++
			status.LastError = err.Error()
			s.log.ErrorContext(s.ctx, "Job failed",
				logger.String("name", name),
				logger.Int64("duration_ns", duration.Nanoseconds()),
				logger.Error(err),
			)
		} else {
			status.LastError = ""
			s.log.InfoContext(s.ctx, "Job completed",
				logger.String("name", name),
				logger.Int64("duration_ns", duration.Nanoseconds()),
			)
		}
		s.mu.Unlock()
	}
}

// GetJobStatuses 获取所有任务状态
func (s *CronScheduler) GetJobStatuses() map[string]*JobStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*JobStatus)
	for name, status := range s.jobStatuses {
		// 深拷贝状态
		statusCopy := *status
		result[name] = &statusCopy
	}

	return result
}

// GetJobStatus 获取指定任务状态
func (s *CronScheduler) GetJobStatus(name string) (*JobStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status, exists := s.jobStatuses[name]
	if !exists {
		return nil, fmt.Errorf("job %s not found", name)
	}

	// 返回副本
	statusCopy := *status
	return &statusCopy, nil
}

// RunJobOnce 手动执行一次任务
func (s *CronScheduler) RunJobOnce(name string) error {
	s.mu.RLock()
	job, exists := s.jobs[name]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("job %s not found", name)
	}

	s.log.InfoContext(s.ctx, "Running job manually", logger.String("name", name))

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(s.ctx, job.Timeout)
	defer cancel()

	return job.JobFunc(ctx)
}
