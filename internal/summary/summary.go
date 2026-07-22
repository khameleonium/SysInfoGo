package summary

import (
	"context"

	"github.com/user/sysinfogo/internal/output"
)

type Info struct {
	OS             string           `json:"os"`
	Kernel         string           `json:"kernel"`
	Arch           string           `json:"arch"`
	Hostname       string           `json:"hostname"`
	Uptime         string           `json:"uptime"`
	BootTime       string           `json:"boot_time"`
	Motherboard    string           `json:"motherboard"`
	BiosVersion    string           `json:"bios_version"`
	CPUModel       string           `json:"cpu_model"`
	CPUCores       int              `json:"cpu_cores"`
	CPULogical     int              `json:"cpu_logical"`
	CPUTempC       float64          `json:"cpu_temp_c,omitempty"`
	RAMTotalGB     float64          `json:"ram_total_gb"`
	RAMUsedGB      float64          `json:"ram_used_gb"`
	RAMUsagePct    float64          `json:"ram_usage_pct"`
	RAMType        string           `json:"ram_type,omitempty"`
	GPUModel       string           `json:"gpu_model,omitempty"`
	GPUs           []GPUSummary     `json:"gpus,omitempty"`
	PrimaryIP      string           `json:"primary_ip"`
	Virtualization string           `json:"virtualization,omitempty"`
	Storages       []StorageSummary `json:"storages"`
}

type GPUSummary struct {
	Name       string  `json:"name"`
	VRAMMB     int     `json:"vram_mb"`
	TempC      float64 `json:"temp_c,omitempty"`
	GPULoadPct float64 `json:"gpu_load_pct,omitempty"`
}

type StorageSummary struct {
	MountPoint string  `json:"mount_point"`
	TotalGB    float64 `json:"total_gb"`
	FreeGB     float64 `json:"free_gb"`
}

func Collect(ctx context.Context) (*Info, []output.Warning, error) {
	return collect(ctx)
}
