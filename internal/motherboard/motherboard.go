package motherboard

import (
	"context"

	"github.com/user/sysinfogo/internal/output"
)

type Info struct {
	Manufacturer string `json:"manufacturer"`
	Model        string `json:"model"`
	Vendor       string `json:"vendor"`
	Chipset      string `json:"chipset,omitempty"`
	Serial       string `json:"serial,omitempty"`
	BiosVendor   string `json:"bios_vendor"`
	BiosVersion  string `json:"bios_version"`
	BiosDate     string `json:"bios_date"`
}

func Collect(ctx context.Context) (*Info, []output.Warning, error) {
	return collect(ctx)
}
