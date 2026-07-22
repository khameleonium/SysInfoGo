package gpu

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/user/sysinfogo/internal/output"
	"github.com/user/sysinfogo/internal/wmi"
)

func collect(ctx context.Context) (*Info, []output.Warning, error) {
	var warns []output.Warning
	var gpus []GPUInfo
	var displays []DisplayInfo

	raw := wmi.QueryList(ctx, "path", "Win32_VideoController", "get", "Name,AdapterRAM,DriverVersion,VideoProcessor,PNPDeviceID,CurrentHorizontalResolution,CurrentVerticalResolution,CurrentRefreshRate")
	if raw != "" {
		records := wmi.ParseList(raw)
		for _, record := range records {
			name := record["Name"]
			if name == "" {
				name = record["VideoProcessor"]
			}
			if name == "" {
				continue
			}

			pnpID := record["PNPDeviceID"]
			hRes, _ := strconv.Atoi(record["CurrentHorizontalResolution"])
			vRes, _ := strconv.Atoi(record["CurrentVerticalResolution"])
			refresh, _ := strconv.Atoi(record["CurrentRefreshRate"])

			isVirt := isVirtualAdapter(name, pnpID)

			if isVirt {
				disp := DisplayInfo{
					Name:        name,
					IsVirtual:   true,
					RefreshRate: refresh,
				}
				if hRes > 0 && vRes > 0 {
					disp.Resolution = fmt.Sprintf("%dx%d", hRes, vRes)
				}
				displays = append(displays, disp)
				continue
			}

			g := GPUInfo{
				Name:          name,
				Vendor:        detectVendor(name),
				DriverVersion: record["DriverVersion"],
				IsVirtual:     false,
			}
			if ramStr, ok := record["AdapterRAM"]; ok && ramStr != "" {
				if ramBytes, err := strconv.ParseInt(ramStr, 10, 64); err == nil && ramBytes > 0 {
					g.VRAMMB = int(ramBytes / (1024 * 1024))
				}
			}

			nvWarns := collectNvidia(ctx, &g)
			warns = append(warns, nvWarns...)
			gpus = append(gpus, g)

			if hRes > 0 && vRes > 0 {
				disp := DisplayInfo{
					Name:        name + " Display",
					Resolution:  fmt.Sprintf("%dx%d", hRes, vRes),
					RefreshRate: refresh,
					IsVirtual:   false,
					GPUName:     name,
				}
				displays = append(displays, disp)
			}
		}
	}

	// Also check Win32_DesktopMonitor for physical monitors
	monRaw := wmi.QueryList(ctx, "path", "Win32_DesktopMonitor", "get", "Name,MonitorManufacturer,PNPDeviceID,ScreenWidth,ScreenHeight")
	if monRaw != "" {
		monRecords := wmi.ParseList(monRaw)
		for _, rec := range monRecords {
			mName := rec["Name"]
			mMan := rec["MonitorManufacturer"]
			w, _ := strconv.Atoi(rec["ScreenWidth"])
			h, _ := strconv.Atoi(rec["ScreenHeight"])

			if mName != "" && w > 0 && h > 0 {
				fullName := mName
				if mMan != "" && !strings.Contains(fullName, mMan) {
					fullName = mMan + " " + mName
				}
				// Check if duplicate
				exists := false
				for _, d := range displays {
					if d.Resolution == fmt.Sprintf("%dx%d", w, h) && !d.IsVirtual {
						exists = true
						break
					}
				}
				if !exists {
					displays = append(displays, DisplayInfo{
						Name:       fullName,
						Resolution: fmt.Sprintf("%dx%d", w, h),
						IsVirtual:  false,
					})
				}
			}
		}
	}

	if len(gpus) == 0 {
		gpus = append(gpus, GPUInfo{Name: "Unknown GPU", Vendor: "Unknown"})
	}

	return &Info{GPUs: gpus, Displays: displays}, warns, nil
}

func isVirtualAdapter(name, pnpID string) bool {
	upperName := strings.ToUpper(name)
	upperPNP := strings.ToUpper(pnpID)
	if strings.HasPrefix(upperPNP, "ROOT\\") {
		return true
	}
	virtualKeywords := []string{
		"VIRTUAL", "INDIRECT", "PARSEC", "CITRIX", "SPACEDESK", "MIRAGE",
		"REMOTE", "BASIC DISPLAY", "SOFTWARE RENDERER", "LUMINON", "HYPER-V",
	}
	for _, kw := range virtualKeywords {
		if strings.Contains(upperName, kw) {
			return true
		}
	}
	return false
}

func collectNvidia(ctx context.Context, g *GPUInfo) []output.Warning {
	smartCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(smartCtx, "nvidia-smi",
		"--query-gpu=temperature.gpu,utilization.gpu,utilization.memory,fan.speed,power.draw,memory.total",
		"--format=csv,noheader,nounits")
	out, err := cmd.Output()
	if err != nil {
		if g.Vendor == "NVIDIA" {
			return []output.Warning{{
				Message: "nvidia-smi не найден или недоступен",
				OSHint:  "Убедитесь, что драйверы NVIDIA корректно установлены",
			}}
		}
		return nil
	}

	fields := strings.Split(strings.TrimSpace(string(out)), ",")
	if len(fields) >= 1 {
		fmt.Sscanf(strings.TrimSpace(fields[0]), "%f", &g.TempC)
	}
	if len(fields) >= 2 {
		fmt.Sscanf(strings.TrimSpace(fields[1]), "%f", &g.GPULoadPct)
	}
	if len(fields) >= 3 {
		fmt.Sscanf(strings.TrimSpace(fields[2]), "%f", &g.VRAMLoadPct)
	}
	if len(fields) >= 4 {
		fmt.Sscanf(strings.TrimSpace(fields[3]), "%f", &g.FanSpeedPct)
	}
	if len(fields) >= 5 {
		fmt.Sscanf(strings.TrimSpace(fields[4]), "%f", &g.PowerWatts)
	}
	if len(fields) >= 6 {
		var totalMB int
		if _, err := fmt.Sscanf(strings.TrimSpace(fields[5]), "%d", &totalMB); err == nil && totalMB > 0 {
			g.VRAMMB = totalMB
		}
	}

	return nil
}

func detectVendor(name string) string {
	upper := strings.ToUpper(name)
	if strings.Contains(upper, "NVIDIA") {
		return "NVIDIA"
	}
	if strings.Contains(upper, "AMD") || strings.Contains(upper, "RADEON") {
		return "AMD"
	}
	if strings.Contains(upper, "INTEL") {
		return "Intel"
	}
	return "Unknown"
}
