package storage

import (
	"fmt"
	"strings"
)

func parseSmartOutput(raw string, d *DiskInfo) {
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, "Temperature_Celsius") || (strings.Contains(line, "Temperature") && !strings.Contains(line, "Sensor") && d.TempC == 0) {
			if v, ok := parseSmartValue(line); ok {
				if v > 0 {
					d.TempC = v
				}
				d.SMART = append(d.SMART, SMARTIndicator{
					Name:      "Temperature",
					RawValue:  fmt.Sprintf("%.0f °C", v),
					IsWarning: v > 55,
				})
			}
		} else if strings.Contains(line, "Power_On_Hours") || strings.Contains(line, "Power-On Hours") || strings.Contains(line, "Power On Hours") {
			if v, ok := parseSmartValue(line); ok {
				if v > 0 {
					d.PowerOnHrs = uint64(v)
				}
				d.SMART = append(d.SMART, SMARTIndicator{
					Name:      "Power On Hours",
					RawValue:  fmt.Sprintf("%d", uint64(v)),
					IsWarning: false,
				})
			}
		} else if strings.Contains(line, "Reallocated_Sector") || strings.Contains(line, "Reallocated_Sector_Ct") {
			if v, ok := parseSmartValue(line); ok {
				if v > 0 {
					d.ReallocSec = uint64(v)
				}
				d.SMART = append(d.SMART, SMARTIndicator{
					Name:      "Reallocated Sectors",
					RawValue:  fmt.Sprintf("%d", uint64(v)),
					IsWarning: v > 0,
				})
			}
		} else if strings.Contains(line, "Media_Wearout") || strings.Contains(line, "Wear_Leveling") || strings.Contains(line, "Percentage Used") {
			if v, ok := parseSmartValue(line); ok {
				if v > 0 {
					d.WearoutPct = v
				}
				d.SMART = append(d.SMART, SMARTIndicator{
					Name:      "Percentage Used",
					RawValue:  fmt.Sprintf("%.0f%%", v),
					IsWarning: v > 80,
				})
			}
		} else if strings.Contains(line, "Unsafe Shutdowns") {
			if v, ok := parseSmartValue(line); ok {
				d.SMART = append(d.SMART, SMARTIndicator{
					Name:      "Unsafe Shutdowns",
					RawValue:  fmt.Sprintf("%d", uint64(v)),
					IsWarning: false,
				})
			}
		} else if strings.Contains(line, "Media and Data Integrity Errors") {
			if v, ok := parseSmartValue(line); ok {
				if v > 0 {
					d.Errors = uint64(v)
				}
				d.SMART = append(d.SMART, SMARTIndicator{
					Name:      "Media Errors",
					RawValue:  fmt.Sprintf("%d", uint64(v)),
					IsWarning: v > 0,
				})
			}
		}
	}
	assessHealth(d)
}

func parseSmartValue(line string) (float64, bool) {
	if strings.Contains(line, ":") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) > 1 {
			valStr := strings.TrimSpace(parts[1])
			valStr = strings.Split(valStr, " ")[0]
			valStr = strings.TrimSuffix(valStr, "%")
			valStr = strings.ReplaceAll(valStr, ",", "")
			var f float64
			if _, err := fmt.Sscanf(valStr, "%f", &f); err == nil {
				return f, true
			}
		}
	} else {
		parts := strings.Fields(line)
		if len(parts) >= 10 {
			valStr := parts[len(parts)-1]
			var f float64
			if _, err := fmt.Sscanf(valStr, "%f", &f); err == nil {
				return f, true
			}
		} else {
			for i := 1; i < len(parts); i++ {
				var f float64
				if _, err := fmt.Sscanf(parts[i], "%f", &f); err == nil {
					if f >= 0 && f < 1000000000 {
						return f, true
					}
				}
			}
		}
	}
	return 0, false
}

func assessHealth(d *DiskInfo) {
	if d.ReallocSec > 0 || d.Errors > 0 {
		d.Health = "WARNING"
		d.HealthPct = 50
		return
	}
	if d.TempC > 55 {
		d.Health = "WARNING"
		d.HealthPct = 70
		return
	}
	if d.WearoutPct > 80 {
		d.Health = "WARNING"
		d.HealthPct = 20
		return
	}
	d.Health = "OK"
	d.HealthPct = 100
}
