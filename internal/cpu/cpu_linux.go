package cpu

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/sensors"
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

	collectTempLinux(ctx, info)

	return info, nil, nil
}

func collectTempLinux(ctx context.Context, info *Info) {
	sensorsList, err := sensors.TemperaturesWithContext(ctx)
	if err == nil {
		for _, sensor := range sensorsList {
			name := strings.ToLower(sensor.SensorKey)
			if strings.Contains(name, "cpu") || strings.Contains(name, "core") || strings.Contains(name, "package") || strings.Contains(name, "tctl") || strings.Contains(name, "k10temp") {
				if strings.Contains(name, "package") || strings.Contains(name, "tctl") || info.PackageTemp == 0 {
					info.PackageTemp = sensor.Temperature
				}
			}
		}
		if info.PackageTemp > 0 {
			return
		}
	}

	data, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp")
	if err == nil {
		if tempMillicelsius, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			info.PackageTemp = float64(tempMillicelsius) / 1000.0
		}
	}
}
