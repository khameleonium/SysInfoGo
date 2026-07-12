package cpu

import (
	"context"
	"fmt"
	"runtime"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/user/sysinfogo/internal/output"
)

func collect(ctx context.Context) (*Info, []output.Warning, error) {
	cpuInfos, err := cpu.InfoWithContext(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("cpu info: %w", err)
	}
	if len(cpuInfos) == 0 {
		return nil, nil, fmt.Errorf("no cpu info")
	}
	c0 := cpuInfos[0]

	physical, _ := cpu.CountsWithContext(ctx, false)
	logical, _ := cpu.CountsWithContext(ctx, true)

	usageTotalSlice, _ := cpu.PercentWithContext(ctx, 0, false)
	usagePerCore, _ := cpu.PercentWithContext(ctx, 0, true)

	var usageTotal float64
	if len(usageTotalSlice) > 0 {
		usageTotal = usageTotalSlice[0]
	}

	info := &Info{
		Model:           c0.ModelName,
		Vendor:          c0.VendorID,
		Architecture:    runtime.GOARCH,
		PhysicalCores:   physical,
		LogicalCores:    logical,
		CurrentSpeedGHz: c0.Mhz / 1000.0,
		BaseSpeedGHz:    c0.Mhz / 1000.0,
		MaxSpeedGHz:     c0.Mhz / 1000.0,
		UsagePercent:    usageTotal,
		UsagePerCore:    usagePerCore,
		CacheL1DataKB:   0,
		CacheL2KB:       int(c0.CacheSize),
		CacheL3KB:       0,
		InstructionSets: ParseFlags(c0.Flags),
	}

	return info, nil, nil
}
