package storage

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

	blkMap := make(map[string]*DiskInfo)

	for _, p := range parts {
		blockName := blockDeviceName(p.Device)
		if blockName == "" {
			continue
		}
		if !strings.HasPrefix(p.Device, "/dev/") {
			continue
		}

		if _, ok := blkMap[blockName]; !ok {
			d := readBlockDeviceInfo(blockName)
			blkMap[blockName] = d
		}

		d := blkMap[blockName]
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

	hasSmartctl := false
	if smartctl, _ := exec.LookPath("smartctl"); smartctl != "" {
		hasSmartctl = true
	}

	warnedSmartctl := false
	for _, d := range blkMap {
		if len(d.Partitions) > 0 {
			if hasSmartctl {
				smartWarns := collectSmartLinux(ctx, d.Partitions[0].Name, d)
				warns = append(warns, smartWarns...)
			} else if !warnedSmartctl {
				warns = append(warns, output.Warning{
					Section: "storage",
					Message: "Утилита smartctl не найдена.",
					OSHint:  "Пожалуйста, установите пакет smartmontools, чтобы видеть данные о здоровье дисков (SMART).",
				})
				warnedSmartctl = true
			}
		}
	}

	var disks []DiskInfo
	for _, d := range blkMap {
		disks = append(disks, *d)
	}

	return &Info{Disks: disks}, warns, nil
}

func blockDeviceName(device string) string {
	base := filepath.Base(device)
	name := strings.TrimRight(base, "0123456789")
	if len(name) < 3 {
		return ""
	}
	return name
}

func readBlockDeviceInfo(blockName string) *DiskInfo {
	sysPath := "/sys/block/" + blockName + "/device"
	d := &DiskInfo{
		DeviceName: "/dev/" + blockName,
		Interface:  detectLinuxInterface(blockName),
		Health:     "Unknown",
		HealthPct:  0,
	}

	if model, err := os.ReadFile(sysPath + "/model"); err == nil {
		d.Model = strings.TrimSpace(string(model))
	}
	if serial, err := os.ReadFile(sysPath + "/serial"); err == nil {
		d.Serial = strings.TrimSpace(string(serial))
	}
	if rev, err := os.ReadFile(sysPath + "/rev"); err == nil {
		d.FirmwareRevision = strings.TrimSpace(string(rev))
	}

	rotPath := "/sys/block/" + blockName + "/queue/rotational"
	if rot, err := os.ReadFile(rotPath); err == nil {
		if strings.TrimSpace(string(rot)) == "0" {
			d.MediaType = "SSD"
		} else {
			d.MediaType = "HDD"
		}
	}

	sectorSizePath := "/sys/block/" + blockName + "/queue/hw_sector_size"
	if data, err := os.ReadFile(sectorSizePath); err == nil {
		fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &d.BytesPerSector)
	}

	numSectorsPath := "/sys/block/" + blockName + "/size"
	if data, err := os.ReadFile(numSectorsPath); err == nil {
		fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &d.TotalSectors)
	}

	if d.TotalSectors > 0 && d.BytesPerSector > 0 {
		d.SizeGB = float64(d.BytesPerSector) * float64(d.TotalSectors) / (1024 * 1024 * 1024)
	}

	if d.Model == "" {
		d.Model = blockName
	}

	return d
}

func detectLinuxInterface(name string) string {
	upper := strings.ToUpper(name)
	if strings.HasPrefix(upper, "NVME") {
		return "NVMe"
	}
	if strings.HasPrefix(upper, "SD") || strings.HasPrefix(upper, "HD") {
		return "SATA"
	}
	if strings.HasPrefix(upper, "VD") || strings.HasPrefix(upper, "DM") {
		return "Virtual"
	}
	if strings.HasPrefix(upper, "LOOP") {
		return "Loop"
	}
	return "Unknown"
}

func collectSmartLinux(ctx context.Context, device string, d *DiskInfo) []output.Warning {
	smartCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(smartCtx, "smartctl", "-A", device)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	parseSmartOutput(string(out), d)
	return nil
}
