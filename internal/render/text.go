package render

import (
	"fmt"
	"strings"

	"github.com/user/sysinfogo/internal/battery"
	"github.com/user/sysinfogo/internal/cpu"
	"github.com/user/sysinfogo/internal/diagnostic"
	"github.com/user/sysinfogo/internal/gpu"
	"github.com/user/sysinfogo/internal/locale"
	mem "github.com/user/sysinfogo/internal/memory"
	"github.com/user/sysinfogo/internal/motherboard"
	"github.com/user/sysinfogo/internal/network"
	"github.com/user/sysinfogo/internal/output"
	"github.com/user/sysinfogo/internal/processes"
	"github.com/user/sysinfogo/internal/storage"
	"github.com/user/sysinfogo/internal/summary"
)

type TextFormatter struct {
	UseColor     bool
	Verbose      bool
	Units        string
	AllProcesses bool
}

func NewTextFormatter(useColor, verbose bool, units string, allProcesses bool) *TextFormatter {
	return &TextFormatter{UseColor: useColor, Verbose: verbose, Units: units, AllProcesses: allProcesses}
}

func (f *TextFormatter) Format(data *output.AggregatedData) string {
	var b strings.Builder

	for _, name := range data.SectionOrder {
		section, ok := data.Sections[name]
		if !ok || section == nil {
			continue
		}
		switch name {
		case "summary":
			f.formatSummary(&b, section)
		case "cpu":
			f.formatCPU(&b, section)
		case "memory":
			f.formatMemory(&b, section)
		case "storage":
			f.formatStorage(&b, section)
		case "gpu":
			f.formatGPU(&b, section)
		case "network":
			f.formatNetwork(&b, section)
		case "motherboard":
			f.formatMotherboard(&b, section)
		case "processes":
			f.formatProcesses(&b, section)
		case "battery":
			f.formatBattery(&b, section)
		}
		b.WriteString("\n")
	}

	if len(data.Warnings) > 0 {
		b.WriteString(output.ColorCyan + locale.T("Предупреждения:") + output.ColorReset + "\n")
		for _, w := range data.Warnings {
			b.WriteString(fmt.Sprintf("  [!] %s\n", w.Message))
			if w.OSHint != "" && f.Verbose {
				b.WriteString(fmt.Sprintf("      %s\n", w.OSHint))
			}
		}
	}

	if f.Verbose {
		b.WriteString(fmt.Sprintf("\n%s%s%s\n", output.ColorDim, locale.T("Отладочная информация:"), output.ColorReset))
		b.WriteString(fmt.Sprintf("  %s: %s\n", locale.T("ОС:"), data.OS))
		b.WriteString(fmt.Sprintf("  %s: %v\n", locale.T("Администратор:"), data.IsAdmin))
		b.WriteString(fmt.Sprintf("  %s: %s\n", locale.T("Время сбора:"), data.Timestamp))
	}

	return b.String()
}

func (f *TextFormatter) sectionTitle(name string) string {
	return output.ColorCyan + output.ColorBold + "══════ " + locale.T(name) + " ══════" + output.ColorReset + "\n"
}

func (f *TextFormatter) label(s string) string {
	return output.ColorGreen + locale.T(s) + output.ColorReset
}

