package processes

import (
	"context"
	"fmt"
	"sort"
	"sync"

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

	data := make([]procData, len(procs))
	sem := make(chan struct{}, 16)
	var wg sync.WaitGroup

	for i, p := range procs {
		wg.Add(1)
		go func(idx int, pr *process.Process) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			name, _ := pr.NameWithContext(ctx)
			cpuPct, _ := pr.CPUPercentWithContext(ctx)
			memPct, _ := pr.MemoryPercentWithContext(ctx)
			memInfo, _ := pr.MemoryInfoWithContext(ctx)
			user, _ := pr.UsernameWithContext(ctx)

			pd := procData{
				info: ProcessInfo{
					Name: name,
					PID:  pr.Pid,
					User: user,
				},
				cpu: cpuPct,
				mem: memPct,
			}
			if memInfo != nil {
				pd.info.RSSMB = memInfo.RSS / (1024 * 1024)
			}
			data[idx] = pd
		}(i, p)
	}
	wg.Wait()

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
