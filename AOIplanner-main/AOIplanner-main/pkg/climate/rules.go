package climate

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"time"
	"strings"

	"github.com/xuri/excelize/v2"

	"aoi/entities"
	"aoi/pkg/plan/types"
)

type RulesEngine interface {
	BuildStages(*entities.Field) []types.StagePlan
	ExpandDaily(*entities.Field, []types.StagePlan) []types.PlanOp
	ToSchedule(*entities.Field, uint, []types.PlanOp) []entities.ScheduleTask
	EvaluateDrift(*entities.Field, []entities.Measurement, []types.StagePlan) (bool, string)
}

type stageRow struct {
	Name         string
	Days         int
	WaterMMDay   float64
	IntervalDays int
	Notes        string
}

type rules struct {
	stageCfg []stageRow
	adj      map[string]float64       // crop_type -> factor
	soilIrr  map[string]int           // soil -> interval override
	fertTips map[string]string        // stage -> tip
}

func LoadFromFiles(stageCSV, cropAdjCSV, irrigXLSX string) (RulesEngine, error) {
	r := &rules{adj: map[string]float64{"new_plant":1.0, "ratoon":0.95}, soilIrr: map[string]int{}, fertTips: map[string]string{}}

	if stageCSV != "" { if err := r.loadStagesCSV(stageCSV); err != nil { return nil, err } }
	if cropAdjCSV != "" { _ = r.loadAdjCSV(cropAdjCSV) }
	if irrigXLSX != "" { _ = r.loadIrrigationXLSX(irrigXLSX) }

	if len(r.stageCfg) == 0 { return nil, errors.New("no stage config loaded") }
	return r, nil
}

func (r *rules) loadStagesCSV(path string) error {
    f, err := os.Open(path)
    if err != nil { return err }
    defer f.Close()

    cr := csv.NewReader(f)
    head, err := cr.Read()
    if err != nil { return err }

    // Build normalized header map
    norm := func(s string) string {
        s = strings.TrimSpace(s)
        s = strings.TrimPrefix(s, "\uFEFF") // BOM
        s = strings.ToLower(s)
        s = strings.ReplaceAll(s, " ", "")
        s = strings.ReplaceAll(s, "-", "")
        s = strings.ReplaceAll(s, "_", "")
        return s
    }
    hmap := map[string]int{}
    for i, h := range head {
        hmap[norm(h)] = i
    }

    // Accept multiple aliases
    findAny := func(keys ...string) int {
        for _, k := range keys {
            if idx, ok := hmap[norm(k)]; ok { return idx }
        }
        return -1
    }

    cStage := findAny("Stage", "stage", "phase")
    cDays  := findAny("Days", "duration", "days_in_stage", "stagedays")
    cWmm   := findAny("WaterNeed_mm_per_day", "water_mm_day", "waterperdaymm", "waterneed", "watermmday")
    cInt   := findAny("IrrigationInterval", "interval", "irrigation_interval", "wateringintervaldays")
    cNote  := findAny("Notes", "note", "remark", "tips")

    if cStage == -1 || cDays == -1 || cWmm == -1 {
        return fmt.Errorf("StageConfig.csv missing required columns. Found headers: %v\nNeed at least: Stage, Days, WaterNeed_mm_per_day", head)
    }
    // interval/notes can be optional; we’ll default interval if missing.
    defaultInterval := 3

    for {
        rec, err := cr.Read()
        if err != nil {
            if errors.Is(err, io.EOF) { break }
            return err
        }
        // guard against short rows
        get := func(idx int) string {
            if idx < 0 || idx >= len(rec) { return "" }
            return rec[idx]
        }

        days, _ := strconv.Atoi(strings.TrimSpace(get(cDays)))
        if days <= 0 { continue } // skip invalid rows

        wmm, _ := strconv.ParseFloat(strings.TrimSpace(get(cWmm)), 64)
        intval := defaultInterval
        if cInt != -1 {
            if v, err := strconv.Atoi(strings.TrimSpace(get(cInt))); err == nil && v > 0 {
                intval = v
            }
        }

        r.stageCfg = append(r.stageCfg, stageRow{
            Name:        strings.TrimSpace(get(cStage)),
            Days:        days,
            WaterMMDay:  wmm,
            IntervalDays:intval,
            Notes:       strings.TrimSpace(get(cNote)),
        })
    }
    return nil
}


func (r *rules) loadAdjCSV(path string) error {
	f, err := os.Open(path); if err != nil { return err }
	defer f.Close()
	cr := csv.NewReader(f)
	_, _ = cr.Read()
	for {
		rec, err := cr.Read(); if err != nil { break }
		fac, _ := strconv.ParseFloat(rec[1], 64)
		r.adj[rec[0]] = fac
	}
	return nil
}

