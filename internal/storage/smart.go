package storage

import (
	"context"
	"fmt"
	"runtime"
	"strings"
)

func cleanDiskDeviceName(dev string) string {
	dev = strings.TrimSpace(dev)
	if dev == "" {
		return dev
	}
	
	// If e.g. /dev/mmcblk1p1 or /dev/mmcblk1p -> /dev/mmcblk1
	if strings.Contains(dev, "mmcblk") {
		idx := strings.Index(dev, "mmcblk")
		prefix := dev[:idx+6]
		rest := dev[idx+6:]
		digits := ""
		for _, c := range rest {
			if c >= '0' && c <= '9' {
				digits += string(c)
			} else {
				break
			}
		}
		if digits != "" {
			return prefix + digits
		}
	}

	// If e.g. /dev/nvme0n1p1 -> /dev/nvme0n1
	if strings.Contains(dev, "nvme") {
		pIdx := strings.LastIndex(dev, "p")
		if pIdx > 0 && pIdx < len(dev)-1 {
			afterP := dev[pIdx+1:]
			isDigits := true
			for _, c := range afterP {
				if c < '0' || c > '9' {
					isDigits = false
					break
				}
			}
			if isDigits && len(afterP) > 0 {
				return dev[:pIdx]
			}
		}
	}

	// If e.g. /dev/sda1 -> /dev/sda
	if strings.HasPrefix(dev, "/dev/sd") || strings.HasPrefix(dev, "/dev/hd") || strings.HasPrefix(dev, "/dev/vd") {
		return strings.TrimRight(dev, "0123456789")
	}

	return dev
}

// GetSmartReportForDevice returns SMART report or fallback disk status for a device name.
func GetSmartReportForDevice(ctx context.Context, deviceName string) string {
	devName := cleanDiskDeviceName(deviceName)
	
	// Try to match disk info from collector
	var matchedDisk DiskInfo
	matchedDisk.DeviceName = devName

	stInfo, _, err := Collect(ctx)
	if err == nil && stInfo != nil {
		for _, d := range stInfo.Disks {
			cleanD := cleanDiskDeviceName(d.DeviceName)
			if cleanD == devName || strings.HasSuffix(cleanD, devName) || strings.HasSuffix(devName, cleanD) || d.DeviceName == deviceName {
				matchedDisk = d
				break
			}
		}
	}

	return GetSmartReport(ctx, matchedDisk)
}

// GetSmartReport returns full SMART output using smartctl or internal fallback.
func GetSmartReport(ctx context.Context, d DiskInfo) string {
	if d.DeviceName == "" {
		return "Ошибка: не указано имя устройства для SMART."
	}

	devName := cleanDiskDeviceName(d.DeviceName)
	devArg := devName
	if runtime.GOOS == "windows" {
		devArg = "/dev/" + devName
	}

	out, status, err := ExecSmartctl(ctx, "-a", devArg)
	if err == nil && len(out) > 0 {
		if strings.Contains(status, "системная версия") || strings.Contains(status, "встроенный") {
			return fmt.Sprintf("[%s]\n\n%s", status, string(out))
		}
		return string(out)
	}

	var b strings.Builder
	modelStr := d.Model
	if modelStr == "" {
		modelStr = devName
	}
	b.WriteString(fmt.Sprintf("=== S.M.A.R.T. Отчёт: %s (%s) ===\n\n", modelStr, devName))

	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "executable file not found") || strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "no such file") {
			b.WriteString(fmt.Sprintf("⚠️  Утилита smartctl не найдена в системе (%s/%s).\n\n", runtime.GOOS, runtime.GOARCH))
			b.WriteString("Причина: В состав SysInfoGo встроен бинарник smartctl под x86_64, но для архитектуры " + runtime.GOARCH + " (" + runtime.GOOS + ") требуется системный пакет smartmontools.\n\n")
			b.WriteString("Инструкция по установке:\n")
			if runtime.GOOS == "linux" {
				b.WriteString("  • Debian / Ubuntu / Armbian:  sudo apt update && sudo apt install -y smartmontools\n")
				b.WriteString("  • Arch Linux / Manjaro:      sudo pacman -S smartmontools\n")
				b.WriteString("  • Alpine Linux:              sudo apk add smartmontools\n")
				b.WriteString("  • Fedora / RHEL / CentOS:    sudo dnf install smartmontools\n")
			} else if runtime.GOOS == "darwin" {
				b.WriteString("  • macOS (Homebrew):          brew install smartmontools\n")
			} else {
				b.WriteString("  • Установите smartmontools с сайта https://www.smartmontools.org/\n")
			}
			b.WriteString("\n--------------------------------------------------\n")
			b.WriteString("Текущие данные датчиков SysInfoGo:\n")
		} else {
			b.WriteString(fmt.Sprintf("⚠️  smartctl завершился с ошибкой: %v\n\n", err))
			if len(out) > 0 {
				b.WriteString(string(out) + "\n")
			}
			b.WriteString("--------------------------------------------------\n")
			b.WriteString("Текущие данные датчиков SysInfoGo:\n")
		}
	}

	healthStr := d.Health
	if healthStr == "" {
		healthStr = "OK"
	}
	b.WriteString(fmt.Sprintf("Здоровье (Health): %s", healthStr))
	if d.HealthPct > 0 {
		b.WriteString(fmt.Sprintf(" (%.0f%%)", d.HealthPct))
	}
	b.WriteString("\n")
	if d.TempC > 0 {
		b.WriteString(fmt.Sprintf("Температура: %.0f °C\n", d.TempC))
	}
	if d.PowerOnHrs > 0 {
		b.WriteString(fmt.Sprintf("Время работы: %d ч.\n", d.PowerOnHrs))
	}
	if d.ReallocSec > 0 {
		b.WriteString(fmt.Sprintf("Переназначенные сектора: %d\n", d.ReallocSec))
	}
	if d.WearoutPct > 0 {
		b.WriteString(fmt.Sprintf("Износ: %.0f%%\n", d.WearoutPct))
	}
	if len(d.SMART) > 0 {
		b.WriteString("\nИндикаторы SMART:\n")
		for _, ind := range d.SMART {
			st := "OK"
			if ind.IsWarning {
				st = "ВНИМАНИЕ"
			}
			b.WriteString(fmt.Sprintf("  - %-25s: %-15s [%s]\n", ind.Name, ind.RawValue, st))
		}
	}

	return b.String()
}

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
