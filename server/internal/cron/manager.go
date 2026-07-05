// Package cron 统一管理后台定时任务：注册、调度、状态查询、手动触发。
// 所有周期性后台任务必须通过 Manager 注册，禁止在 main.go 里散落裸 goroutine。
package cron

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// JobFunc 定时任务执行体
type JobFunc func(ctx context.Context) error

// scheduleKind 调度类型
type scheduleKind int

const (
	kindInterval scheduleKind = iota // 固定间隔
	kindDaily                        // 每天固定时刻
)

// job 已注册的任务及其运行时状态
type job struct {
	name        string
	description string
	kind        scheduleKind
	interval    time.Duration // kindInterval 用
	hour        int           // kindDaily 用
	minute      int           // kindDaily 用
	runAtStart  bool          // 启动时先执行一次
	timeout     time.Duration
	fn          JobFunc

	mu           sync.Mutex
	running      bool
	nextRunAt    time.Time
	lastRunAt    time.Time
	lastDuration time.Duration
	lastError    string
	runCount     int64
	failCount    int64
}

// JobStatus 任务状态快照（供管理后台展示）
type JobStatus struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Schedule     string `json:"schedule"`       // 人类可读的调度描述
	Running      bool   `json:"running"`        // 是否正在执行
	NextRunAt    string `json:"next_run_at"`    // 下次执行时间
	LastRunAt    string `json:"last_run_at"`    // 上次执行时间，空=从未执行
	LastDuration string `json:"last_duration"`  // 上次执行耗时
	LastError    string `json:"last_error"`     // 上次执行错误，空=成功
	RunCount     int64  `json:"run_count"`      // 累计执行次数
	FailCount    int64  `json:"fail_count"`     // 累计失败次数
}

// Manager 定时任务管理器
type Manager struct {
	mu      sync.Mutex
	jobs    []*job
	started bool
	cancel  context.CancelFunc
}

func NewManager() *Manager {
	return &Manager{}
}

// RegisterInterval 注册固定间隔任务；runAtStart=true 时启动后立即执行一次
func (m *Manager) RegisterInterval(name, description string, interval, timeout time.Duration, runAtStart bool, fn JobFunc) {
	m.register(&job{
		name: name, description: description,
		kind: kindInterval, interval: interval,
		runAtStart: runAtStart, timeout: timeout, fn: fn,
	})
}

// RegisterDaily 注册每天 hour:minute 执行的任务
func (m *Manager) RegisterDaily(name, description string, hour, minute int, timeout time.Duration, fn JobFunc) {
	m.register(&job{
		name: name, description: description,
		kind: kindDaily, hour: hour, minute: minute,
		timeout: timeout, fn: fn,
	})
}

func (m *Manager) register(j *job) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started {
		log.Printf("[cron] 任务 %s 注册失败：Manager 已启动", j.name)
		return
	}
	for _, existing := range m.jobs {
		if existing.name == j.name {
			log.Printf("[cron] 任务 %s 重复注册，忽略", j.name)
			return
		}
	}
	m.jobs = append(m.jobs, j)
}

// Start 启动所有已注册任务的调度循环
func (m *Manager) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started {
		return
	}
	m.started = true
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	for _, j := range m.jobs {
		go m.loop(ctx, j)
		log.Printf("[cron] 任务 %s 已启动（%s）", j.name, j.scheduleDesc())
	}
}

// Stop 停止调度（不打断正在执行中的任务）
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancel != nil {
		m.cancel()
	}
}

func (m *Manager) loop(ctx context.Context, j *job) {
	if j.runAtStart {
		m.execute(j, "startup")
	}
	for {
		next := j.nextTime(time.Now())
		j.mu.Lock()
		j.nextRunAt = next
		j.mu.Unlock()
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			m.execute(j, "schedule")
		}
	}
}

// nextTime 计算下次执行时间
func (j *job) nextTime(now time.Time) time.Time {
	switch j.kind {
	case kindDaily:
		next := time.Date(now.Year(), now.Month(), now.Day(), j.hour, j.minute, 0, 0, now.Location())
		if !next.After(now) {
			next = next.Add(24 * time.Hour)
		}
		return next
	default:
		return now.Add(j.interval)
	}
}

func (j *job) scheduleDesc() string {
	switch j.kind {
	case kindDaily:
		return fmt.Sprintf("每天 %02d:%02d", j.hour, j.minute)
	default:
		return fmt.Sprintf("每 %s", j.interval)
	}
}

// execute 执行一次任务并记录状态；同一任务不允许并发执行
func (m *Manager) execute(j *job, trigger string) {
	j.mu.Lock()
	if j.running {
		j.mu.Unlock()
		log.Printf("[cron] 任务 %s 上一轮仍在执行，跳过本次（trigger=%s）", j.name, trigger)
		return
	}
	j.running = true
	j.mu.Unlock()

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), j.timeout)
	err := j.fn(ctx)
	cancel()
	elapsed := time.Since(start)

	j.mu.Lock()
	j.running = false
	j.lastRunAt = start
	j.lastDuration = elapsed
	j.runCount++
	if err != nil {
		j.failCount++
		j.lastError = err.Error()
		log.Printf("[cron] 任务 %s 执行失败（耗时 %s）: %v", j.name, elapsed, err)
	} else {
		j.lastError = ""
	}
	j.mu.Unlock()
}

// TriggerNow 手动触发一次任务（管理后台用），任务不存在返回 error
func (m *Manager) TriggerNow(name string) error {
	m.mu.Lock()
	var target *job
	for _, j := range m.jobs {
		if j.name == name {
			target = j
			break
		}
	}
	m.mu.Unlock()
	if target == nil {
		return fmt.Errorf("定时任务不存在: %s", name)
	}
	go m.execute(target, "manual")
	return nil
}

// List 返回所有任务的状态快照
func (m *Manager) List() []JobStatus {
	m.mu.Lock()
	jobs := make([]*job, len(m.jobs))
	copy(jobs, m.jobs)
	m.mu.Unlock()

	const layout = "2006-01-02 15:04:05"
	result := make([]JobStatus, 0, len(jobs))
	for _, j := range jobs {
		j.mu.Lock()
		s := JobStatus{
			Name:        j.name,
			Description: j.description,
			Schedule:    j.scheduleDesc(),
			Running:     j.running,
			RunCount:    j.runCount,
			FailCount:   j.failCount,
			LastError:   j.lastError,
		}
		if !j.nextRunAt.IsZero() {
			s.NextRunAt = j.nextRunAt.Format(layout)
		}
		if !j.lastRunAt.IsZero() {
			s.LastRunAt = j.lastRunAt.Format(layout)
			s.LastDuration = j.lastDuration.Round(time.Millisecond).String()
		}
		j.mu.Unlock()
		result = append(result, s)
	}
	return result
}
