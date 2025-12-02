package serviceImp

import (
	"encoding/json"
	"fmt"
	"time"

	"aoi/entities"
	"aoi/pkg/ai"
	"aoi/pkg/measure/repository"
	planrepo "aoi/pkg/plan/repository"
	schedrepo "aoi/pkg/schedule/repository"
	"aoi/pkg/climate"
	"aoi/pkg/plan/types"
	"strings"
)

type ReplanOptions struct {
	Reason   string
	Problems []string
}
type kbSearcher interface {
    Search(query string, k int) ([]entities.KBChunk, error)
    DocsMeta(ids []uint) (map[uint]entities.KBDocument, error)  // ← add
}
type PlanSvc struct{
	rules climate.RulesEngine
	llm   ai.Client
	repoPlan   planrepo.PlanRepository
	repoSched  schedrepo.ScheduleRepository
	repoMeas   repository.MeasureRepository
	kb        kbSearcher
}

var lastKBRefs []map[string]string

func setLastKBRefs(refs []map[string]string) { lastKBRefs = refs }
func LastKBRefs() []map[string]string { return lastKBRefs }

func NewPlanService(r climate.RulesEngine, llm ai.Client, pr planrepo.PlanRepository, sr schedrepo.ScheduleRepository, mr repository.MeasureRepository, kb kbSearcher) *PlanSvc {
	return &PlanSvc{rules:r, llm:llm, repoPlan:pr, repoSched:sr, repoMeas:mr, kb:kb}
}

func (s *PlanSvc) GenerateFirstPlan(field *entities.Field) (*entities.Plan, []entities.ScheduleTask, error) {
	stages := s.rules.BuildStages(field)
	ops := s.rules.ExpandDaily(field, stages)

	var kbCtx string

	if s.kb != nil {
    query := field.Variety + " sugarcane " +
        field.SoilTexture + " " + field.Province + " " + field.District +
        " irrigation fertilizer pest Thailand"
    snips, _ := s.kb.Search(query, 6)
	var kbRefs []map[string]string

    // build kbCtx
    for _, ch := range snips {
        if len(kbCtx) > 6000 { break }
        kbCtx += "\n---\n" + ch.Text
    }

    // collect unique docs → refs
    seen := map[uint]struct{}{}
    ids := make([]uint, 0, len(snips))
    for _, ch := range snips {
        if _, ok := seen[ch.DocID]; !ok {
            seen[ch.DocID] = struct{}{}
            ids = append(ids, ch.DocID)
        }
    }
	if len(ids) > 0 {
		if meta, err := s.kb.DocsMeta(ids); err == nil {
			for _, id := range ids {
				if d, ok := meta[id]; ok {
					kbRefs = append(kbRefs, map[string]string{"title": d.Title, "source_url": d.SourceURL})
				}
			}
		}
	}
	setLastKBRefs(kbRefs)
}

	summary := s.llm.SummarizePlan(field, stages, ops, kbCtx)
	stagesJSON, _ := json.Marshal(stages)
	p := &entities.Plan{FieldID: field.FieldID, Version: 1, SummaryMD: summary, StagesJSON: string(stagesJSON)}
	if err := s.repoPlan.Create(p); err != nil { return nil, nil, err }
	tasks := s.rules.ToSchedule(field, p.PlanID, ops)
	if err := s.repoSched.BulkInsert(tasks); err != nil { return nil, nil, err }
	return p, tasks, nil
}