func (f *TextFormatter) formatSummary(b *strings.Builder, section any) {
	info, ok := section.(*summary.Info)
	if !ok {
		return
	}

	b.WriteString(f.sectionTitle("СВОДКА"))

	b.WriteString(fmt.Sprintf("  %s %s", 	f.label("Система:"), info.OS))
	if info.Kernel != "" {
		b.WriteString(fmt.Sprintf("  |  %s %s", f.label("Ядро:"), info.Kernel))
	}
	b.WriteString(fmt.Sprintf("  |  %s %s\n", f.label("Арх:"), info.Arch))

	b.WriteString(fmt.Sprintf("  %s %s  |  %s %s\n",
		f.label("Хост:"), info.Hostname,
		f.label("Uptime:"), info.Uptime))

	b.WriteString("\n")
	cpuStr := info.CPUModel
	if info.CPUTempC > 0 {
		cpuStr += fmt.Sprintf("  |  %s %s", f.label("Temp:"), output.TempColor(info.CPUTempC, fmt.Sprintf("%.0f°C", info.CPUTempC)))
	} else {
		cpuStr += fmt.Sprintf("  |  %s %sN/A (нет доступа)%s", f.label("Temp:"), output.ColorDim, output.ColorReset)
	}
	b.WriteString(fmt.Sprintf("  %s %s\n", f.label("Процессор:"), cpuStr))
	b.WriteString(fmt.Sprintf("    %s %d %s / %d %s\n",
		f.label("Ядра:"), info.CPUCores, locale.T("физических"), info.CPULogical, locale.T("логических")))

	b.WriteString("\n")
	ramSpecStr := ""
	if info.RAMType != "" {
		ramSpecStr = " [" + info.RAMType + "]"
	}
	b.WriteString(fmt.Sprintf("  %s %s%s (%s %s, %.1f%%)\n",
		f.label("ОЗУ:"),
		output.FormatMB(info.RAMTotalGB*1024, f.Units),
		ramSpecStr,
		locale.T("занято"),
		output.FormatMB(info.RAMUsedGB*1024, f.Units),
		info.RAMUsagePct))

	if len(info.GPUs) > 0 {
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s:\n", f.label("Графика (GPU)")))
		for _, g := range info.GPUs {
			gpuStr := g.Name
			if g.VRAMMB > 0 {
				gpuStr += fmt.Sprintf(" | VRAM: %d MB", g.VRAMMB)
			}
			if g.TempC > 0 {
				gpuStr += fmt.Sprintf(" | Temp: %.0f°C", g.TempC)
			}
			if g.GPULoadPct > 0 {
				gpuStr += fmt.Sprintf(" | %s: %.1f%%", locale.T("Загрузка"), g.GPULoadPct)
			}
			b.WriteString(fmt.Sprintf("    %s\n", gpuStr))
		}
	} else if info.GPUModel != "" {
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("Графика (GPU):"), info.GPUModel))
	}

	if len(info.Storages) > 0 {
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s:\n", f.label("Накопители")))
		for i, s := range info.Storages {
			letterTag := ""
			if i < 26 {
				letterTag = fmt.Sprintf("%c)", 'a'+i)
			} else {
				letterTag = fmt.Sprintf("%d)", i+1)
			}
			tagStr := output.ColorGreen + letterTag + output.ColorReset
			if !f.UseColor {
				tagStr = letterTag
			}
			b.WriteString(fmt.Sprintf("    %-3s %-6s %s %s / %s %s\n",
				tagStr,
				s.MountPoint,
				output.FormatMB(s.TotalGB*1024, f.Units), locale.T("всего"),
				output.FormatMB(s.FreeGB*1024, f.Units), locale.T("свободно")))
		}
	}
}

