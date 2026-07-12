package motherboard

import (
	"context"

	"github.com/user/sysinfogo/internal/output"
	"github.com/user/sysinfogo/internal/wmi"
)

func collect(ctx context.Context) (*Info, []output.Warning, error) {
	info := &Info{}

	collectWMI(ctx, info)

	return info, nil, nil
}

func collectWMI(ctx context.Context, info *Info) {
	baseboardRaw := wmi.QueryList(ctx, "baseboard", "get", "Manufacturer,Product,SerialNumber")
	if baseboardRaw != "" {
		records := wmi.ParseList(baseboardRaw)
		if len(records) > 0 {
			info.Manufacturer = records[0]["Manufacturer"]
			info.Model = records[0]["Product"]
			info.Serial = records[0]["SerialNumber"]
		}
	}

	biosRaw := wmi.QueryList(ctx, "bios", "get", "Manufacturer,SMBIOSBIOSVersion,ReleaseDate")
	if biosRaw != "" {
		records := wmi.ParseList(biosRaw)
		if len(records) > 0 {
			info.BiosVendor = records[0]["Manufacturer"]
			info.BiosVersion = records[0]["SMBIOSBIOSVersion"]
			info.BiosDate = records[0]["ReleaseDate"]

			if len(info.BiosDate) >= 8 {
				info.BiosDate = info.BiosDate[:4] + "-" + info.BiosDate[4:6] + "-" + info.BiosDate[6:8]
			}
		}
	}
}
