package storage

import (
	"testing"
)

func TestParseSmartOutputSATA(t *testing.T) {
	raw := `
ID# ATTRIBUTE_NAME          FLAG     VALUE WORST THRESH TYPE      UPDATED  WHEN_FAILED RAW_VALUE
  5 Reallocated_Sector_Ct   0x0033   100   100   010    Pre-fail  Always       -       0
  9 Power_On_Hours          0x0032   099   099   000    Old_age   Always       -       4321
194 Temperature_Celsius     0x0022   045   045   000    Old_age   Always       -       45
`
	d := &DiskInfo{}
	parseSmartOutput(raw, d)

	if d.TempC != 45 {
		t.Errorf("expected TempC=45, got %f", d.TempC)
	}
	if d.PowerOnHrs != 4321 {
		t.Errorf("expected PowerOnHrs=4321, got %d", d.PowerOnHrs)
	}
	if d.ReallocSec != 0 {
		t.Errorf("expected ReallocSec=0, got %d", d.ReallocSec)
	}
	if len(d.SMART) != 3 {
		t.Errorf("expected 3 SMART indicators, got %d", len(d.SMART))
	} else {
		foundTemp := false
		for _, s := range d.SMART {
			if s.Name == "Temperature" && s.RawValue == "45 °C" {
				foundTemp = true
			}
		}
		if !foundTemp {
			t.Errorf("missing or incorrect Temperature indicator")
		}
	}
}

func TestParseSmartOutputNVMe(t *testing.T) {
	raw := `
Critical Warning:                   0x00
Temperature:                        35 Celsius
Available Spare:                    100%
Available Spare Threshold:          10%
Percentage Used:                    5%
Data Units Read:                    1234567 [632 GB]
Data Units Written:                 7654321 [3.91 TB]
Power On Hours:                     150
Unsafe Shutdowns:                   12
Media and Data Integrity Errors:    0
`
	d := &DiskInfo{}
	parseSmartOutput(raw, d)

	if d.TempC != 35 {
		t.Errorf("expected TempC=35, got %f", d.TempC)
	}
	if d.PowerOnHrs != 150 {
		t.Errorf("expected PowerOnHrs=150, got %d", d.PowerOnHrs)
	}
	if d.WearoutPct != 5 {
		t.Errorf("expected WearoutPct=5, got %f", d.WearoutPct)
	}
	if d.Errors != 0 {
		t.Errorf("expected Errors=0, got %d", d.Errors)
	}
	if len(d.SMART) != 5 {
		t.Errorf("expected 5 SMART indicators, got %d", len(d.SMART))
	} else {
		// Percentage Used
		foundPct := false
		for _, s := range d.SMART {
			if s.Name == "Percentage Used" && s.RawValue == "5%" {
				foundPct = true
			}
		}
		if !foundPct {
			t.Errorf("missing or incorrect Percentage Used indicator")
		}
	}
}