func (f *TextFormatter) formatCPU(b *strings.Builder, section any) {
	info, ok := section.(*cpu.Info)
	if !ok {
		return
	}

	b.WriteString(f.sectionTitle("ПРОЦЕССОР"))
	b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("Модель:"), info.Model))
	b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("Производитель:"), info.Vendor))
	b.WriteString(fmt.Sprintf("  %-20s %d %s / %d %s\n", f.label("Ядра:"), info.PhysicalCores, locale.T("физических"), info.LogicalCores, locale.T("логических")))
	if info.BaseSpeedGHz > 0 {
		b.WriteString(fmt.Sprintf("  %-20s %.2f GHz\n", f.label("Базовая частота:"), info.BaseSpeedGHz))
	}
	if info.CurrentSpeedGHz > 0 {
		b.WriteString(fmt.Sprintf("  %-20s %.2f GHz\n", f.label("Текущая частота:"), info.CurrentSpeedGHz))
	}
	if info.CacheL1DataKB > 0 {
		b.WriteString(fmt.Sprintf("  %-20s L1: %d KB (%s), %d KB (%s)\n", f.label("Кэш:"), info.CacheL1DataKB, locale.T("данные"), info.CacheL1InstKB, locale.T("инстр.")))
	}
	if info.CacheL2KB > 0 {
		b.WriteString(fmt.Sprintf("  %-20s L2: %d KB\n", "", info.CacheL2KB))
	}
	if info.CacheL3KB > 0 {
		b.WriteString(fmt.Sprintf("  %-20s L3: %d KB\n", "", info.CacheL3KB))
	}
	if info.PackageTemp > 0 {
		tempStr := fmt.Sprintf("%.1f°C", info.PackageTemp)
		b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("Температура:"), output.TempColor(info.PackageTemp, tempStr)))
	} else {
		b.WriteString(fmt.Sprintf("  %-20s %sN/A (нет доступа к датчику)%s\n",
			f.label("Температура:"), output.ColorDim, output.ColorReset))
	}
	if info.FanSpeedRPM > 0 {
		b.WriteString(fmt.Sprintf("  %-20s %d RPM\n", f.label("Кулер CPU:"), info.FanSpeedRPM))
	} else {
		b.WriteString(fmt.Sprintf("  %-20s %sN/A (нет доступа к датчику)%s\n",
			f.label("Кулер CPU:"), output.ColorDim, output.ColorReset))
	}
	for core, temp := range info.TempPerCore {
		tempStr := fmt.Sprintf("%.0f°C", temp)
		b.WriteString(fmt.Sprintf("  %-20s %s %d: %s\n", "", locale.T("Ядро"), core, output.TempColor(temp, tempStr)))
	}
	b.WriteString(fmt.Sprintf("  %-20s %.1f%%\n", f.label("Загрузка:"), info.UsagePercent))
	if len(info.UsagePerCore) > 0 {
		b.WriteString(fmt.Sprintf("  %s:\n", f.label("По ядрам")))
		for i, u := range info.UsagePerCore {
			b.WriteString(fmt.Sprintf("    Core %d: %.1f%%\n", i, u))
		}
	}
	if len(info.InstructionSets) > 0 {
		b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("Инструкции:"), strings.Join(info.InstructionSets, ", ")))
	}
}

func (f *TextFormatter) formatMemory(b *strings.Builder, section any) {
	info, ok := section.(*mem.Info)
	if !ok {
		return
	}

	b.WriteString(f.sectionTitle("ОПЕРАТИВНАЯ ПАМЯТЬ"))
	b.WriteString(fmt.Sprintf("  %-20s %.1f GB\n", f.label("Всего:"), info.TotalGB))
	b.WriteString(fmt.Sprintf("  %-20s %.1f GB\n", f.label("Занято:"), info.UsedGB))
	b.WriteString(fmt.Sprintf("  %-20s %.1f GB\n", f.label("Свободно:"), info.FreeGB))
	b.WriteString(fmt.Sprintf("  %-20s %.1f%%\n", f.label("Загрузка:"), info.UsagePercent))
	if info.BuffersCacheGB > 0 {
		b.WriteString(fmt.Sprintf("  %-20s %.1f GB\n", f.label("Буферы/Кэш:"), info.BuffersCacheGB))
	}
	if info.SwapTotalGB > 0 {
		b.WriteString(fmt.Sprintf("  %-20s %.1f GB / %.1f GB\n", f.label("Swap:"), info.SwapUsedGB, info.SwapTotalGB))
	}
	if info.Spec != "" {
		b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("Спецификация:"), info.Spec))
	} else {
		if info.FormFactor != "" {
			b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("Форм-фактор:"), info.FormFactor))
		}
		if info.Type != "" {
			b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("Тип:"), info.Type))
		}
		if info.SpeedMTs > 0 {
			b.WriteString(fmt.Sprintf("  %-20s %d MT/s\n", f.label("Частота:"), info.SpeedMTs))
		}
	}
	if info.Model != "" && !strings.Contains(info.Spec, info.Model) {
		b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("Модель/PartNo:"), info.Model))
	}
	b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("Тайминги:"), info.Timings))
	if info.TotalSlots > 0 {
		b.WriteString(fmt.Sprintf("  %-20s %d %s, %d %s\n", f.label("Слоты:"), info.TotalSlots, locale.T("всего"), info.UsedSlots, locale.T("занято")))
	}
	for _, slot := range info.Slots {
		details := ""
		if slot.FormFactor != "" {
			details += slot.FormFactor + " "
		}
		if slot.Type != "" {
			details += slot.Type + " "
		}
		if slot.SpeedMTs > 0 {
			details += fmt.Sprintf("%d MT/s ", slot.SpeedMTs)
		}
		if slot.Model != "" {
			details += fmt.Sprintf("(%s)", slot.Model)
		}
		b.WriteString(fmt.Sprintf("    %-8s %s %d GB %s\n", slot.Locator+":", slot.Manufacturer, slot.SizeGB, strings.TrimSpace(details)))
	}
}

