package scheduler

import (
	"ai_push_message/config"
	"ai_push_message/logger"
	"ai_push_message/repository"
	"ai_push_message/services"
	"fmt"
	"sync"
	"time"
)

// 将秒数转换为时间间隔
func secondsToDuration(seconds int) time.Duration {
	return time.Duration(seconds) * time.Second
}

// 验证小时和分钟是否有效
func validateHourMinute(cfg *config.Config, hour, minute int) (int, int) {
	defaultHour := cfg.Scheduler.DefaultHour
	defaultMinute := cfg.Scheduler.DefaultMinute

	if hour < 0 || hour > 23 {
		logger.Warn("无效的小时值", "hour", hour, "default", defaultHour)
		hour = defaultHour
	}
	if minute < 0 || minute > 59 {
		logger.Warn("无效的分钟值", "minute", minute, "default", defaultMinute)
		minute = defaultMinute
	}
	return hour, minute
}

// 计算下一个指定时间点
func getNextTimePoint(now time.Time, hour, minute int) time.Time {
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if next.Before(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

// 任务类型
type TaskType int

const (
	TaskPush TaskType = iota
	TaskProfileGeneration
)

// 任务状态
type TaskStatus struct {
	LastRun     time.Time
	NextRun     time.Time
	IsRunning   bool
	Description string
}

// 任务调度器
type Scheduler struct {
	cfg         *config.Config
	concurrency int
	tasks       map[TaskType]*TaskStatus
	mutex       sync.Mutex
}

// 创建新的调度器
func NewScheduler(cfg *config.Config) *Scheduler {
	concurrency := cfg.Cron.Concurrency
	if concurrency <= 0 {
		concurrency = 10
	}

	return &Scheduler{
		cfg:         cfg,
		concurrency: concurrency,
		tasks:       make(map[TaskType]*TaskStatus),
		mutex:       sync.Mutex{},
	}
}

// 启动调度器
func Start(cfg *config.Config) {
	scheduler := NewScheduler(cfg)

	// 初始化任务
	scheduler.initTasks()

	// 启动主循环
	go scheduler.run()

	checkInterval := cfg.Scheduler.CheckIntervalSec
	if checkInterval <= 0 {
		checkInterval = 60 // 默认值
	}
	logger.Info("调度器已启动", "check_interval_sec", checkInterval)
}

// 初始化任务
func (s *Scheduler) initTasks() {
	now := time.Now()

	// 用户画像生成任务 - 根据debug模式决定运行频率
	if s.cfg.Debug.Enabled {
		// Debug模式：按配置的秒数间隔生成一次，执行完整流程（画像生成 → 推荐生成 → 推送）
		freqSeconds := s.cfg.Debug.RecommendationFreq
		profileInterval := time.Duration(freqSeconds) * time.Second
		nextProfileRun := now.Add(profileInterval)

		s.tasks[TaskProfileGeneration] = &TaskStatus{
			LastRun:     now.Add(-profileInterval),
			NextRun:     nextProfileRun,
			IsRunning:   false,
			Description: fmt.Sprintf("完整推荐流程 (Debug模式: 每%d秒)", freqSeconds),
		}
		logger.Info("Debug模式已启用", "frequency_seconds", freqSeconds, "workflow", "画像生成 → 推荐生成 → 推送")
	} else {
		// 正常模式：每天在指定时间点运行完整流程
		hour, minute := validateHourMinute(s.cfg, s.cfg.Cron.ProfileHour, s.cfg.Cron.ProfileMin)
		nextProfileRun := getNextTimePoint(now, hour, minute)

		s.tasks[TaskProfileGeneration] = &TaskStatus{
			LastRun:     nextProfileRun.Add(-24 * time.Hour),
			NextRun:     nextProfileRun,
			IsRunning:   false,
			Description: fmt.Sprintf("完整推荐流程 (%02d:%02d)", hour, minute),
		}
		logger.Info("正常模式", "schedule_time", fmt.Sprintf("%02d:%02d", hour, minute), "workflow", "画像生成 → 推荐生成 → 推送")
	}

	logger.Info("定时任务初始化完成", "task_count", len(s.tasks))
}

// 主循环
func (s *Scheduler) run() {
	checkInterval := s.cfg.Scheduler.CheckIntervalSec
	if checkInterval <= 0 {
		checkInterval = 60 // 默认值
	}
	ticker := time.NewTicker(time.Duration(checkInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case now := <-ticker.C:
			s.checkTasks(now)
		}
	}
}

// 检查任务
func (s *Scheduler) checkTasks(now time.Time) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for taskType, status := range s.tasks {
		// 如果任务正在运行，跳过
		if status.IsRunning {
			continue
		}

		// 如果任务的NextRun为零值，跳过（表示不需要定期调度）
		if status.NextRun.IsZero() {
			continue
		}

		// 如果到达或超过下次运行时间，执行任务
		if now.After(status.NextRun) || now.Equal(status.NextRun) {
			status.IsRunning = true
			go s.runTask(taskType, now)
		}
	}
}

// 运行任务
func (s *Scheduler) runTask(taskType TaskType, now time.Time) {
	defer func() {
		s.mutex.Lock()
		defer s.mutex.Unlock()

		status := s.tasks[taskType]
		status.IsRunning = false
		status.LastRun = now

		// 更新下次运行时间
		switch taskType {
		case TaskProfileGeneration:
			if s.cfg.Debug.Enabled {
				// Debug模式：按配置的秒数间隔
				freqSeconds := s.cfg.Debug.RecommendationFreq
				if freqSeconds <= 0 {
					freqSeconds = 1800
				}
				profileInterval := time.Duration(freqSeconds) * time.Second
				status.NextRun = now.Add(profileInterval)
			} else {
				// 正常模式：获取下一个每日时间点
				hour, minute := validateHourMinute(s.cfg, s.cfg.Cron.ProfileHour, s.cfg.Cron.ProfileMin)
				status.NextRun = getNextTimePoint(now, hour, minute)
			}
		}

		logger.Info("任务执行完成", "task", status.Description, "next_run", status.NextRun.Format("2006-01-02 15:04:05"))
	}()

	logger.Info("开始执行任务", "task", func() string {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		if status, ok := s.tasks[taskType]; ok {
			return status.Description
		}
		return "Unknown Task"
	}())

	switch taskType {
	case TaskProfileGeneration:
		// 执行完整推荐流程：画像生成 → 推荐生成 → 推送
		// 获取所有候选用户
		cids, err := repository.ListCandidateCIDs(s.cfg.Cron.LookbackDays)
		if err != nil {
			logger.Error("获取候选用户列表失败", "error", err)
			return
		}

		logger.Info("找到候选用户", "count", len(cids), "concurrency", s.concurrency, "workflow", "完整推荐流程")

		// 步骤1：使用并发控制生成用户画像
		logger.Info("[步骤1/3] 开始生成用户画像")
		services.GenerateProfilesWithConcurrency(s.cfg, cids, s.concurrency)
		logger.Info("[步骤1/3] 用户画像生成完成")

		// 步骤2：生成推荐内容
		logger.Info("[步骤2/3] 开始生成推荐内容")
		services.GenerateRecommendationsWithConcurrency(s.cfg, cids, s.concurrency)
		logger.Info("[步骤2/3] 推荐内容生成完成")

		// 步骤3：执行推送
		logger.Info("[步骤3/3] 开始执行推送任务")
		if err := services.PushAll(s.cfg); err != nil {
			logger.Error("[步骤3/3] 推送任务执行错误", "error", err)
		} else {
			logger.Info("[步骤3/3] 推送任务执行完成")
		}
		logger.Info("完整推荐流程执行完成")
	}
}