func (s *PlanSvc) Replan(field *entities.Field) (*entities.Plan, []entities.ScheduleTask, *entities.ReplanLog, error) {
	// load latest plan
	old, err := s.repoPlan.LatestByField(field.FieldID)
	if err != nil { return nil, nil, nil, err }
	// recent measurements
	recent, _ := s.repoMeas.Recent(field.FieldID, 14)
	// parse old stages
	var oldStages []types.StagePlan
	_ = json.Unmarshal([]byte(old.StagesJSON), &oldStages)
	// evaluate drift
	need, reason := s.rules.EvaluateDrift(field, recent, oldStages)
	if !need {
		return old, nil, nil, nil
	}
	// Build new stages (simple: shift forward 3 days)
	field.PlantingDate = field.PlantingDate.AddDate(0,0,-3) // nudge earlier to increase expected height
	newStages := s.rules.BuildStages(field)
	ops := s.rules.ExpandDaily(field, newStages)

	var kbCtx string
	if s.kb != nil {
		// build a focused query from the field
		query := field.Variety + " sugarcane " +
			field.SoilTexture + " " + field.Province + " " + field.District +
			" irrigation fertilizer pest Thailand"
		snips, _ := s.kb.Search(query, 6) // ignore errors for robustness
		for _, ch := range snips {
			if len(kbCtx) > 6000 { break }
			kbCtx += "\n---\n" + ch.Text
		}
	}

	summary := s.llm.SummarizePlan(field, newStages, ops, kbCtx)
	stagesJSON, _ := json.Marshal(newStages)
	p := &entities.Plan{FieldID: field.FieldID, Version: old.Version+1, SummaryMD: summary, StagesJSON: string(stagesJSON)}
	if err := s.repoPlan.Create(p); err != nil { return nil, nil, nil, err }
	tasks := s.rules.ToSchedule(field, p.PlanID, ops)
	if err := s.repoSched.BulkInsert(tasks); err != nil { return nil, nil, nil, err }
	log := &entities.ReplanLog{
    FieldID: field.FieldID,
    PlanID:  p.PlanID, // store the new plan id here
    Reason:  reason,
    DeltaMD: fmt.Sprintf("replanned at %s due to %s", time.Now().Format(time.RFC3339), reason),
	}
	return p, tasks, log, nil
}

// ReplanWithOptions keeps your original flow but augments with problems → KB → LLM actions.
func (s *PlanSvc) ReplanWithOptions(f *entities.Field, opts ReplanOptions) (*entities.Plan, []entities.ScheduleTask, *entities.ReplanLog, error) {
	// 1) Run your current replan to get baseline plan + tasks + replan log
	p, tasks, rep, err := s.Replan(f) // <-- call your existing method
	if err != nil {
		return nil, nil, nil, err
	}

	// 2) Build KB terms from problems + stage/season (light heuristic)
	terms := []string{}
	if len(opts.Problems) > 0 {
		terms = append(terms, strings.Join(opts.Problems, " "))
	}
	// if you have helpers to get current stage/season, add them here

	// 3) Search KB and collect context (prefer mitrpholmodernfarm.com, then fallback)
	kbCtx := ""
	var kbRefs []entities.ArticleRef
	if len(terms) > 0 && s.kb != nil {
		chunks, _ := s.kb.Search(strings.Join(terms, " "), 12)
		if len(chunks) > 0 {
			ids := uniqueDocIDs(chunks)
			meta, _ := s.kb.DocsMeta(ids)

			var mitr, other []entities.ArticleRef
			var sb strings.Builder
			for _, ch := range chunks {
				m := meta[ch.DocID]
				if t := strings.TrimSpace(m.Title); t != "" {
					sb.WriteString(t)
					sb.WriteString("\n")
				}
				sb.WriteString(ch.Text)
				sb.WriteString("\n---\n")

				ref := entities.ArticleRef{Title: m.Title, URL: m.SourceURL}
				if strings.Contains(strings.ToLower(m.SourceURL), "mitrpholmodernfarm.com") {
					mitr = append(mitr, ref)
				} else {
					other = append(other, ref)
				}
			}
			kbCtx = sb.String()
			kbRefs = append(kbRefs, mitr...)
			kbRefs = append(kbRefs, other...)
		}
	}

	// 4) Ask LLM for structured extra ops (fallback to simple heuristics if LLM not configured)
	var extraOps []types.PlanOp
	if s.llm != nil {
		if ops, err := s.llm.ProposeOps(f, /* stages */ nil, /* ops */ nil, opts.Problems, kbCtx); err == nil {
			extraOps = ops
		}
	}
	if len(extraOps) == 0 {
		extraOps = s.deriveSuggestedOpsFallback(f, /* stages */ nil, opts.Problems, kbCtx)
	}

	// 5) Materialize extra ops to tasks (allow new kinds: "inspect", "advisory")
	extraTasks := s.materializeOpsToTasks(f, p.PlanID, extraOps)

	// Ensure at least one inspect when problems mention diseases/season risk
	if s.needsDiseaseScout(f, /* stages */ nil, opts.Problems) {
		extraTasks = append(extraTasks, entities.ScheduleTask{
			FieldID: f.FieldID,
			PlanID:  p.PlanID,
			Date:    time.Now().AddDate(0, 0, 3),
			Type:    "inspect",
			Title:   "สำรวจโรคตามฤดูกาล",
			Notes:   "ดูใบขาว/กอตะไคร้/แส้ดำ/ใบจุดวงแหวนตามฤดูกาล",
			Status:  "todo",
		})
	}

	// 6) Append tasks to baseline & persist with your existing save path
	tasks = append(tasks, extraTasks...)
	if len(extraTasks) > 0 {
		if err := s.repoSched.BulkInsert(extraTasks); err != nil {
			return nil, nil, nil, err
		}
	}

	// 7) Attach problems & suggested articles to the replan log (articles are transient)
	if rep != nil {
		rep.Problems = opts.Problems
		// prefer mitr articles first (already ordered)
		max := 5
		if len(kbRefs) < max {
			max = len(kbRefs)
		}
		rep.SuggestedArticles = kbRefs[:max]
	}

	return p, tasks, rep, nil
}

