package memory

import (
	"context"
	"fmt"
	"strconv"

	"github.com/shirou/gopsutil/v4/mem"
	"github.com/user/sysinfogo/internal/output"
	"github.com/user/sysinfogo/internal/wmi"
)

func collect(ctx context.Context) (*Info, []output.Warning, error) {
	v, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("virtual memory: %w", err)
	}
	s, _ := mem.SwapMemoryWithContext(ctx)

	info := &Info{
		TotalGB:        float64(v.Total) / (1024 * 1024 * 1024),
		UsedGB:         float64(v.Used) / (1024 * 1024 * 1024),
		FreeGB:         float64(v.Available) / (1024 * 1024 * 1024),
		BuffersCacheGB: float64(v.Cached+v.Buffers) / (1024 * 1024 * 1024),
		UsagePercent:   v.UsedPercent,
		Timings:        "N/A",
	}

	if s != nil {
		info.SwapTotalGB = float64(s.Total) / (1024 * 1024 * 1024)
		info.SwapUsedGB = float64(s.Used) / (1024 * 1024 * 1024)
	}

	collectWMI(ctx, info)

	return info, nil, nil
}

func collectWMI(ctx context.Context, info *Info) {
	raw := wmi.QueryList(ctx, "path", "Win32_PhysicalMemory", "get", "Capacity,Speed,Manufacturer,SMBIOSMemoryType,DeviceLocator")
	if raw == "" {
		return
	}

	records := wmi.ParseList(raw)
	for _, rec := range records {
		slot := SlotInfo{
			Locator:      rec["DeviceLocator"],
			Manufacturer: rec["Manufacturer"],
		}
		if sizeStr, ok := rec["Capacity"]; ok && sizeStr != "" {
			if sizeBytes, err := strconv.ParseUint(sizeStr, 10, 64); err == nil {
				slot.SizeGB = int(sizeBytes / (1024 * 1024 * 1024))
			}
		}
		if speedStr, ok := rec["Speed"]; ok && speedStr != "" {
			if speed, err := strconv.Atoi(speedStr); err == nil {
				slot.SpeedMTs = speed
				if info.SpeedMTs == 0 {
					info.SpeedMTs = speed
				}
			}
		}
		if typeStr, ok := rec["SMBIOSMemoryType"]; ok && typeStr != "" {
			if typ, err := strconv.Atoi(typeStr); err == nil {
				slot.Type = parseMemoryType(typ)
				info.Type = slot.Type
			}
		}
		info.Slots = append(info.Slots, slot)
		info.UsedSlots++
	}

	arrayRaw := wmi.QueryList(ctx, "path", "Win32_PhysicalMemoryArray", "get", "MemoryDevices")
	if arrayRaw != "" {
		arrRecords := wmi.ParseList(arrayRaw)
		if len(arrRecords) > 0 {
			if devs, err := strconv.Atoi(arrRecords[0]["MemoryDevices"]); err == nil {
				info.TotalSlots = devs
			}
		}
	}
}

func parseMemoryType(typ int) string {
	switch typ {
	case 20:
		return "DDR"
	case 21:
		return "DDR2"
	case 24:
		return "DDR3"
	case 26:
		return "DDR4"
	case 34:
		return "DDR5"
	case 35:
		return "LPDDR5"
	default:
		return "Unknown"
	}
}
