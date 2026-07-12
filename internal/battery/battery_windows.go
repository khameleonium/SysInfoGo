package battery

import (
	"context"
	"fmt"
	"strconv"

	"github.com/user/sysinfogo/internal/output"
	"github.com/user/sysinfogo/internal/wmi"
)

func collect(ctx context.Context) (*Info, []output.Warning, error) {
	raw := wmi.QueryList(ctx, "path", "Win32_Battery", "get", "BatteryStatus,EstimatedChargeRemaining,EstimatedRunTime")
	if raw == "" {
		return &Info{Present: false}, nil, nil
	}

	records := wmi.ParseList(raw)
	if len(records) == 0 {
		return &Info{Present: false}, nil, nil
	}

	record := records[0]
	info := &Info{Present: true}

	status, _ := strconv.Atoi(record["BatteryStatus"])
	switch status {
	case 1:
		info.Status = "разряжается"
	case 2:
		info.Status = "подключена к сети"
	case 3:
		info.Status = "заряжается"
	default:
		info.Status = "неизвестно"
	}

	if val, ok := record["EstimatedChargeRemaining"]; ok && val != "" {
		if pct, err := strconv.ParseFloat(val, 64); err == nil {
			info.ChargePct = pct
		}
	}
	if val, ok := record["EstimatedRunTime"]; ok && val != "" {
		if mins, err := strconv.Atoi(val); err == nil && mins > 0 {
			info.TimeRemain = fmt.Sprintf("%dм", mins)
		}
	}

	return info, nil, nil
}
