package battery

import (
	"context"

	"github.com/user/sysinfogo/internal/output"
)

type Info struct {
	Present    bool    `json:"present"`
	Status     string  `json:"status,omitempty"`
	ChargePct  float64 `json:"charge_pct,omitempty"`
	TimeRemain string  `json:"time_remain,omitempty"`
	HealthPct  float64 `json:"health_pct,omitempty"`
	CycleCount int     `json:"cycle_count,omitempty"`
}

func Collect(ctx context.Context) (*Info, []output.Warning, error) {
	return collect(ctx)
}