func (f *TextFormatter) formatStorage(b *strings.Builder, section any) {
	info, ok := section.(*storage.Info)
	if !ok {
		return
	}

	b.WriteString(f.sectionTitle("НАКОПИТЕЛИ"))
	diskIdx := 0
	for _, d := range info.Disks {
		if d.IsRAMDisk {
			f.formatRAMDisk(b, d)
			continue
		}

		letterTag := ""
		if diskIdx < 26 {
			letterTag = fmt.Sprintf("%c)", 'a'+diskIdx)
		} else {
			letterTag = fmt.Sprintf("%d)", diskIdx+1)
		}
		diskIdx++

		tagStr := output.ColorGreen + letterTag + output.ColorReset
		if !f.UseColor {
			tagStr = letterTag
		}

		mediaTag := ""
		if d.MediaType != "" {
			mediaTag = " — " + d.MediaType
		}
		hdr := fmt.Sprintf("%s | %.1f GB%s", d.Model, d.SizeGB, mediaTag)
		if d.Interface != "" {
			hdr += fmt.Sprintf(" | %s", d.Interface)
		}
		if d.RPM > 0 {
			hdr += fmt.Sprintf(" | %d RPM", d.RPM)
		}
		b.WriteString(fmt.Sprintf("  %s %s\n", tagStr, hdr))

		if d.Serial != "" {
			b.WriteString(fmt.Sprintf("    %s %s", locale.T("Серийный номер:"), d.Serial))
		}
		if d.FirmwareRevision != "" {
			b.WriteString(fmt.Sprintf(" | %s %s", locale.T("Прошивка:"), d.FirmwareRevision))
		}
		if d.Serial != "" || d.FirmwareRevision != "" {
			b.WriteString("\n")
		}

		healthStr := locale.T(d.Health)
		if d.Health != "OK" {
			healthStr = output.ColorRed + healthStr + output.ColorReset
		} else {
			healthStr = output.ColorGreen + healthStr + output.ColorReset
		}
		hparts := []string{fmt.Sprintf("SMART: %s (%d%%)", healthStr, int(d.HealthPct))}
		if d.TempC > 0 {
			hparts = append(hparts, output.TempColor(d.TempC, fmt.Sprintf("%.0f°C", d.TempC)))
		}
		if d.PowerOnHrs > 0 {
			hparts = append(hparts, fmt.Sprintf("%d %s", d.PowerOnHrs, locale.T("ч.")))
		}
		if d.WearoutPct > 0 {
			hparts = append(hparts, fmt.Sprintf("%s: %.0f%%", locale.T("износ:"), d.WearoutPct))
		}
		b.WriteString(fmt.Sprintf("    %s\n", strings.Join(hparts, " | ")))

		if d.ReallocSec > 0 {
			b.WriteString(fmt.Sprintf("    %s%s: %d%s\n",
				output.ColorRed, locale.T("Переназначенные сектора:"), d.ReallocSec, output.ColorReset))
		}

		if d.BytesPerSector > 0 && d.TotalSectors > 0 {
			physGB := float64(d.BytesPerSector) * float64(d.TotalSectors) / (1024 * 1024 * 1024)
			b.WriteString(fmt.Sprintf("    %s %.1f GB (%d %s × %d %s)\n",
				locale.T("Физический объём:"), physGB, d.TotalSectors, locale.T("секторов"), d.BytesPerSector, locale.T("байт")))
		}

		if len(d.Partitions) > 0 {
			b.WriteString(fmt.Sprintf("    %s:\n", f.label("Разделы")))
		}
		for _, p := range d.Partitions {
			if p.Hidden {
				fsType := p.FSType
				if fsType == "" {
					fsType = "RAW"
				}
				typeTag := ""
				if p.Type != "" {
					typeTag = " — " + p.Type
				}
				totalStr := formatGB(p.TotalGB)
				b.WriteString(fmt.Sprintf("      [%s] %s | %s %s%s\n",
					locale.T("Скрытый"),
					fsType,
					totalStr,
					output.ColorDim,
					typeTag+output.ColorReset,
				))
			} else {
				b.WriteString(fmt.Sprintf("      %-6s %-8s %7.1f GB / %-7.1f GB (%d%%)\n",
					p.MountPoint, p.FSType, p.TotalGB, p.FreeGB, int(p.UsedPct)))
			}
		}
		b.WriteString("\n")
	}
}

