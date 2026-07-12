package memory

import (
	"context"

	"github.com/user/sysinfogo/internal/output"
)

type SlotInfo struct {
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	Serial       string `json:"serial,omitempty"`
	SizeGB       int    `json:"size_gb"`
	SpeedMTs     int    `json:"speed_mts"`
	Type         string `json:"type"`
	Locator      string `json:"locator"`
}

type Info struct {
	TotalGB        float64    `json:"total_gb"`
	UsedGB         float64    `json:"used_gb"`
	FreeGB         float64    `json:"free_gb"`
	BuffersCacheGB float64    `json:"buffers_cache_gb"`
	SwapTotalGB    float64    `json:"swap_total_gb"`
	SwapUsedGB     float64    `json:"swap_used_gb"`
	UsagePercent   float64    `json:"usage_percent"`
	Type           string     `json:"type"`
	SpeedMTs       int        `json:"speed_mts"`
	Timings        string     `json:"timings"`
	TotalSlots     int        `json:"total_slots"`
	UsedSlots      int        `json:"used_slots"`
	Slots          []SlotInfo `json:"slots,omitempty"`
}

func Collect(ctx context.Context) (*Info, []output.Warning, error) {
	return collect(ctx)
}
