package cpu

import (
	"context"
	"strings"

	"github.com/user/sysinfogo/internal/output"
)

type Info struct {
	Model           string          `json:"model"`
	Vendor          string          `json:"vendor"`
	Architecture    string          `json:"architecture"`
	Socket          string          `json:"socket"`
	PhysicalCores   int             `json:"physical_cores"`
	LogicalCores    int             `json:"logical_cores"`
	NUMANodes       int             `json:"numa_nodes"`
	BaseSpeedGHz    float64         `json:"base_speed_ghz"`
	MaxSpeedGHz     float64         `json:"max_speed_ghz"`
	CurrentSpeedGHz float64         `json:"current_speed_ghz"`
	CacheL1DataKB   int             `json:"cache_l1_data_kb"`
	CacheL1InstKB   int             `json:"cache_l1_inst_kb"`
	CacheL2KB       int             `json:"cache_l2_kb"`
	CacheL3KB       int             `json:"cache_l3_kb"`
	TempPerCore     map[int]float64 `json:"temp_per_core,omitempty"`
	PackageTemp     float64         `json:"package_temp,omitempty"`
	InstructionSets []string        `json:"instruction_sets"`
	UsagePercent    float64         `json:"usage_percent"`
	UsagePerCore    []float64       `json:"usage_per_core"`
}

func Collect(ctx context.Context) (*Info, []output.Warning, error) {
	return collect(ctx)
}

func ParseFlags(flags []string) []string {
	var sets []string
	known := map[string]bool{
		"mmx": true, "sse": true, "sse2": true, "sse3": true, "ssse3": true,
		"sse4_1": true, "sse4_2": true, "avx": true, "avx2": true, "avx512f": true,
		"fma": true, "aes": true, "neon": true, "asimd": true,
	}
	for _, flag := range flags {
		f := strings.ToLower(flag)
		if known[f] {
			sets = append(sets, strings.ToUpper(f))
		}
	}
	return sets
}
