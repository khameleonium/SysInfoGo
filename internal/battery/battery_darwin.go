package battery

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/user/sysinfogo/internal/output"
)

func collect(ctx context.Context) (*Info, []output.Warning, error) {
	cmd := exec.CommandContext(ctx, "pmset", "-g", "batt")
	out, err := cmd.Output()
	if err != nil {
		return &Info{Present: false}, nil, nil
	}

	info := &Info{Present: true, Status: "неизвестно"}
	raw := string(out)
	lines := strings.Split(raw, "\n")

	for _, line := range lines {
		if strings.Contains(line, "InternalBattery") || strings.Contains(line, "Battery") {
			if strings.Contains(line, "discharging") {
				info.Status = "разряжается"
			} else if strings.Contains(line, "charging") {
				info.Status = "заряжается"
			} else if strings.Contains(line, "charged") {
				info.Status = "заряжена"
			} else if strings.Contains(line, "AC attached") || strings.Contains(line, "AC Power") {
				info.Status = "подключена к сети"
			}

			parts := strings.Split(line, "\t")
			for _, p := range parts {
				if strings.Contains(p, "%") {
					var pct float64
					fmt.Sscanf(p, "%f%%", &pct)
					info.ChargePct = pct
				}
				if strings.Contains(p, ":") && (strings.Contains(p, "remaining") || strings.Contains(p, "до")) {
					info.TimeRemain = strings.TrimSpace(p)
				}
			}
			break
		}
	}

	cmdSP := exec.CommandContext(ctx, "system_profiler", "SPPowerDataType")
	if outSP, err := cmdSP.Output(); err == nil {
		linesSP := strings.Split(string(outSP), "\n")
		for _, line := range linesSP {
			if strings.Contains(line, "Cycle Count") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &info.CycleCount)
				}
			} else if strings.Contains(line, "Maximum Capacity") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					var pct float64
					if _, err := fmt.Sscanf(strings.TrimSpace(parts[1]), "%f%%", &pct); err == nil {
						info.HealthPct = pct
					}
				}
			}
		}
	}

	return info, nil, nil
}
