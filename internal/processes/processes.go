package processes

import (
	"context"

	"github.com/user/sysinfogo/internal/output"
)

type ProcessInfo struct {
	Name   string  `json:"name"`
	PID    int32   `json:"pid"`
	CPU    float64 `json:"cpu_pct"`
	Memory float64 `json:"mem_pct"`
	RSSMB  uint64  `json:"rss_mb"`
	User   string  `json:"user"`
}

type Info struct {
	TotalCount int           `json:"total_count"`
	LoadAvg1   float64       `json:"load_avg_1,omitempty"`
	LoadAvg5   float64       `json:"load_avg_5,omitempty"`
	LoadAvg15  float64       `json:"load_avg_15,omitempty"`
	Processes  []ProcessInfo `json:"processes"`
}

func Collect(ctx context.Context) (*Info, []output.Warning, error) {
	return collect(ctx)
}