func (f *TextFormatter) formatRAMDisk(b *strings.Builder, d storage.DiskInfo) {
	b.WriteString(fmt.Sprintf("  %s  %s\n", f.label("Устройство:"), d.Model))
	b.WriteString(fmt.Sprintf("    %s RAM Disk | %s %.1f GB\n", locale.T("Тип:"), locale.T("Объём:"), d.SizeGB))
	if len(d.Partitions) > 0 {
		p := d.Partitions[0]
		b.WriteString(fmt.Sprintf("      %-6s %-8s %7.1f GB / %-7.1f GB (%d%%)\n",
			p.MountPoint, p.FSType, p.TotalGB, p.FreeGB, int(p.UsedPct)))
	}
	b.WriteString("\n")
}

func formatGB(gb float64) string {
	if gb >= 1 {
		return fmt.Sprintf("%.2f GB", gb)
	}
	mb := gb * 1024
	if mb >= 1 {
		return fmt.Sprintf("%.0f MB", mb)
	}
	return fmt.Sprintf("%.0f KB", gb*1024*1024)
}

func (f *TextFormatter) formatGPU(b *strings.Builder, section any) {
	info, ok := section.(*gpu.Info)
	if !ok {
		return
	}

	b.WriteString(f.sectionTitle("ВИДЕОКАРТЫ (GPU)"))
	for _, g := range info.GPUs {
		if g.IsVirtual {
			continue
		}
		b.WriteString(fmt.Sprintf("  %-20s %s (%s)\n", f.label("Модель:"), g.Name, g.Vendor))
		if g.VRAMMB > 0 {
			b.WriteString(fmt.Sprintf("  %-20s %d MB\n", f.label("Видеопамять:"), g.VRAMMB))
		}
		if g.DriverVersion != "" {
			b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("Драйвер:"), g.DriverVersion))
		}
		if g.TempC > 0 {
			b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("Температура:"), output.TempColor(g.TempC, fmt.Sprintf("%.0f°C", g.TempC))))
		}
		if g.GPULoadPct > 0 {
			b.WriteString(fmt.Sprintf("  %-20s %.1f%%\n", f.label("Загрузка GPU:"), g.GPULoadPct))
		}
		if g.VRAMLoadPct > 0 {
			b.WriteString(fmt.Sprintf("  %-20s %.1f%%\n", f.label("Загрузка VRAM:"), g.VRAMLoadPct))
		}
		if g.FanSpeedRPM > 0 || g.FanSpeedPct > 0 {
			fanStr := ""
			if g.FanSpeedRPM > 0 && g.FanSpeedPct > 0 {
				fanStr = fmt.Sprintf("%d RPM (%.0f%%)", g.FanSpeedRPM, g.FanSpeedPct)
			} else if g.FanSpeedRPM > 0 {
				fanStr = fmt.Sprintf("%d RPM", g.FanSpeedRPM)
			} else {
				fanStr = fmt.Sprintf("%.0f%%", g.FanSpeedPct)
			}
			b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("Кулер:"), fanStr))
		} else {
			b.WriteString(fmt.Sprintf("  %-20s %sN/A (пассивное / нет датчика)%s\n",
				f.label("Кулер:"), output.ColorDim, output.ColorReset))
		}
		if g.PowerWatts > 0 {
			b.WriteString(fmt.Sprintf("  %-20s %.1f W\n", f.label("Потребление:"), g.PowerWatts))
		}
		b.WriteString("\n")
	}

	if len(info.Displays) > 0 {
		b.WriteString(f.sectionTitle("МОНИТОРЫ И ДИСПЛЕИ"))
		for _, d := range info.Displays {
			tag := ""
			if d.IsVirtual {
				tag = " [Виртуальный/Софтверный]"
			}
			resStr := d.Resolution
			if d.RefreshRate > 0 {
				resStr += fmt.Sprintf(" @ %dHz", d.RefreshRate)
			}
			if resStr != "" {
				b.WriteString(fmt.Sprintf("  %-24s %s%s\n", f.label(d.Name+":"), resStr, tag))
			} else {
				b.WriteString(fmt.Sprintf("  %-24s%s\n", f.label(d.Name+":"), tag))
			}
		}
		b.WriteString("\n")
	}
}