// deriveSuggestedOpsFallback provides deterministic suggestions if the LLM is unavailable.
func (s *PlanSvc) deriveSuggestedOpsFallback(f *entities.Field, stages []types.StagePlan, problems []string, kbCtx string) []types.PlanOp {
	joined := strings.Join(problems, " ")
	out := make([]types.PlanOp, 0, 6)

	if strings.Contains(joined, "พายุ") {
		out = append(out, types.PlanOp{
			Type:  "advisory",
			Title: "เตรียมระบายน้ำ/ขุดร่อง",
			Notes: "กันน้ำขังเกิน 48 ชม.",
		})
	}
	if strings.Contains(joined, "แห้ง") {
		q := 20.0
		out = append(out, types.PlanOp{
			Type:  "irrigation",
			Title: "เพิ่มน้ำชดเชยความชื้น",
			Qty:   &q,
			Unit:  "mm",
			Notes: "ตามศักยภาพปั๊ม",
		})
	}
	if strings.Contains(joined, "ใบขาว") || strings.Contains(joined, "กอตะไคร้") || strings.Contains(joined, "แส้ดำ") {
		out = append(out, types.PlanOp{
			Type:  "inspect",
			Title: "สำรวจอาการโรคอ้อย",
			Notes: "สุ่มตรวจ 5 จุด/แปลง พร้อมภาพประกอบ",
		})
	}
	// always add seasonal scouting
	out = append(out, types.PlanOp{
		Type:  "inspect",
		Title: "สำรวจโรคตามฤดูกาล",
		Notes: "ดูใบจุดวงแหวน/เน่าแดงช่วงชื้น",
	})
	return out
}

// materializeOpsToTasks maps suggested ops into schedule tasks for the given planID.
func (s *PlanSvc) materializeOpsToTasks(field *entities.Field, planID uint, ops []types.PlanOp) []entities.ScheduleTask {
	base := time.Now().Add(48 * time.Hour)
	tasks := make([]entities.ScheduleTask, 0, len(ops))
	for i, op := range ops {
		t := entities.ScheduleTask{
			FieldID: field.FieldID,
			PlanID:  planID,
			Date:    base.AddDate(0, 0, i),
			Title:   op.Title,
			Notes:   op.Notes,
			Status:  "todo",
		}
		switch strings.ToLower(op.Type) {
		case "irrigation":
			t.Type = "irrigation"
			t.Qty = op.Qty
			if op.Unit != "" { t.Unit = op.Unit } else { t.Unit = "mm" }
		case "fertilizer":
			t.Type = "fertilizer"
			t.Qty = op.Qty
			t.Unit = op.Unit
		case "pesticide":
			t.Type = "pesticide"
		case "inspect":
			t.Type = "inspect"
		default:
			t.Type = "advisory"
		}
		tasks = append(tasks, t)
	}
	return tasks
}

// needsDiseaseScout enforces at least one inspection when problems imply disease/season risk.
func (s *PlanSvc) needsDiseaseScout(f *entities.Field, stages []types.StagePlan, problems []string) bool {
    joined := strings.Join(problems, " ")
    if strings.Contains(joined, "ใบขาว") || strings.Contains(joined, "กอตะไคร้") ||
       strings.Contains(joined, "แส้ดำ") || strings.Contains(joined, "จุดวงแหวน") {
        return true
    }
    // you can extend this by season/stage if needed
    return false
}

func uniqueDocIDs(chs []entities.KBChunk) []uint {
	seen := map[uint]struct{}{}
	var ids []uint
	for _, ch := range chs {
		if _, ok := seen[ch.DocID]; !ok {
			seen[ch.DocID] = struct{}{}
			ids = append(ids, ch.DocID)
		}
	}
	return ids
}