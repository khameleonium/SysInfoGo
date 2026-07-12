package battery

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/user/sysinfogo/internal/output"
)

func collect(ctx context.Context) (*Info, []output.Warning, error) {
	batDir := "/sys/class/power_supply/BAT0"
	if _, err := os.Stat(batDir); os.IsNotExist(err) {
		return &Info{Present: false}, nil, nil
	}

	info := &Info{Present: true, Status: "неизвестно"}

	status, err := os.ReadFile(batDir + "/status")
	if err == nil {
		s := strings.TrimSpace(string(status))
		switch s {
		case "Discharging":
			info.Status = "разряжается"
		case "Charging":
			info.Status = "заряжается"
		case "Full":
			info.Status = "заряжена"
		case "Not charging":
			info.Status = "подключена к сети"
		}
	}

	chargeNow, _ := readFloat(batDir + "/charge_now")
	chargeFull, _ := readFloat(batDir + "/charge_full")
	if chargeFull > 0 {
		info.ChargePct = (chargeNow / chargeFull) * 100
		info.HealthPct = info.ChargePct
	}

	energyNow, _ := readFloat(batDir + "/energy_now")
	energyFull, _ := readFloat(batDir + "/energy_full")
	if info.ChargePct == 0 && energyFull > 0 {
		info.ChargePct = (energyNow / energyFull) * 100
	}

	chargeFullDesign, _ := readFloat(batDir + "/charge_full_design")
	energyFullDesign, _ := readFloat(batDir + "/energy_full_design")

	if chargeFullDesign > 0 && chargeFull > 0 {
		info.HealthPct = (chargeFull / chargeFullDesign) * 100
	} else if energyFullDesign > 0 && energyFull > 0 {
		info.HealthPct = (energyFull / energyFullDesign) * 100
	}

	cycleCount, _ := readFloat(batDir + "/cycle_count")
	if cycleCount > 0 {
		info.CycleCount = int(cycleCount)
	}

	return info, nil, nil
}

func readFloat(path string) (float64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var val float64
	fmt.Sscanf(strings.TrimSpace(string(data)), "%f", &val)
	return val, nil
}
