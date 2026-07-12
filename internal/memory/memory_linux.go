package memory

import (
	"context"
	"fmt"

	"github.com/shirou/gopsutil/v4/mem"
	"github.com/user/sysinfogo/internal/output"
)

func collect(ctx context.Context) (*Info, []output.Warning, error) {
	v, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("virtual memory: %w", err)
	}
	s, _ := mem.SwapMemoryWithContext(ctx)

	info := &Info{
		TotalGB:        float64(v.Total) / (1024 * 1024 * 1024),
		UsedGB:         float64(v.Used) / (1024 * 1024 * 1024),
		FreeGB:         float64(v.Available) / (1024 * 1024 * 1024),
		BuffersCacheGB: float64(v.Cached+v.Buffers) / (1024 * 1024 * 1024),
		UsagePercent:   v.UsedPercent,
		Timings:        "N/A",
	}

	if s != nil {
		info.SwapTotalGB = float64(s.Total) / (1024 * 1024 * 1024)
		info.SwapUsedGB = float64(s.Used) / (1024 * 1024 * 1024)
	}

	return info, nil, nil
}
