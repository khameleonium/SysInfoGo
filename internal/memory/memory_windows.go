package memory

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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
	raw := wmi.QueryList(ctx, "path", "Win32_PhysicalMemory", "get", "Capacity,Speed,ConfiguredClockSpeed,Manufacturer,PartNumber,SMBIOSMemoryType,FormFactor,DeviceLocator")
	if raw == "" {
		return
	}

	records := wmi.ParseList(raw)
	for _, rec := range records {
		slot := SlotInfo{
			Locator:      strings.TrimSpace(rec["DeviceLocator"]),
			Manufacturer: strings.TrimSpace(rec["Manufacturer"]),
			Model:        strings.TrimSpace(rec["PartNumber"]),
		}
		if sizeStr, ok := rec["Capacity"]; ok && sizeStr != "" {
			if sizeBytes, err := strconv.ParseUint(sizeStr, 10, 64); err == nil {
				slot.SizeGB = int(sizeBytes / (1024 * 1024 * 1024))
			}
		}
		speedVal := 0
		if speedStr, ok := rec["ConfiguredClockSpeed"]; ok && speedStr != "" {
			speedVal, _ = strconv.Atoi(speedStr)
		}
		if speedVal == 0 {
			if speedStr, ok := rec["Speed"]; ok && speedStr != "" {
				speedVal, _ = strconv.Atoi(speedStr)
			}
		}
		if speedVal > 0 {
			slot.SpeedMTs = speedVal
			if info.SpeedMTs == 0 {
				info.SpeedMTs = speedVal
			}
		}
		if typeStr, ok := rec["SMBIOSMemoryType"]; ok && typeStr != "" {
			if typ, err := strconv.Atoi(typeStr); err == nil {
				slot.Type = parseMemoryType(typ)
				if info.Type == "" && slot.Type != "Unknown" {
					info.Type = slot.Type
				}
			}
		}
		if ffStr, ok := rec["FormFactor"]; ok && ffStr != "" {
			if ff, err := strconv.Atoi(ffStr); err == nil {
				slot.FormFactor = parseFormFactor(ff)
				if info.FormFactor == "" && slot.FormFactor != "Unknown" {
					info.FormFactor = slot.FormFactor
				}
			}
		}

		if info.Model == "" && slot.Model != "" {
			info.Model = slot.Model
		}
		if info.Manufacturer == "" && slot.Manufacturer != "" {
			info.Manufacturer = slot.Manufacturer
		}

		info.Slots = append(info.Slots, slot)
		info.UsedSlots++
	}

	var specParts []string
	if info.FormFactor != "" && info.FormFactor != "Unknown" {
		specParts = append(specParts, info.FormFactor)
	}
	if info.Type != "" && info.Type != "Unknown" {
		specParts = append(specParts, info.Type)
	}
	typeAndForm := strings.Join(specParts, " ")
	if info.SpeedMTs > 0 {
		if typeAndForm != "" {
			typeAndForm += fmt.Sprintf("-%d", info.SpeedMTs)
		} else {
			typeAndForm = fmt.Sprintf("%d MHz", info.SpeedMTs)
		}
	}

	modelStr := info.Model
	if info.Manufacturer != "" {
		if modelStr != "" {
			modelStr = info.Manufacturer + " " + modelStr
		} else {
			modelStr = info.Manufacturer
		}
	}
	if modelStr != "" {
		if typeAndForm != "" {
			info.Spec = fmt.Sprintf("%s (%s)", typeAndForm, modelStr)
		} else {
			info.Spec = modelStr
		}
	} else {
		info.Spec = typeAndForm
	}

	arrayRaw := wmi.QueryList(ctx, "path", "Win32_PhysicalMemoryArray", "get", "MemoryDevices")
	if arrayRaw != "" {
		arrRecords := wmi.ParseList(arrayRaw)
		if len(arrRecords) > 0 {
			if devs, err := strconv.Atoi(arrRecords[0]["MemoryDevices"]); err == nil && devs > info.UsedSlots {
				info.TotalSlots = devs
			}
		}
	}
	if info.TotalSlots < info.UsedSlots {
		info.TotalSlots = info.UsedSlots
	}
}

func parseFormFactor(ff int) string {
	switch ff {
	case 8:
		return "DIMM"
	case 12, 17:
		return "SO-DIMM"
	case 7:
		return "SIMM"
	case 13:
		return "SRIMM"
	case 16:
		return "RIMM"
	default:
		return "DIMM"
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