func (r *rules) loadIrrigationXLSX(path string) error {
	x, err := excelize.OpenFile(path); if err != nil { return err }
	defer x.Close()
	// Optional: read a sheet for soil interval overrides, Fert tips, etc.
	return nil
}

func (r *rules) BuildStages(f *entities.Field) []types.StagePlan {
	start := f.PlantingDate
	adj := r.adj[f.CropType]
	if adj == 0 { adj = 1.0 }
	var stages []types.StagePlan
	cur := start
	for _, row := range r.stageCfg {
		dDur := int(float64(row.Days) * adj)
		end := cur.AddDate(0,0,dDur)
		stages = append(stages, types.StagePlan{
			Stage: row.Name,
			StartDate: cur.Format("2006-01-02"),
			EndDate: end.Format("2006-01-02"),
			WaterMMDay: row.WaterMMDay,
			Notes: row.Notes,
		})
		cur = end
	}
	return stages
}

func (r *rules) ExpandDaily(f *entities.Field, stages []types.StagePlan) []types.PlanOp {
	soilInterval := map[string]int{"sand":2, "loam":3, "clay":4}
	if v, ok := r.soilIrr[f.SoilTexture]; ok { soilInterval[f.SoilTexture] = v }
	var ops []types.PlanOp
	// Iterate each day
	for _, st := range stages {
		sd, _ := time.Parse("2006-01-02", st.StartDate)
		ed, _ := time.Parse("2006-01-02", st.EndDate)
		interval := soilInterval[f.SoilTexture]
		if interval <= 0 { interval = 3 }
		for d := sd; !d.After(ed.AddDate(0,0,-1)); d = d.AddDate(0,0,1) {
			dayStr := d.Format("2006-01-02")
			// Observation every 2 days
			if d.Sub(sd).Hours()/24.0 == 0 || int(d.Sub(sd).Hours()/24.0)%2 == 0 {
				ops = append(ops, types.PlanOp{Date: dayStr, Type:"observe", Title:"วัดความสูงและความชื้น", Notes:"บันทึกค่าให้ระบบปรับแผน"})
			}
			// Irrigation per interval
			daysFromStageStart := int(d.Sub(sd).Hours()/24.0)
			if daysFromStageStart%interval == 0 {
				mm := st.WaterMMDay * float64(interval)
				volPerRaiM3 := mm * 0.001 * 1600.0 // mm -> m * m^2 (1 rai ≈ 1600 m2)
				qty := volPerRaiM3 * f.AreaRai
				ops = append(ops, types.PlanOp{Date: dayStr, Type:"irrigation", Title:"รดน้ำตามรอบ", Qty:&qty, Unit:"m3", Notes:fmt.Sprintf("%.1f mm ต่อ %d วัน", mm, interval)})
			}
			// Fertilizer marker at stage boundaries
			if d.Equal(sd) && (st.Stage=="Tillering" || st.Stage=="Elongation") {
				qty := 30.0 * f.AreaRai // simple placeholder kg/rai
				ops = append(ops, types.PlanOp{Date: dayStr, Type:"fertilizer", Title:"ใส่ปุ๋ย 15-15-15", Qty:&qty, Unit:"kg", Notes:"ตัวอย่าง: ปรับในภายหลังตามงบประมาณ"})
			}
		}
	}
	// Sort by date
	sort.SliceStable(ops, func(i, j int) bool { return ops[i].Date < ops[j].Date })
	return ops
}

func (r *rules) ToSchedule(f *entities.Field, planID uint, ops []types.PlanOp) []entities.ScheduleTask {
	var out []entities.ScheduleTask
	for _, op := range ops {
		d, _ := time.Parse("2006-01-02", op.Date)
		out = append(out, entities.ScheduleTask{
			FieldID: f.FieldID, PlanID: planID, Date: d, Title: op.Title, Type: op.Type, Qty: op.Qty, Unit: op.Unit, Notes: op.Notes, Status: "todo",
		})
	}
	return out
}

func (r *rules) EvaluateDrift(f *entities.Field, recent []entities.Measurement, stages []types.StagePlan) (bool, string) {
	// Very simple rule: if last height < expected*0.85 OR 3 consecutive moist_state==dry
	if len(recent) == 0 { return false, "" }
	last := recent[len(recent)-1]
	// expected height: naive 1.2 cm/day since planting (placeholder)
	days := int(time.Since(f.PlantingDate).Hours()/24.0)
	expected := 1.2 * float64(days)
	if last.CaneHeightCM != nil && *last.CaneHeightCM < 0.85*expected {
		return true, fmt.Sprintf("height drift: got %.1f vs %.1f", *last.CaneHeightCM, expected)
	}
	// moisture state
	cntDry := 0
	for i := len(recent)-1; i>=0 && i>=len(recent)-5; i-- {
		if recent[i].MoistState == "dry" { cntDry++ } else { break }
	}
	if cntDry >= 3 { return true, "soil moisture low 3+ days" }
	return false, ""
}