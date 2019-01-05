package api

import (
	"time"

	"github.com/gin-gonic/gin"
)

func (h *Handler) preStatus(c *gin.Context) {
	c.Header("content-type", "image/svg+xml;charset=utf-8")
	c.Header("date", time.Now().String())
}

func (h *Handler) StatusTask(c *gin.Context) {
	h.preStatus(c)
	status := h.s.Status()
	if _, found := c.GetQuery("daily"); found {
		greenBadge(c.Writer, "daily task", status.Daily.TaskSubmit)
	} else {
		blueBadge(c.Writer, "total task", status.Total.TaskSubmit)
	}
}

func (h *Handler) StatusNotified(c *gin.Context) {
	h.preStatus(c)
	status := h.s.Status()
	if _, found := c.GetQuery("daily"); found {
		greenBadge(c.Writer, "daily notified", status.Daily.Notification)
	} else {
		blueBadge(c.Writer, "total notified", status.Total.Notification)
	}
}
