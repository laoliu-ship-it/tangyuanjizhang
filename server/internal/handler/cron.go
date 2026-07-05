package handler

import (
	"net/http"

	"fandianjizhang/server/internal/cron"
	"fandianjizhang/server/internal/dto"

	"github.com/gin-gonic/gin"
)

// CronHandler 平台管理员的定时任务管理接口
type CronHandler struct {
	manager *cron.Manager
}

func NewCronHandler(manager *cron.Manager) *CronHandler {
	return &CronHandler{manager: manager}
}

// ListJobs 查询所有定时任务的状态
// GET /api/platform/cron/jobs
func (h *CronHandler) ListJobs(c *gin.Context) {
	statuses := h.manager.List()
	items := make([]*dto.CronJobItem, 0, len(statuses))
	for _, s := range statuses {
		items = append(items, &dto.CronJobItem{
			Name:         s.Name,
			Description:  s.Description,
			Schedule:     s.Schedule,
			Running:      s.Running,
			NextRunAt:    s.NextRunAt,
			LastRunAt:    s.LastRunAt,
			LastDuration: s.LastDuration,
			LastError:    s.LastError,
			RunCount:     s.RunCount,
			FailCount:    s.FailCount,
		})
	}
	c.JSON(http.StatusOK, dto.OK(dto.CronJobListResp{Jobs: items}))
}

// TriggerJob 手动触发一次指定任务
// POST /api/platform/cron/jobs/:name/run
func (h *CronHandler) TriggerJob(c *gin.Context) {
	name := c.Param("name")
	if err := h.manager.TriggerNow(name); err != nil {
		c.JSON(http.StatusOK, dto.Fail(404, err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.OK(dto.CronTriggerResp{Name: name, Triggered: true}))
}
