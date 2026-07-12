package gpu

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/user/sysinfogo/internal/output"
)

func getLspciVGA(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "lspci")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "VGA") || strings.Contains(line, "3D controller") {
			return line
		}
	}
	return ""
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

func collect(ctx context.Context) (*Info, []output.Warning, error) {
	var warns []output.Warning
	var gpus []GPUInfo

	g := GPUInfo{
		Name:   "Unknown GPU",
		Vendor: "Unknown",
	}

	vgaLine := getLspciVGA(ctx)
	if vgaLine != "" {
		parts := strings.SplitN(vgaLine, ": ", 2)
		if len(parts) == 2 {
			g.Name = strings.TrimSpace(parts[1])
			g.Vendor = detectVendor(g.Name)
		}
	}

	if g.Vendor == "NVIDIA" {
		warns = append(warns, collectNvidiaLinux(ctx, &g)...)
	} else if g.Vendor == "AMD" {
		warns = append(warns, collectAmdLinux(ctx, &g)...)
	} else {
		collectNvidiaLinux(ctx, &g)
		if g.Vendor == "Unknown" {
			collectAmdLinux(ctx, &g)
		}
	}

	gpus = append(gpus, g)
	return &Info{GPUs: gpus}, warns, nil
}

func collectNvidiaLinux(ctx context.Context, g *GPUInfo) []output.Warning {
	smartCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(smartCtx, "nvidia-smi",
		"--query-gpu=name,temperature.gpu,utilization.gpu,utilization.memory,fan.speed,power.draw",
		"--format=csv,noheader,nounits")
	out, err := cmd.Output()
	if err != nil {
		if g.Vendor == "NVIDIA" {
			return []output.Warning{{
				Message: "nvidia-smi не найден",
				OSHint:  "Установите драйверы NVIDIA (пакет nvidia-utils)",
			}}
		}
		return nil
	}

	fields := strings.Split(strings.TrimSpace(string(out)), ",")
	if len(fields) >= 1 {
		g.Name = strings.TrimSpace(fields[0])
		g.Vendor = "NVIDIA"
	}
	if len(fields) >= 2 {
		fmt.Sscanf(strings.TrimSpace(fields[1]), "%f", &g.TempC)
	}
	if len(fields) >= 3 {
		fmt.Sscanf(strings.TrimSpace(fields[2]), "%f", &g.GPULoadPct)
	}
	if len(fields) >= 4 {
		fmt.Sscanf(strings.TrimSpace(fields[3]), "%f", &g.VRAMLoadPct)
	}
	if len(fields) >= 5 {
		fmt.Sscanf(strings.TrimSpace(fields[4]), "%f", &g.FanSpeedPct)
	}
	if len(fields) >= 6 {
		fmt.Sscanf(strings.TrimSpace(fields[5]), "%f", &g.PowerWatts)
	}

	return nil
}

func collectAmdLinux(ctx context.Context, g *GPUInfo) []output.Warning {
	smartCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(smartCtx, "rocm-smi", "--showtemp", "--showuse", "--json")
	out, err := cmd.Output()
	if err != nil {
		if g.Vendor == "AMD" {
			return []output.Warning{{
				Message: "rocm-smi не найден",
				OSHint:  "Установите драйверы AMD ROCm",
			}}
		}
		return nil
	}

	g.Vendor = "AMD"
	_ = out

	return nil
}