func (f *TextFormatter) formatNetwork(b *strings.Builder, section any) {
	info, ok := section.(*network.Info)
	if !ok {
		return
	}

	b.WriteString(f.sectionTitle("СЕТЬ"))
	for _, iface := range info.Interfaces {
		statusStr := locale.T(iface.Status)
		if iface.Status == "down" {
			statusStr = output.ColorDim + statusStr + output.ColorReset
		} else {
			statusStr = output.ColorGreen + statusStr + output.ColorReset
		}
		b.WriteString(fmt.Sprintf("  %s  Status: %s | MTU: %d\n",
			f.label(iface.Name+":"), statusStr, iface.MTU))

		if iface.MAC != "" {
			b.WriteString(fmt.Sprintf("    MAC: %s\n", iface.MAC))
		}
		for _, ip := range iface.IPv4 {
			b.WriteString(fmt.Sprintf("    IP:  %s\n", ip))
		}
		if iface.BytesRecv > 0 || iface.BytesSent > 0 {
			b.WriteString(fmt.Sprintf("    RX: %s / TX: %s\n",
				formatBytes(iface.BytesRecv), formatBytes(iface.BytesSent)))
		}
	}
	if len(info.DNSServers) > 0 {
		b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("DNS:"), strings.Join(info.DNSServers, ", ")))
	}
}

func (f *TextFormatter) formatMotherboard(b *strings.Builder, section any) {
	info, ok := section.(*motherboard.Info)
	if !ok {
		return
	}

	b.WriteString(f.sectionTitle("МАТЕРИНСКАЯ ПЛАТА"))
	if info.Manufacturer != "" {
		b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("Производитель:"), info.Manufacturer))
	}
	if info.Model != "" {
		b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("Модель:"), info.Model))
	}
	if info.Serial != "" {
		b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("Серийный номер:"), info.Serial))
	}
	if info.BiosVendor != "" {
		b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("BIOS:"), info.BiosVendor))
	}
	if info.BiosVersion != "" {
		b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("Версия BIOS:"), info.BiosVersion))
	}
	if info.BiosDate != "" {
		b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("Дата BIOS:"), info.BiosDate))
	}
}

func (f *TextFormatter) formatProcesses(b *strings.Builder, section any) {
	info, ok := section.(*processes.Info)
	if !ok || info == nil {
		return
	}

	b.WriteString(f.sectionTitle("ПРОЦЕССЫ"))
	b.WriteString(fmt.Sprintf("  %-20s %d\n", f.label("Всего процессов:"), info.TotalCount))
	if info.LoadAvg1 > 0 || info.LoadAvg5 > 0 || info.LoadAvg15 > 0 {
		b.WriteString(fmt.Sprintf("  %-20s %.2f / %.2f / %.2f\n", f.label("Load Average:"), info.LoadAvg1, info.LoadAvg5, info.LoadAvg15))
	}

	if len(info.Processes) > 0 {
		b.WriteString(fmt.Sprintf("  %s:\n", f.label("Процессы")))
		
		limit := 10
		if f.AllProcesses || len(info.Processes) < limit {
			limit = len(info.Processes)
		}
		for i := 0; i < limit; i++ {
			p := info.Processes[i]
			b.WriteString(fmt.Sprintf("    %-6d %-20s %-8.1f %-8.1f %s\n", p.PID, truncate(p.Name, 20), p.CPU, p.Memory, p.User))
		}
	}
}

