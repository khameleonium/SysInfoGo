package cpu

import (
	"context"
	"fmt"
	"runtime"
	"strconv"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/user/sysinfogo/internal/output"
	"github.com/user/sysinfogo/internal/wmi"
)

func collect(ctx context.Context) (*Info, []output.Warning, error) {
	var warns []output.Warning

	cpuInfos, err := cpu.InfoWithContext(ctx)
	if err != nil {
		return nil, warns, fmt.Errorf("cpu info: %w", err)
	}
	if len(cpuInfos) == 0 {
		return nil, warns, fmt.Errorf("no cpu information available")
	}
	c0 := cpuInfos[0]

	physical, err := cpu.CountsWithContext(ctx, false)
	if err != nil {
		physical = runtime.NumCPU() / 2
	}
	logical, err := cpu.CountsWithContext(ctx, true)
	if err != nil {
		logical = runtime.NumCPU()
	}

	usagePerCore, _ := cpu.PercentWithContext(ctx, 0, true)
	var usageTotal float64
	if len(usagePerCore) > 0 {
		for _, u := range usagePerCore {
			usageTotal += u
		}
		usageTotal /= float64(len(usagePerCore))
	}

	info := &Info{
		Model:           c0.ModelName,
		Vendor:          c0.VendorID,
		Architecture:    runtime.GOARCH,
		PhysicalCores:   physical,
		LogicalCores:    logical,
		CurrentSpeedGHz: c0.Mhz / 1000.0,
		UsagePercent:    usageTotal,
		UsagePerCore:    usagePerCore,
		CacheL1DataKB:   0,
		CacheL2KB:       int(c0.CacheSize),
		CacheL3KB:       0,
		InstructionSets: ParseFlags(c0.Flags),
	}

	if c0.Mhz > 0 {
		info.BaseSpeedGHz = c0.Mhz / 1000.0
	}
	info.MaxSpeedGHz = info.BaseSpeedGHz

	collectTempWindows(ctx, info)

	return info, warns, nil
}

func collectTempWindows(ctx context.Context, info *Info) {
	raw := wmi.QueryList(ctx, "path", "MSAcpi_ThermalZoneTemperature", "get", "CurrentTemperature")
	if raw == "" {
		return
	}
	records := wmi.ParseList(raw)
	for _, rec := range records {
		if tempStr, ok := rec["CurrentTemperature"]; ok && tempStr != "" {
			if tempK10, err := strconv.Atoi(tempStr); err == nil && tempK10 > 0 {
				celsius := (float64(tempK10) / 10.0) - 273.15
				if celsius > 0 && celsius < 150 {
					info.PackageTemp = celsius
					return
				}
			}
		}
	}
}
