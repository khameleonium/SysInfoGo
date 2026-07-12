package gpu

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/user/sysinfogo/internal/output"
)

func collect(ctx context.Context) (*Info, []output.Warning, error) {
	var warns []output.Warning
	var gpus []GPUInfo

	smartCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(smartCtx, "system_profiler", "SPDisplaysDataType")
	out, err := cmd.Output()
	if err != nil {
		gpus = append(gpus, GPUInfo{Name: "Unknown GPU", Vendor: "Apple"})
		return &Info{GPUs: gpus}, warns, nil
	}

	g := GPUInfo{Vendor: "Apple"}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "Chipset Model:") {
			g.Name = strings.TrimPrefix(trimmed, "Chipset Model: ")
		}
		if strings.Contains(trimmed, "VRAM") {
			var mb int
			fmt.Sscanf(trimmed, "VRAM (Dynamic, Max): %d MB", &mb)
			if mb > 0 {
				g.VRAMMB = mb
			}
		}
	}

	gpus = append(gpus, g)
	return &Info{GPUs: gpus}, warns, nil
}
