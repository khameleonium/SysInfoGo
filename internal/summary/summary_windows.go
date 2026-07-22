package summary

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	cpuPkg "github.com/user/sysinfogo/internal/cpu"
	"github.com/user/sysinfogo/internal/gpu"
	"github.com/user/sysinfogo/internal/memory"
	"github.com/user/sysinfogo/internal/output"
)

func collect(ctx context.Context) (*Info, []output.Warning, error) {
	hostInfo, _ := host.InfoWithContext(ctx)
	hostname, _ := os.Hostname()
	uptime, _ := host.UptimeWithContext(ctx)
	bootTime, _ := host.BootTimeWithContext(ctx)
	memInfo, _ := mem.VirtualMemoryWithContext(ctx)
	cpuInfos, _ := cpu.InfoWithContext(ctx)
	parts, _ := disk.PartitionsWithContext(ctx, false)

	info := &Info{
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		Hostname:   hostname,
		Uptime:     formatUptime(uptime),
		BootTime:   time.Unix(int64(bootTime), 0).Format("2006-01-02 15:04:05"),
		RAMTotalGB: 0,
	}

	if hostInfo != nil {
		info.OS = hostInfo.Platform + " " + hostInfo.PlatformVersion
		info.Kernel = hostInfo.KernelVersion
		if hostInfo.VirtualizationSystem != "" {
			info.Virtualization = hostInfo.VirtualizationSystem
			if hostInfo.VirtualizationRole != "" {
				info.Virtualization += " (" + hostInfo.VirtualizationRole + ")"
			}
		}
	}

	if memInfo != nil {
		info.RAMTotalGB = float64(memInfo.Total) / (1024 * 1024 * 1024)
		info.RAMUsagePct = memInfo.UsedPercent
		info.RAMUsedGB = float64(memInfo.Used) / (1024 * 1024 * 1024)
	}

	if memDetails, _, _ := memory.Collect(ctx); memDetails != nil && memDetails.Spec != "" {
		info.RAMType = memDetails.Spec
	}

	if len(cpuInfos) > 0 {
		info.CPUModel = cpuInfos[0].ModelName
		physical, _ := cpu.CountsWithContext(ctx, false)
		logical, _ := cpu.CountsWithContext(ctx, true)
		info.CPUCores = physical
		info.CPULogical = logical
	}

	if cpuDetails, _, _ := cpuPkg.Collect(ctx); cpuDetails != nil && cpuDetails.PackageTemp > 0 {
		info.CPUTempC = cpuDetails.PackageTemp
	}

	if gpuInfo, _, _ := gpu.Collect(ctx); gpuInfo != nil && len(gpuInfo.GPUs) > 0 {
		info.GPUModel = gpuInfo.GPUs[0].Name
		for _, g := range gpuInfo.GPUs {
			info.GPUs = append(info.GPUs, GPUSummary{
				Name:       g.Name,
				VRAMMB:     g.VRAMMB,
				TempC:      g.TempC,
				GPULoadPct: g.GPULoadPct,
			})
		}
	}

	var storages []StorageSummary
	for _, p := range parts {
		usage, err := disk.UsageWithContext(ctx, p.Mountpoint)
		if err != nil {
			continue
		}
		storages = append(storages, StorageSummary{
			MountPoint: p.Mountpoint,
			TotalGB:    float64(usage.Total) / (1024 * 1024 * 1024),
			FreeGB:     float64(usage.Free) / (1024 * 1024 * 1024),
		})
	}
	info.Storages = storages

	return info, nil, nil
}

func formatUptime(seconds uint64) string {
	d := time.Duration(seconds) * time.Second
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	parts = append(parts, fmt.Sprintf("%dm", minutes))
	return strings.Join(parts, " ")
}
