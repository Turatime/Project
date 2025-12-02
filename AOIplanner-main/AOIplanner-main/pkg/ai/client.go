// pkg/ai/client.go

package ai

import (
	"aoi/entities"
	"aoi/pkg/plan/types"
)

type Client interface {
	SummarizePlan(f *entities.Field, stages []types.StagePlan, ops []types.PlanOp, kbCtx string) string

	// NEW: ask the model to propose structured additional actions based on problems + KB context
	ProposeOps(f *entities.Field, stages []types.StagePlan, ops []types.PlanOp, problems []string, kbCtx string) ([]types.PlanOp, error)
}
