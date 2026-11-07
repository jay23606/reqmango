package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/reqmango/backend/internal/common"
	"github.com/reqmango/backend/internal/dto/request"
	"github.com/reqmango/backend/internal/model"
	"github.com/reqmango/backend/internal/service"
)

type TimeTrackHandler struct{ svc *service.TimeTrackService }

func NewTimeTrackHandler(svc *service.TimeTrackService) *TimeTrackHandler {
	return &TimeTrackHandler{svc: svc}
}

func (h *TimeTrackHandler) getUserID(c *gin.Context) uint64 {
	u, _ := c.Get("currentUser")
	if u, ok := u.(*model.User); ok {
		return u.ID
	}
	return 0
}

func (h *TimeTrackHandler) Create(c *gin.Context) {
	issueID, _ := strconv.ParseUint(c.Param("issueId"), 10, 64)
	var req request.TimeTrackCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}
	r, err := h.svc.Create(issueID, h.getUserID(c), &req)
	if err != nil {
		common.RespondError(c, err)
		return
	}
	c.JSON(201, r)
}

func (h *TimeTrackHandler) Start(c *gin.Context) {
	issueID, _ := strconv.ParseUint(c.Param("issueId"), 10, 64)
	var req request.TimeTrackStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"message": err.Error()})
		return
	}
	r, err := h.svc.Start(issueID, h.getUserID(c), &req)
	if err != nil {
		common.RespondError(c, err)
		return
	}
	c.JSON(201, r)
}

func (h *TimeTrackHandler) Stop(c *gin.Context) {
	issueID, _ := strconv.ParseUint(c.Param("issueId"), 10, 64)
	r, err := h.svc.Stop(issueID, h.getUserID(c))
	if err != nil {
		common.RespondError(c, err)
		return
	}
	c.JSON(200, r)
}

func (h *TimeTrackHandler) List(c *gin.Context) {
	issueID, _ := strconv.ParseUint(c.Param("issueId"), 10, 64)
	r, err := h.svc.List(issueID)
	if err != nil {
		common.RespondError(c, err)
		return
	}
	c.JSON(200, r)
}

func (h *TimeTrackHandler) Summary(c *gin.Context) {
	issueID, _ := strconv.ParseUint(c.Param("issueId"), 10, 64)
	r, err := h.svc.Summary(issueID)
	if err != nil {
		common.RespondError(c, err)
		return
	}
	c.JSON(200, r)
}

func (h *TimeTrackHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.svc.Delete(id, h.getUserID(c)); err != nil {
		common.RespondError(c, err)
		return
	}
	c.JSON(200, gin.H{"message": "Time entry deleted"})
}

// ProjectSummary returns aggregated time tracking stats for a project.
func (h *TimeTrackHandler) ProjectSummary(c *gin.Context) {
	projectID, _ := strconv.ParseUint(c.Param("projectId"), 10, 64)
	r, err := h.svc.ProjectSummary(projectID)
	if err != nil {
		common.RespondError(c, err)
		return
	}
	c.JSON(200, r)
}
