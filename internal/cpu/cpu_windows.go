package cpu

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"

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

	rawProc := wmi.QueryList(ctx, "path", "Win32_Processor", "get", "L2CacheSize,L3CacheSize,MaxClockSpeed")
	if rawProc != "" {
		recs := wmi.ParseList(rawProc)
		if len(recs) > 0 {
			if l2, err := strconv.Atoi(recs[0]["L2CacheSize"]); err == nil && l2 > 0 {
				info.CacheL2KB = l2
			}
			if l3, err := strconv.Atoi(recs[0]["L3CacheSize"]); err == nil && l3 > 0 {
				info.CacheL3KB = l3
			}
			if maxMhz, err := strconv.Atoi(recs[0]["MaxClockSpeed"]); err == nil && maxMhz > 0 {
				info.MaxSpeedGHz = float64(maxMhz) / 1000.0
			}
		}
	}

	collectTempWindows(ctx, info)
	collectFanWindows(ctx, info)

	return info, warns, nil
}

func collectFanWindows(ctx context.Context, info *Info) {
	for _, ns := range []string{"root\\OpenHardwareMonitor", "root\\LibreHardwareMonitor"} {
		raw := wmi.QueryList(ctx, "path", ns+":Sensor", "get", "Name,Value,SensorType")
		if raw != "" {
			records := wmi.ParseList(raw)
			for _, rec := range records {
				if rec["SensorType"] == "Fan" {
					if val, err := strconv.ParseFloat(rec["Value"], 64); err == nil && val > 0 {
						info.FanSpeedRPM = int(val)
						return
					}
				}
			}
		}
	}
	rawFan := wmi.QueryList(ctx, "path", "Win32_Fan", "get", "DesiredSpeed")
	if rawFan != "" {
		records := wmi.ParseList(rawFan)
		for _, rec := range records {
			if spd, err := strconv.Atoi(rec["DesiredSpeed"]); err == nil && spd > 0 {
				info.FanSpeedRPM = spd
				return
			}
		}
	}
}

func collectTempWindows(ctx context.Context, info *Info) {
	// 1. Try OpenHardwareMonitor / LibreHardwareMonitor WMI if running
	for _, ns := range []string{"root\\OpenHardwareMonitor", "root\\LibreHardwareMonitor"} {
		raw := wmi.QueryList(ctx, "path", ns+":Sensor", "get", "Name,Value,SensorType")
		if raw != "" {
			records := wmi.ParseList(raw)
			for _, rec := range records {
				if rec["SensorType"] == "Temperature" && (strings.Contains(strings.ToLower(rec["Name"]), "cpu package") || strings.Contains(strings.ToLower(rec["Name"]), "cpu core")) {
					if val, err := strconv.ParseFloat(rec["Value"], 64); err == nil && val > 0 && val < 150 {
						info.PackageTemp = val
						return
					}
				}
			}
		}
	}

	// 2. Try MSAcpi_ThermalZoneTemperature
	raw := wmi.QueryList(ctx, "path", "MSAcpi_ThermalZoneTemperature", "get", "CurrentTemperature")
	if raw != "" {
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
}
