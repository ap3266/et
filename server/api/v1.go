package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/wrfly/et/notify"
	"github.com/wrfly/et/server/asset"
	"github.com/wrfly/et/storage"
	"github.com/wrfly/et/types"
)

const (
	fileName = "png/pixel.png"
	timeZone = "Asia/Shanghai"
)

var (
	pngFile, _ = asset.Asset(fileName)
	local, _   = time.LoadLocation(timeZone)
)

type Handler struct {
	n notify.Notifier
	s storage.Database
}

func New(n notify.Notifier, s storage.Database) *Handler {
	return &Handler{
		n: n,
		s: s,
	}
}

func (h *Handler) Open(c *gin.Context) {
	defer func() {
		c.Header("content-type", "image/png")
		c.Header("content-length", "126")
		c.Writer.Write(pngFile)
	}()

	var (
		taskID = c.Param("taskID")
		ip     = c.ClientIP()
		ua     = c.Request.UserAgent()
	)
	if len(taskID) > 40 {
		return
	}

	go func() {
		task, err := h.s.FindTask(taskID)
		if err != nil {
			logrus.Warnf("find task [%s] error: %s", taskID, err)
			return
		}
		if task.Opentimes > TaskLimit {
			logrus.Warnf("task [%s] open too many times", taskID)
			return
		}

		n := types.Notification{
			TaskID: taskID,
			Event: types.OpenEvent{
				IP:   ip,
				UA:   ua,
				Time: time.Now().Add(task.Adjust),
			},
		}
		n.ID = genNotificationID(n)

		// save notification
		if err := h.s.SaveNotification(n); err != nil {
			logrus.Errorf("save notification error: %s", err)
		}

		logrus.Debugf("send notification to %s", task.NotifyTo)
		body, code, err := h.n.Send(task.NotifyTo, notify.NewContent(n, task.Comments))
		if err != nil {
			logrus.Errorf("send notification err: %s", err)
		}

		// sendgrid service returns 202
		if code != http.StatusAccepted {
			logrus.Errorf("handle task [%+v], body: %s, code: %d",
				task, body, code)
		}

		logrus.Debugf("send notification %+v done", n)
	}()
}

func (h *Handler) Submit(c *gin.Context) {
	r := taskRequest{}
	if err := c.BindJSON(&r); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, taskResponse{
			Error: err.Error(),
		})
		return
	}
	if r.LocalTime.IsZero() {
		r.LocalTime = time.Now().In(local)
	}

	t := types.Task{
		NotifyTo: r.NotifyTo,
		Comments: r.Comments,
		SubmitAt: r.LocalTime,
		Adjust: r.LocalTime.Sub(time.Now()).
			Truncate(time.Second),
	}

	if !validTask(t) {
		c.AbortWithStatusJSON(http.StatusBadRequest, taskResponse{
			Error: "bad request, check your email address and comments",
		})
		return
	}

	t.ID = genTaskID(t)
	logrus.Debugf("submit task [%s], notify to [%s] with comments [%s]",
		t.ID, t.NotifyTo, t.Comments)
	logrus.Debugf("submitAt: %s, adjust: %s", t.SubmitAt, t.Adjust)
	if err := h.s.SaveTask(t); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, taskResponse{
			Error: fmt.Sprintf("save task error: %s", err),
		})
		return
	}

	c.JSON(http.StatusOK, taskResponse{
		TaskID:    t.ID,
		TrackLink: fmt.Sprintf("%s/t/%s", DomainPrefix, t.ID),
	})
}
