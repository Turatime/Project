package controllerImp

import (
	"net/http"
	"strconv"
	"github.com/labstack/echo/v4"
	repo "aoi/pkg/schedule/repository"
)

type SchedCtrl struct{ repo repo.ScheduleRepository }

func New(repo repo.ScheduleRepository) *SchedCtrl { return &SchedCtrl{repo} }

func (h *SchedCtrl) List(c echo.Context) error {
	fid, _ := strconv.Atoi(c.Param("id"))
	from := c.QueryParam("from")
	to := c.QueryParam("to")
	out, err := h.repo.List(uint(fid), from, to)
	if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	return c.JSON(http.StatusOK, out)
}

func (h *SchedCtrl) Patch(c echo.Context) error {
	tid, _ := strconv.Atoi(c.Param("task_id"))
	var body struct{ Status string `json:"status"`; Qty *float64 `json:"qty"` }
	if err := c.Bind(&body); err != nil { return c.JSON(http.StatusBadRequest, map[string]string{"error":"bad json"}) }
	if body.Status == "" { body.Status = "done" }
	if err := h.repo.PatchStatus(uint(tid), body.Status, body.Qty); err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	return c.JSON(http.StatusOK, map[string]string{"status":"ok"})
}