func (f *TextFormatter) formatBattery(b *strings.Builder, section any) {
	info, ok := section.(*battery.Info)
	if !ok {
		return
	}

	b.WriteString(f.sectionTitle("БАТАРЕЯ"))
	if !info.Present {
		b.WriteString(fmt.Sprintf("  %s\n", output.ColorDim+locale.T("Батарея не обнаружена")+output.ColorReset))
		return
	}

	chargeStr := fmt.Sprintf("%.0f%%", info.ChargePct)
	if info.ChargePct < 20 {
		chargeStr = output.ColorRed + chargeStr + output.ColorReset
	} else if info.ChargePct < 50 {
		chargeStr = output.ColorYellow + chargeStr + output.ColorReset
	} else {
		chargeStr = output.ColorGreen + chargeStr + output.ColorReset
	}

	b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("Статус:"), locale.T(info.Status)))
	b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("Заряд:"), chargeStr))
	if info.TimeRemain != "" {
		b.WriteString(fmt.Sprintf("  %-20s %s\n", f.label("Осталось:"), info.TimeRemain))
	}
	if info.HealthPct > 0 {
		b.WriteString(fmt.Sprintf("  %-20s %.0f%%\n", f.label("Износ:"), info.HealthPct))
	}
	if info.CycleCount > 0 {
		b.WriteString(fmt.Sprintf("  %-20s %d\n", f.label("Циклов:"), info.CycleCount))
	}
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}


func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "..."
}

func (f *TextFormatter) FormatDiagnostic(diag *diagnostic.DiagnosticResult) string {
	var b strings.Builder

	b.WriteString(f.sectionTitle("ОТЧЁТ ДИАГНОСТИКИ СИСТЕМЫ И ДАТЧИКОВ"))
	b.WriteString(fmt.Sprintf("  Дата и время:          %s\n", diag.Timestamp.Format("2006-01-02 15:04:05")))
	b.WriteString(fmt.Sprintf("  Операционная система:  %s (%s)\n", diag.OS, diag.Kernel))
	b.WriteString(fmt.Sprintf("  Имя хоста:             %s\n", diag.Hostname))
	if diag.IsAdmin {
		b.WriteString(fmt.Sprintf("  Права доступа:         %s\n\n", output.Green("Администратор / Root (Полный доступ)")))
	} else {
		b.WriteString(fmt.Sprintf("  Права доступа:         %s\n\n", output.Yellow("Обычный пользователь (Ограниченный доступ к сенсорам)")))
	}

	totalWarnings := 0
	totalErrors := 0

	for _, rep := range diag.Reports {
		b.WriteString(f.sectionTitle(rep.ComponentName))
		for _, ch := range rep.Checks {
			statusStr := ""
			switch ch.Status {
			case diagnostic.StatusOK:
				statusStr = output.Green("[ OK ]")
			case diagnostic.StatusWarn:
				statusStr = output.Yellow("[ВНИМАНИЕ]")
				totalWarnings++
			case diagnostic.StatusFail:
				statusStr = output.Red("[ОШИБКА]")
				totalErrors++
			}

			b.WriteString(fmt.Sprintf("  %-42s %s", ch.Name+":", statusStr))
			if ch.Value != "" {
				b.WriteString(fmt.Sprintf("  %s", ch.Value))
			}
			b.WriteString("\n")

			if ch.ErrorMessage != "" {
				b.WriteString(fmt.Sprintf("    %s %s\n", output.Bold("Симптом:"), ch.ErrorMessage))
			}
			if ch.RootCause != "" {
				b.WriteString(fmt.Sprintf("    %s %s\n", output.Yellow("Причина:"), ch.RootCause))
			}
			if ch.Recommendation != "" {
				lines := strings.Split(ch.Recommendation, "\n")
				b.WriteString(fmt.Sprintf("    %s %s\n", output.Green("Решение:"), lines[0]))
				for _, line := range lines[1:] {
					b.WriteString(fmt.Sprintf("             %s\n", line))
				}
			}
			if ch.Status != diagnostic.StatusOK {
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}

	b.WriteString(f.sectionTitle("ИТОГ ДИАГНОСТИКИ"))
	if totalErrors == 0 && totalWarnings == 0 {
		b.WriteString(fmt.Sprintf("  %s Все подсистемы, права и сенсоры работают идеально.\n", output.Green("✔ ИДЕАЛЬНО:")))
	} else {
		b.WriteString(fmt.Sprintf("  Обнаружено предупреждений: %s, ошибок: %s\n", output.Yellow(fmt.Sprintf("%d", totalWarnings)), output.Red(fmt.Sprintf("%d", totalErrors))))
		if !diag.IsAdmin {
			b.WriteString(fmt.Sprintf("\n  %s Запустите SysInfoGo от имени Администратора (или sudo) для доступа ко всем MSR-сенсорам и сырым дискам SMART.\n", output.Bold("СОВЕТ:")))
		}
	}

	return b.String()
}
