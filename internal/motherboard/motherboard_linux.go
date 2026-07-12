package motherboard

import (
	"context"
	"os"
	"strings"

	"github.com/user/sysinfogo/internal/output"
)

func collect(ctx context.Context) (*Info, []output.Warning, error) {
	info := &Info{}

	readDMIFile(info)

	return info, nil, nil
}

func readDMIFile(info *Info) {
	paths := map[string]*string{
		"/sys/class/dmi/id/board_vendor": &info.Manufacturer,
		"/sys/class/dmi/id/board_name":   &info.Model,
		"/sys/class/dmi/id/board_serial": &info.Serial,
		"/sys/class/dmi/id/bios_vendor":  &info.BiosVendor,
		"/sys/class/dmi/id/bios_version": &info.BiosVersion,
		"/sys/class/dmi/id/bios_date":    &info.BiosDate,
	}

	for path, dest := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			*dest = strings.TrimSpace(string(data))
		}
	}
}
