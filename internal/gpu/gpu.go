package gpu

import (
	"context"

	"github.com/user/sysinfogo/internal/output"
)

type GPUInfo struct {
	Name           string  `json:"name"`
	Vendor         string  `json:"vendor"`
	VRAMMB         int     `json:"vram_mb"`
	VRAMType       string  `json:"vram_type,omitempty"`
	DriverVersion  string  `json:"driver_version,omitempty"`
	DriverDate     string  `json:"driver_date,omitempty"`
	TempC          float64 `json:"temp_c,omitempty"`
	GPULoadPct     float64 `json:"gpu_load_pct,omitempty"`
	VRAMLoadPct    float64 `json:"vram_load_pct,omitempty"`
	FanSpeedPct    float64 `json:"fan_speed_pct,omitempty"`
	PowerWatts     float64 `json:"power_watts,omitempty"`
	EncoderDecoder string  `json:"encoder_decoder,omitempty"`
	IsVirtual      bool    `json:"is_virtual,omitempty"`
}

type DisplayInfo struct {
	Name        string `json:"name"`
	Resolution  string `json:"resolution,omitempty"`
	RefreshRate int    `json:"refresh_rate,omitempty"`
	IsPrimary   bool   `json:"is_primary,omitempty"`
	IsVirtual   bool   `json:"is_virtual,omitempty"`
	GPUName     string `json:"gpu_name,omitempty"`
}

type Info struct {
	GPUs     []GPUInfo     `json:"gpus"`
	Displays []DisplayInfo `json:"displays,omitempty"`
}

func Collect(ctx context.Context) (*Info, []output.Warning, error) {
	return collect(ctx)
}
