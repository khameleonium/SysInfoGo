package storage

import (
	"context"

	"github.com/shirou/gopsutil/v4/disk"
	"github.com/user/sysinfogo/internal/output"
)

type SMARTIndicator struct {
	Name      string `json:"name"`
	RawValue  string `json:"raw_value"`
	IsWarning bool   `json:"is_warning"`
}

type DiskInfo struct {
	DiskNumber       int              `json:"disk_number"`
	DeviceName       string           `json:"device_name,omitempty"`
	Model            string           `json:"model"`
	Serial           string           `json:"serial,omitempty"`
	Interface        string           `json:"interface"`
	MediaType        string           `json:"media_type"`
	SizeGB           float64          `json:"size_gb"`
	FirmwareRevision string           `json:"firmware,omitempty"`
	BytesPerSector   int              `json:"bytes_per_sector,omitempty"`
	TotalSectors     uint64           `json:"total_sectors,omitempty"`
	RPM              int              `json:"rpm,omitempty"`
	PNPID            string           `json:"pnp_id,omitempty"`
	IsRAMDisk        bool             `json:"is_ramdisk"`
	Partitions       []PartitionInfo  `json:"partitions,omitempty"`
	Health           string           `json:"health"`
	HealthPct        float64          `json:"health_pct"`
	TempC            float64          `json:"temp_c,omitempty"`
	PowerOnHrs       uint64           `json:"power_on_hours,omitempty"`
	ReallocSec       uint64           `json:"reallocated_sectors,omitempty"`
	Errors           uint64           `json:"errors,omitempty"`
	WearoutPct       float64          `json:"wearout_pct,omitempty"`
	SMART            []SMARTIndicator `json:"smart,omitempty"`
	ReadBytes        uint64           `json:"read_bytes,omitempty"`
	WriteBytes       uint64           `json:"write_bytes,omitempty"`
	ReadMBps         float64          `json:"read_mbps,omitempty"`
	WriteMBps        float64          `json:"write_mbps,omitempty"`
}

type PartitionInfo struct {
	Name       string  `json:"name"`
	MountPoint string  `json:"mount_point"`
	FSType     string  `json:"fs_type"`
	TotalGB    float64 `json:"total_gb"`
	FreeGB     float64 `json:"free_gb"`
	UsedPct    float64 `json:"used_pct"`
	Type       string  `json:"type,omitempty"`
	Hidden     bool    `json:"hidden"`
}

type Info struct {
	Disks []DiskInfo `json:"disks"`
}

func Collect(ctx context.Context) (*Info, []output.Warning, error) {
	info, warns, err := collect(ctx)
	if err != nil {
		return nil, warns, err
	}

	ioCounters, err := disk.IOCountersWithContext(ctx)
	if err == nil {
		for i := range info.Disks {
			for _, p := range info.Disks[i].Partitions {
				key := p.MountPoint
				if len(key) >= 2 && key[1] == ':' {
					key = key[:2] // "C:\" -> "C:"
				} else {
					// Linux/Mac: "/dev/sda1" -> "sda1"
					if len(key) > 5 && key[:5] == "/dev/" {
						key = key[5:]
					}
					// Check by p.Name if p.MountPoint doesn't match
					if _, ok := ioCounters[key]; !ok {
						if len(p.Name) > 5 && p.Name[:5] == "/dev/" {
							key = p.Name[5:]
						} else {
							key = p.Name
						}
					}
				}
				if counter, ok := ioCounters[key]; ok {
					info.Disks[i].ReadBytes += counter.ReadBytes
					info.Disks[i].WriteBytes += counter.WriteBytes
				}
			}
		}
	}

	return info, warns, nil
}
