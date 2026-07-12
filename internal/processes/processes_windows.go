package processes

import (
	"context"
	"fmt"
	"sort"

	"github.com/shirou/gopsutil/v4/process"
	"github.com/user/sysinfogo/internal/output"
)

// CPUThreshold is the threshold for high CPU usage
const CPUThreshold = 80.0

func collect(ctx context.Context) (*Info, []output.Warning, error) {
	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("processes: %w", err)
	}

	type procData struct {
		info ProcessInfo
		cpu  float64
		mem  float32
	}

	var data []procData
	for _, p := range procs {
		name, _ := p.NameWithContext(ctx)
		cpuPct, _ := p.CPUPercentWithContext(ctx)
		memPct, _ := p.MemoryPercentWithContext(ctx)
		memInfo, _ := p.MemoryInfoWithContext(ctx)
		user, _ := p.UsernameWithContext(ctx)

		pd := procData{
			info: ProcessInfo{
				Name: name,
				PID:  p.Pid,
				User: user,
			},
			cpu: cpuPct,
			mem: memPct,
		}
		if memInfo != nil {
			pd.info.RSSMB = memInfo.RSS / (1024 * 1024)
		}
		data = append(data, pd)
	}

	sort.Slice(data, func(i, j int) bool {
		return data[i].cpu > data[j].cpu
	})
	info := &Info{TotalCount: len(procs)}
	for i := 0; i < len(data); i++ {
		data[i].info.CPU = data[i].cpu
		data[i].info.Memory = float64(data[i].mem)
		info.Processes = append(info.Processes, data[i].info)
	}

	return info, nil, nil
}
