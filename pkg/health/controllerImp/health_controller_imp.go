package controllerImp

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

var appStart = time.Now()

type HealthCtrl struct {
	db *gorm.DB
}

func NewHealthCtrl(db *gorm.DB) *HealthCtrl { return &HealthCtrl{db: db} }

func (h *HealthCtrl) Health(c echo.Context) error {
	ctx, cancel := context.WithTimeout(c.Request().Context(), 800*time.Millisecond)
	defer cancel()

	dbOK := true
	dbErr := ""
	if h.db != nil {
		sqlDB, err := h.db.DB()
		if err != nil {
			dbOK = false
			dbErr = "db.DB(): " + err.Error()
		} else if err := sqlDB.PingContext(ctx); err != nil {
			dbOK = false
			dbErr = "ping: " + err.Error()
		}
	} else {
		dbOK = false
		dbErr = "gorm db is nil"
	}

	allOK := dbOK
	status := http.StatusOK
	if !allOK {
		status = http.StatusServiceUnavailable
	}

	type sub struct {
		OK  bool   `json:"ok"`
		Err string `json:"err,omitempty"`
	}

	resp := map[string]any{
		"status":     map[string]any{"ok": allOK},
		"uptime_sec": int(time.Since(appStart).Seconds()),
		"checks": map[string]any{
			"database": sub{OK: dbOK, Err: dbErr},
		},
		"time": time.Now().Format(time.RFC3339),
	}

	return c.JSON(status, resp)
}
