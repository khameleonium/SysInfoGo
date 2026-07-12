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

	raw := wmi.QueryList(ctx, "path", "Win32_VideoController", "get", "Name,AdapterRAM,DriverVersion,VideoProcessor")
	if raw == "" {
		gpus = append(gpus, GPUInfo{
			Name:   "Unknown GPU",
			Vendor: "Unknown",
		})
		return &Info{GPUs: gpus}, warns, nil
	}

	records := wmi.ParseList(raw)
	for _, record := range records {
		name := record["Name"]
		if name == "" {
			name = record["VideoProcessor"]
		}
		if name == "" {
			continue
		}

		g := GPUInfo{
			Name:          name,
			Vendor:        detectVendor(name),
			DriverVersion: record["DriverVersion"],
		}
		if ramStr, ok := record["AdapterRAM"]; ok && ramStr != "" {
			if ramBytes, err := strconv.ParseInt(ramStr, 10, 64); err == nil && ramBytes > 0 {
				g.VRAMMB = int(ramBytes / (1024 * 1024))
			}
		}

		nvWarns := collectNvidia(ctx, &g)
		warns = append(warns, nvWarns...)
		gpus = append(gpus, g)
	}

	if len(gpus) == 0 {
		gpus = append(gpus, GPUInfo{Name: "Unknown GPU", Vendor: "Unknown"})
	}

	return &Info{GPUs: gpus}, warns, nil
}

func collectNvidia(ctx context.Context, g *GPUInfo) []output.Warning {
	smartCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(smartCtx, "nvidia-smi",
		"--query-gpu=temperature.gpu,utilization.gpu,utilization.memory,fan.speed,power.draw",
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
