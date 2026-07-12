package motherboard

import (
	"context"
	"os/exec"
	"strings"

	"github.com/user/sysinfogo/internal/output"
)

func collect(ctx context.Context) (*Info, []output.Warning, error) {
	info := &Info{}

	cmd := exec.CommandContext(ctx, "system_profiler", "SPHardwareDataType")
	out, err := cmd.Output()
	if err != nil {
		return info, nil, nil
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Model Identifier:") {
			info.Model = strings.TrimPrefix(trimmed, "Model Identifier: ")
		}
		if strings.HasPrefix(trimmed, "Serial Number") {
			info.Serial = strings.TrimPrefix(trimmed, "Serial Number (system): ")
		}
		if strings.HasPrefix(trimmed, "Boot ROM Version:") {
			info.BiosVersion = strings.TrimPrefix(trimmed, "Boot ROM Version: ")
		}
	}

	info.Vendor = "Apple"
	info.BiosVendor = "Apple"

	return info, nil, nil
}
