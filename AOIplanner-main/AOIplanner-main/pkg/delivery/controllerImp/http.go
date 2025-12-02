package controllerImp

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"aoi/pkg/delivery"
	dsvc "aoi/pkg/delivery/service"
)

type httpCtrl struct{ s dsvc.Service }

func New(s dsvc.Service) *httpCtrl { return &httpCtrl{s: s} }

func (h *httpCtrl) Register(e *echo.Echo) {
    // มี prefix
   g := e.Group("/api/v1")
g.POST("/fields/:field_id/deliveries", h.create)
g.GET("/fields/:field_id/deliveries", h.list)
g.PATCH("/deliveries/:id", h.patch)

// Fallback (ไม่มี prefix) – กันกรณีฝั่งเว็บเรียก /fields/...
e.POST("/fields/:field_id/deliveries", h.create)
e.GET("/fields/:field_id/deliveries", h.list)
e.PATCH("/deliveries/:id", h.patch)

}



func (h *httpCtrl) create(c echo.Context) error {
	fieldID, err := parseUint(c.Param("field_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid field_id"})
	}
	var in delivery.Delivery
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid json"})
	}
	in.FieldID = uint(fieldID)
	if err := h.s.Create(&in); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, in)
}

func (h *httpCtrl) list(c echo.Context) error {
	fieldID, err := parseUint(c.Param("field_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid field_id"})
	}
	var fromPtr, toPtr *time.Time
	if v := c.QueryParam("from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			fromPtr = &t
		}
	}
	if v := c.QueryParam("to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			toPtr = &t
		}
	}
	list, err := h.s.ListByField(uint(fieldID), fromPtr, toPtr)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, list)
}

func (h *httpCtrl) patch(c echo.Context) error {
	id, err := parseUint(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}
	var in dsvc.DeliveryPatch
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid json"})
	}
	out, err := h.s.UpdatePartial(uint(id), in)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, out)
}

func parseUint(s string) (uint64, error) {
	return strconv.ParseUint(s, 10, 64)
}
