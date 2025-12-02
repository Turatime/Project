// pkg/ai/openai_client.go

package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"aoi/entities"
	"aoi/pkg/plan/types"
)

type openAI struct {
	endpoint string
	key      string
	model    string
}

func NewOpenAI(endpoint, key, model string) Client {
	return &openAI{endpoint: endpoint, key: key, model: model}
}

func (c *openAI) SummarizePlan(f *entities.Field, stages []types.StagePlan, ops []types.PlanOp, kbCtx string) string {
	type chatReq struct {
		Model     string                 `json:"model"`
		Messages  []map[string]string    `json:"messages"`
		Temperature float64              `json:"temperature"`
	}
	reqBody := chatReq{
		Model: c.model,
		Messages: []map[string]string{
			{"role": "system", "content": "You are a Thai sugarcane agronomist who writes concise, actionable summaries in Markdown."},
			{"role": "user", "content": renderSummaryPrompt(f, stages, ops, kbCtx)},
		},
		Temperature: 0.2,
	}

	b, _ := json.Marshal(reqBody)
	httpc := &http.Client{Timeout: 25 * time.Second}
	req, _ := http.NewRequest("POST", strings.TrimRight(c.endpoint, "/")+"/v1/chat/completions", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpc.Do(req)
	if err != nil {
		// fallback summary (no external call)
		return fallbackSummary(f, stages)
	}
	defer resp.Body.Close()

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || len(out.Choices) == 0 {
		return fallbackSummary(f, stages)
	}
	content := strings.TrimSpace(out.Choices[0].Message.Content)
	if content == "" {
		return fallbackSummary(f, stages)
	}
	return content
}

// NEW
func (c *openAI) ProposeOps(f *entities.Field, stages []types.StagePlan, ops []types.PlanOp, problems []string, kbCtx string) ([]types.PlanOp, error) {
	type llmOp struct {
		Type  string   `json:"type"`            // irrigation | fertilizer | pesticide | inspect | advisory | other
		Title string   `json:"title"`
		Qty   *float64 `json:"qty,omitempty"`   // numeric amount (if any)
		Unit  string   `json:"unit,omitempty"`  // mm / kg/rai / L / etc
		Notes string   `json:"notes,omitempty"`
	}
	reqBody := map[string]any{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a Thai sugarcane agronomist. Reply ONLY valid JSON."},
			{"role": "user", "content": renderProposeOpsPrompt(f, stages, ops, problems, kbCtx)},
		},
		"temperature": 0.2,
	}
	b, _ := json.Marshal(reqBody)
	httpc := &http.Client{Timeout: 25 * time.Second}
	req, _ := http.NewRequest("POST", strings.TrimRight(c.endpoint, "/")+"/v1/chat/completions", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Choices) == 0 {
		return nil, fmt.Errorf("no choices")
	}

	// Parse JSON payload
	var payload struct{ Actions []llmOp `json:"actions"` }
	if err := json.Unmarshal([]byte(out.Choices[0].Message.Content), &payload); err != nil {
		var arr []llmOp
		if err2 := json.Unmarshal([]byte(out.Choices[0].Message.Content), &arr); err2 != nil {
			return nil, fmt.Errorf("parse propose_ops: %v / raw: %s", err, out.Choices[0].Message.Content)
		}
		payload.Actions = arr
	}

	// Map to PlanOp
	res := make([]types.PlanOp, 0, len(payload.Actions))
	for _, a := range payload.Actions {
		tp := strings.ToLower(strings.TrimSpace(a.Type))
		if tp == "" { tp = "advisory" }
		res = append(res, types.PlanOp{
			Type:  tp,
			Title: strings.TrimSpace(a.Title),
			Qty:   a.Qty,
			Unit:  strings.TrimSpace(a.Unit),
			Notes: strings.TrimSpace(a.Notes),
		})
	}
	// Ensure at least one inspect
	hasInspect := false
	for _, r := range res { if r.Type == "inspect" { hasInspect = true; break } }
	if !hasInspect {
		res = append(res, types.PlanOp{Type: "inspect", Title: "สำรวจโรคตามฤดูกาล", Notes: "ตรวจอาการเสี่ยงในช่วงนี้"})
	}
	return res, nil
}

func renderProposeOpsPrompt(f *entities.Field, stages []types.StagePlan, ops []types.PlanOp, problems []string, kbCtx string) string {
	return fmt.Sprintf(`
จงทำหน้าที่นักวิชาการเกษตรอ้อย ช่วย "เสนอรายการปฏิบัติ" เพิ่มเติมจาก CURRENT PLAN เพื่อรับมือ PROBLEMS โดยใช้ KB NOTES ประกอบ
ข้อกำหนด:
- อนุญาตให้เสนอการกระทำที่นอกเหนือจากชุดเดิม (เช่น ระบายน้ำ, สำรวจ, สุขอนามัยแปลง)
- ถ้ามีความเสี่ยงโรค ให้อย่างน้อย 1 task แบบ inspect
- ให้ระบุปริมาณ/หน่วยถ้าเหมาะสม (mm, kg/rai)
- ตอบเป็น JSON เท่านั้น: {"actions":[{"type":"irrigation|fertilizer|pesticide|inspect|advisory|other","title":"...","qty":10,"unit":"mm","notes":"..."}, ...]}

FIELD: %+v

PROBLEMS: %v

CURRENT PLAN (stages+ops): %v

KB NOTES:
%s
`, f, problems, ops, kbCtx)
}

func renderSummaryPrompt(f *entities.Field, stages []types.StagePlan, ops []types.PlanOp, kbCtx string) string {
	return fmt.Sprintf(`
สรุป “แผนจัดการอ้อย” ภาษาไทยแบบกระชับ เป็นหัวข้อย่อย Markdown (ไม่เกิน 8 บรรทัด) และชัดเจนเชิงปฏิบัติ
- หากมี KB NOTES ให้ผูกบริบท/เหตุผล แต่ห้ามคัดลอกยาว
- ระบุสิ่งที่ต้องทำ, ปริมาณ/หน่วย (mm, kg/rai) เท่าที่เหมาะสม
- หลีกเลี่ยงภาษาทั่วไป เช่น "ควรใส่ใจ" ให้ใช้ประโยคปฏิบัติได้จริง

FIELD:
%+v

STAGES (ย่อ):
%v

OPS (ย่อ):
%v

KB NOTES (ย่อ/คัดใจความ):
%s
`, f, stages, ops, kbCtx)
}

func fallbackSummary(f *entities.Field, stages []types.StagePlan) string {
	return fmt.Sprintf(
		"**สรุปแผนเบื้องต้น**\n\n- แปลง: #%d, พื้นที่ %.2f ไร่\n- ระยะ: %d ช่วงการเจริญเติบโต\n- ดำเนินการตามปฏิทินงานที่ระบบจัดไว้ (รดน้ำ/ให้ปุ๋ย/สำรวจศัตรูพืช) และปรับตามสภาพอากาศจริง",
		f.FieldID, f.AreaRai, len(stages),
	)
}
