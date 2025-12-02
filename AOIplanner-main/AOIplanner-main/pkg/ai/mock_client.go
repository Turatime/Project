// pkg/ai/mock_client.go

package ai

import (
	"strings"

	"aoi/entities"
	"aoi/pkg/plan/types"
)

type mockClient struct{}

func NewMock() Client { return &mockClient{} }

func (m *mockClient) SummarizePlan(f *entities.Field, stages []types.StagePlan, ops []types.PlanOp, kbCtx string) string {
	// ... your existing mock summary ...
	return "สรุปแผนเบื้องต้น (mock)"
}

// NEW
func (m *mockClient) ProposeOps(f *entities.Field, stages []types.StagePlan, ops []types.PlanOp, problems []string, kbCtx string) ([]types.PlanOp, error) {
	out := make([]types.PlanOp, 0, 6)
	joined := strings.Join(problems, " ")

	if strings.Contains(joined, "พายุ") {
		out = append(out, types.PlanOp{Type: "advisory", Title: "เตรียมระบายน้ำ/ขุดร่อง", Notes: "กันน้ำขังเกิน 48 ชม."})
	}
	if strings.Contains(joined, "แห้ง") {
		q := 20.0
		out = append(out, types.PlanOp{Type: "irrigation", Title: "เพิ่มน้ำชดเชยความชื้น", Qty: &q, Unit: "mm", Notes: "ตามศักยภาพปั๊ม"})
	}
	if strings.Contains(joined, "ใบขาว") || strings.Contains(joined, "กอตะไคร้") || strings.Contains(joined, "แส้ดำ") {
		out = append(out, types.PlanOp{Type: "inspect", Title: "สำรวจอาการโรคอ้อย", Notes: "สุ่มตรวจ 5 จุด/แปลง พร้อมภาพประกอบ"})
	}
	// always add seasonal scouting
	out = append(out, types.PlanOp{Type: "inspect", Title: "สำรวจโรคตามฤดูกาล", Notes: "ดูใบจุดวงแหวน/เน่าแดงช่วงชื้น"})
	return out, nil
}

