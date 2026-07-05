package cron

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// Scenario: runAtStart=true 的任务启动后立即执行一次
func TestManager_RunAtStart(t *testing.T) {
	m := NewManager()
	var count atomic.Int64
	m.RegisterInterval("job_a", "测试任务", time.Hour, time.Second, true,
		func(ctx context.Context) error {
			count.Add(1)
			return nil
		})
	m.Start()
	defer m.Stop()

	deadline := time.Now().Add(2 * time.Second)
	for count.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if count.Load() != 1 {
		t.Fatalf("启动后应执行 1 次，实际 %d", count.Load())
	}
}

// Scenario: 固定间隔任务按间隔重复执行
func TestManager_IntervalRepeats(t *testing.T) {
	m := NewManager()
	var count atomic.Int64
	m.RegisterInterval("job_b", "测试任务", 50*time.Millisecond, time.Second, false,
		func(ctx context.Context) error {
			count.Add(1)
			return nil
		})
	m.Start()
	defer m.Stop()

	time.Sleep(220 * time.Millisecond)
	if got := count.Load(); got < 2 {
		t.Fatalf("220ms 内至少应执行 2 次，实际 %d", got)
	}
}

// Scenario: 手动触发执行任务并记录状态；不存在的任务报错
func TestManager_TriggerNow(t *testing.T) {
	m := NewManager()
	var count atomic.Int64
	m.RegisterInterval("job_c", "测试任务", time.Hour, time.Second, false,
		func(ctx context.Context) error {
			count.Add(1)
			return errors.New("模拟失败")
		})
	m.Start()
	defer m.Stop()

	if err := m.TriggerNow("job_c"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for count.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if count.Load() != 1 {
		t.Fatalf("手动触发应执行 1 次，实际 %d", count.Load())
	}

	// 状态记录了失败
	var status JobStatus
	deadline = time.Now().Add(time.Second)
	for status.RunCount == 0 && time.Now().Before(deadline) {
		for _, s := range m.List() {
			if s.Name == "job_c" && s.RunCount == 1 {
				status = s
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if status.RunCount != 1 {
		t.Fatal("List 中未找到已执行的 job_c")
	}
	if status.FailCount != 1 || status.LastError != "模拟失败" {
		t.Errorf("失败状态记录不正确: %+v", status)
	}

	if err := m.TriggerNow("不存在"); err == nil {
		t.Error("触发不存在的任务应报错")
	}
}

// Scenario: 重名任务只注册一次
func TestManager_DuplicateNameIgnored(t *testing.T) {
	m := NewManager()
	noop := func(ctx context.Context) error { return nil }
	m.RegisterInterval("dup", "第一个", time.Hour, time.Second, false, noop)
	m.RegisterInterval("dup", "第二个", time.Hour, time.Second, false, noop)
	if got := len(m.List()); got != 1 {
		t.Fatalf("重名任务应只保留 1 个，实际 %d", got)
	}
	if m.List()[0].Description != "第一个" {
		t.Error("应保留先注册的任务")
	}
}

// Scenario: 每天固定时刻任务的下次执行时间计算正确
func TestJob_NextTimeDaily(t *testing.T) {
	j := &job{kind: kindDaily, hour: 3, minute: 0}
	now := time.Date(2026, 7, 5, 10, 0, 0, 0, time.Local)
	next := j.nextTime(now)
	want := time.Date(2026, 7, 6, 3, 0, 0, 0, time.Local)
	if !next.Equal(want) {
		t.Errorf("10点之后应排到次日3点，实际 %v", next)
	}

	now = time.Date(2026, 7, 5, 1, 0, 0, 0, time.Local)
	next = j.nextTime(now)
	want = time.Date(2026, 7, 5, 3, 0, 0, 0, time.Local)
	if !next.Equal(want) {
		t.Errorf("凌晨1点应排到当日3点，实际 %v", next)
	}
}
