package storage

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/disk"
	"github.com/user/sysinfogo/internal/output"
)

func collect(ctx context.Context) (*Info, []output.Warning, error) {
	var warns []output.Warning

	parts, err := disk.PartitionsWithContext(ctx, false)
	if err != nil {
		return nil, warns, fmt.Errorf("disk partitions: %w", err)
	}

	physDisks := queryMacPhysDisks(ctx)

	var disks []DiskInfo
	for _, pd := range physDisks {
		d := pd
		d.Health = "Unknown"
		d.HealthPct = 0

		for _, p := range parts {
			if !strings.HasPrefix(p.Device, "/dev/") {
				continue
			}
			usage, err := disk.UsageWithContext(ctx, p.Mountpoint)
			if err != nil {
				continue
			}
			part := PartitionInfo{
				Name:       p.Device,
				MountPoint: p.Mountpoint,
				FSType:     p.Fstype,
				TotalGB:    float64(usage.Total) / (1024 * 1024 * 1024),
				FreeGB:     float64(usage.Free) / (1024 * 1024 * 1024),
				UsedPct:    usage.UsedPercent,
			}
			d.Partitions = append(d.Partitions, part)
		}
		if d.SizeGB == 0 && len(d.Partitions) > 0 {
			d.SizeGB = d.Partitions[0].TotalGB
		}

		if len(d.Partitions) > 0 {
			collectSmartDarwin(ctx, d.Partitions[0].Name, &d)
		}
		disks = append(disks, d)
	}

	if len(disks) == 0 {
		for _, p := range parts {
			if !strings.HasPrefix(p.Device, "/dev/") {
				continue
			}
			usage, err := disk.UsageWithContext(ctx, p.Mountpoint)
			if err != nil {
				continue
			}
			d := DiskInfo{
				Model:  p.Device,
				Health: "Unknown",
				HealthPct: 0,
				SizeGB: float64(usage.Total) / (1024 * 1024 * 1024),
				Partitions: []PartitionInfo{{
					Name:       p.Device,
					MountPoint: p.Mountpoint,
					FSType:     p.Fstype,
					TotalGB:    float64(usage.Total) / (1024 * 1024 * 1024),
					FreeGB:     float64(usage.Free) / (1024 * 1024 * 1024),
					UsedPct:    usage.UsedPercent,
				}},
			}
			disks = append(disks, d)
		}
	}

	return &Info{Disks: disks}, warns, nil
}

func queryMacPhysDisks(ctx context.Context) []DiskInfo {
	smartCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(smartCtx, "system_profiler", "SPStorageDataType")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var disks []DiskInfo
	var cur *DiskInfo
	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if !strings.HasPrefix(trimmed, " ") {
			if cur != nil {
				disks = append(disks, *cur)
			}
			cur = &DiskInfo{Model: trimString(trimmed, ":")}
			continue
		}

		if cur == nil {
			continue
		}

		if strings.Contains(trimmed, "Size:") {
			var sizeGB float64
			fmt.Sscanf(trimmed, "Size: %f GB", &sizeGB)
			cur.SizeGB = sizeGB
		}
		if strings.Contains(trimmed, "Medium Type:") {
			cur.MediaType = trimString(trimmed, "Medium Type:")
			if strings.Contains(strings.ToUpper(cur.MediaType), "SOLID") {
				cur.MediaType = "SSD"
			} else {
				cur.MediaType = "HDD"
			}
		}
		if strings.Contains(trimmed, "Device Name:") {
			cur.Serial = trimString(trimmed, "Device Name:")
		}
		if strings.Contains(trimmed, "S.M.A.R.T. Status:") {
			smart := trimString(trimmed, "S.M.A.R.T. Status:")
			if strings.Contains(strings.ToUpper(smart), "VERIFIED") || strings.Contains(strings.ToUpper(smart), "OK") {
				cur.Health = "OK"
			} else {
				cur.Health = "WARNING"
			}
		}
	}

	if cur != nil {
		disks = append(disks, *cur)
	}

	return disks
}

func trimString(s, prefix string) string {
	idx := strings.Index(s, prefix)
	if idx < 0 {
		return s
	}
	return strings.TrimSpace(s[idx+len(prefix):])
}

func collectSmartDarwin(ctx context.Context, device string, d *DiskInfo) []output.Warning {
	smartCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	out, _, err := ExecSmartctl(smartCtx, "-A", device)
	if err != nil {
		return nil
	}

	parseSmartOutput(string(out), d)
	return nil
}
