package controllerImp

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"aoi/pkg/plan/serviceImp"
	fieldrepo "aoi/pkg/field/repository"
	fieldRepoImp "aoi/pkg/field/repositoryImp"
	"gorm.io/gorm"
)


type PlanCtrl struct{ svc *serviceImp.PlanSvc; fields fieldrepo.FieldRepository }

func NewPlanCtrl(db *gorm.DB, svc *serviceImp.PlanSvc) *PlanCtrl { return &PlanCtrl{svc: svc, fields: fieldRepoImp.New(db)} }

func (h *PlanCtrl) Generate(c echo.Context) error {
	uid := c.Get("uid").(string)
	fid, _ := strconv.Atoi(c.Param("id"))
	f, err := h.fields.FindByID(uint(fid), uid)
	if err != nil { return c.JSON(http.StatusNotFound, map[string]string{"error":"field not found"}) }
	p, tasks, err := h.svc.GenerateFirstPlan(f)
	if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
	if c.QueryParam("format") == "calendar" {

		kbdebug := c.QueryParam("kbdebug") == "1"

		type CalItem struct {
			TaskID uint     `json:"task_id"`
			Type   string   `json:"type"`
			Title  string   `json:"title"`
			Qty    *float64 `json:"qty,omitempty"`
			Unit   string   `json:"unit,omitempty"`
			Notes  string   `json:"notes,omitempty"`
			Status string   `json:"status"`
		}
		cal := map[string][]CalItem{} // "YYYY-MM-DD" -> items
		for _, t := range tasks {
			ds := t.Date.Format("2006-01-02")
			cal[ds] = append(cal[ds], CalItem{
				TaskID: t.TaskID, Type: t.Type, Title: t.Title,
				Qty: t.Qty, Unit: t.Unit, Notes: t.Notes, Status: t.Status,
			})
		}
		resp := map[string]any{
			"field_id": f.FieldID,
			"plan_id":  p.PlanID,
			"version":  p.Version,
			"calendar": cal,
		}
		if kbdebug {
			resp["kb_refs"] = serviceImp.LastKBRefs()
		}
		return c.JSON(http.StatusCreated, resp)
	}
	return c.JSON(http.StatusCreated, map[string]any{"plan": p, "tasks": tasks})
}

func (h *PlanCtrl) Replan(c echo.Context) error {
    uid := c.Get("uid").(string)
    fid, _ := strconv.Atoi(c.Param("id"))
    f, err := h.fields.FindByID(uint(fid), uid)
    if err != nil {
        return c.JSON(http.StatusNotFound, map[string]string{"error": "field not found"})
    }

    // Bind body FIRST
    var body struct {
        Reason   string   `json:"reason"`
        Problems []string `json:"problems"`
    }
    if err := c.Bind(&body); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "bad json"})
    }

    // Call the wrapper that applies problems -> KB -> LLM actions
    p, tasks, rep, err := h.svc.ReplanWithOptions(f, serviceImp.ReplanOptions{
        Reason:   strings.TrimSpace(body.Reason),
        Problems: body.Problems,
    })
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
    }

    if c.QueryParam("format") == "calendar" {
        kbdebug := c.QueryParam("kbdebug") == "1"

        type CalItem struct {
            TaskID uint     `json:"task_id"`
            Type   string   `json:"type"`
            Title  string   `json:"title"`
            Qty    *float64 `json:"qty,omitempty"`
            Unit   string   `json:"unit,omitempty"`
            Notes  string   `json:"notes,omitempty"`
            Status string   `json:"status"`
        }
        cal := map[string][]CalItem{} // "YYYY-MM-DD" -> items
        for _, t := range tasks {
            ds := t.Date.Format("2006-01-02")
            cal[ds] = append(cal[ds], CalItem{
                TaskID: t.TaskID,
                Type:   t.Type,
                Title:  t.Title,
                Qty:    t.Qty,
                Unit:   t.Unit,
                Notes:  t.Notes,
                Status: t.Status,
            })
        }

        resp := map[string]any{
            "field_id": f.FieldID,
            "plan_id":  p.PlanID,
            "version":  p.Version,
            "calendar": cal,
            "replan":   rep, // <- use rep
        }
        if kbdebug {
            resp["kb_refs"] = serviceImp.LastKBRefs()
        }
        return c.JSON(http.StatusOK, resp)
    }

    // default (non-calendar) response
    return c.JSON(http.StatusOK, map[string]any{
        "plan":   p,
        "tasks":  tasks,
        "replan": rep, // <- use rep
    })
}

func (h *PlanCtrl) List(c echo.Context) error {
	// TODO: wire to service/repo (latest plan + tasks) when ready.
	// For now, return an empty list to keep the contract intact.
	return c.JSON(http.StatusOK, json.RawMessage(`[]`))
}