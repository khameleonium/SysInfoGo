package gpu

import (
	"context"

	"github.com/user/sysinfogo/internal/output"
)

type GPUInfo struct {
	Name           string  `json:"name"`
	Vendor         string  `json:"vendor"`
	VRAMMB         int     `json:"vram_mb"`
	VRAMType       string  `json:"vram_type"`
	DriverVersion  string  `json:"driver_version"`
	DriverDate     string  `json:"driver_date,omitempty"`
	TempC          float64 `json:"temp_c,omitempty"`
	GPULoadPct     float64 `json:"gpu_load_pct,omitempty"`
	VRAMLoadPct    float64 `json:"vram_load_pct,omitempty"`
	FanSpeedPct    float64 `json:"fan_speed_pct,omitempty"`
	PowerWatts     float64 `json:"power_watts,omitempty"`
	EncoderDecoder string  `json:"encoder_decoder,omitempty"`
}

type Info struct {
	GPUs []GPUInfo `json:"gpus"`
}

func Collect(ctx context.Context) (*Info, []output.Warning, error) {
	return collect(ctx)
}
