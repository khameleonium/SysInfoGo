package tui

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/user/sysinfogo/internal/battery"
	"github.com/user/sysinfogo/internal/cpu"
	"github.com/user/sysinfogo/internal/gpu"
	mem "github.com/user/sysinfogo/internal/memory"
	"github.com/user/sysinfogo/internal/network"
	"github.com/user/sysinfogo/internal/processes"
	"github.com/user/sysinfogo/internal/storage"
	"github.com/user/sysinfogo/internal/summary"
)

func makeProgressBar(pct float64, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int((pct / 100.0) * float64(width))
	if filled > width {
		filled = width
	}
	bar := ""
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := filled; i < width; i++ {
		bar += "░"
	}
	color := "[green]"
	if pct >= 85 {
		color = "[red]"
	} else if pct >= 65 {
		color = "[yellow]"
	}
	return fmt.Sprintf("%s[%s][white] %.1f%%", color, bar, pct)
}

func (a *App) updateWidgets(data map[string]any) {
	if s, ok := data["summary"].(*summary.Info); ok {
		line1 := fmt.Sprintf("[green::b]SysInfoGo[white] | [green]Host:[white] %s | [green]OS:[white] %s", s.Hostname, s.OS)
		if s.Kernel != "" {
			line1 += fmt.Sprintf(" %s", s.Kernel)
		}
		if s.Arch != "" {
			line1 += fmt.Sprintf(" (%s)", s.Arch)
		}

		line2 := fmt.Sprintf("[yellow]%s[white] | [green]Uptime:[white] %s", time.Now().Format("15:04:05"), s.Uptime)
		if s.BootTime != "" {
			line2 += fmt.Sprintf(" | [green]Boot:[white] %s", s.BootTime)
		}
		if s.Motherboard != "" {
			line2 += fmt.Sprintf(" | [green]Plata:[white] %s", s.Motherboard)
		}

		a.headerWidget.SetText(fmt.Sprintf("%s\n%s", line1, line2))
	}

	if c, ok := data["cpu"].(*cpu.Info); ok {
		cpuBar := makeProgressBar(c.UsagePercent, 14)
		tempText := "N/A (no sensor access)"
		if c.PackageTemp > 0 {
			tempText = fmt.Sprintf("%.1f°C", c.PackageTemp)
		}

		cpuTxt := fmt.Sprintf(
			"[green]Model:[white] %s\n[green]Cores:[white] %d physical / %d logical\n",
			c.Model, c.PhysicalCores, c.LogicalCores,
		)
		if c.CurrentSpeedGHz > 0 {
			cpuTxt += fmt.Sprintf("[green]Speed:[white] %.2f GHz", c.CurrentSpeedGHz)
			if c.MaxSpeedGHz > 0 {
				cpuTxt += fmt.Sprintf(" (Max: %.2f GHz)", c.MaxSpeedGHz)
			}
			cpuTxt += "\n"
		}
		if c.CacheL2KB > 0 || c.CacheL3KB > 0 {
			if c.CacheL2KB > 0 {
				cpuTxt += fmt.Sprintf("[green]L2:[white] %d KB ", c.CacheL2KB)
			}
			if c.CacheL3KB > 0 {
				cpuTxt += fmt.Sprintf("[green]L3:[white] %d KB", c.CacheL3KB)
			}
			cpuTxt += "\n"
		}
		cpuTxt += fmt.Sprintf("[green]Usage:[white] %s\n[green]Temp:[white] %s\n", cpuBar, tempText)
		if c.FanSpeedRPM > 0 {
			cpuTxt += fmt.Sprintf("[green]Fan:[white] %d RPM\n", c.FanSpeedRPM)
		} else {
			cpuTxt += "[green]Fan:[white] N/A (no sensor)\n"
		}

		a.cpuWidget.SetText(cpuTxt)
	}

	if g, ok := data["gpu"].(*gpu.Info); ok && len(g.GPUs) > 0 {
		var txt string
		count := 0
		for _, item := range g.GPUs {
			if item.IsVirtual {
				continue
			}
			count++
			txt += fmt.Sprintf("[green]Model:[white] %s\n", item.Name)
			if item.VRAMMB > 0 {
				txt += fmt.Sprintf("[green]VRAM:[white] %d MB", item.VRAMMB)
			}
			if item.TempC > 0 {
				txt += fmt.Sprintf(" | [green]Temp:[white] %.0f°C", item.TempC)
			}
			if item.FanSpeedRPM > 0 || item.FanSpeedPct > 0 {
				if item.FanSpeedRPM > 0 && item.FanSpeedPct > 0 {
					txt += fmt.Sprintf(" | [green]Fan:[white] %d RPM (%.0f%%)", item.FanSpeedRPM, item.FanSpeedPct)
				} else if item.FanSpeedRPM > 0 {
					txt += fmt.Sprintf(" | [green]Fan:[white] %d RPM", item.FanSpeedRPM)
				} else {
					txt += fmt.Sprintf(" | [green]Fan:[white] %.0f%%", item.FanSpeedPct)
				}
			} else {
				txt += " | [green]Fan:[white] N/A"
			}
			if item.GPULoadPct > 0 {
				gpuBar := makeProgressBar(item.GPULoadPct, 12)
				txt += fmt.Sprintf("\n[green]Load:[white] %s", gpuBar)
			}
			txt += "\n"
		}
		if len(g.Displays) > 0 {
			txt += fmt.Sprintf("\n[yellow]Дисплеи (%d):[white]\n", len(g.Displays))
			for _, d := range g.Displays {
				resStr := d.Resolution
				if d.RefreshRate > 0 {
					resStr += fmt.Sprintf(" @ %dHz", d.RefreshRate)
				}
				tag := ""
				if d.IsVirtual {
					tag = " [Вирт]"
				}
				txt += fmt.Sprintf(" • %s %s%s\n", d.Name, resStr, tag)
			}
		}
		if count == 0 {
			txt = "[yellow]Physical GPU not detected[white]"
		}
		a.gpuWidget.SetText(txt)
	} else if s, ok := data["summary"].(*summary.Info); ok && (len(s.GPUs) > 0 || s.GPUModel != "") {
		if len(s.GPUs) > 0 {
			gItem := s.GPUs[0]
			txt := fmt.Sprintf("[green]Model:[white] %s\n", gItem.Name)
			if gItem.VRAMMB > 0 {
				txt += fmt.Sprintf("[green]VRAM:[white] %d MB", gItem.VRAMMB)
			}
			if gItem.TempC > 0 {
				txt += fmt.Sprintf(" | [green]Temp:[white] %.0f°C", gItem.TempC)
			}
			if gItem.GPULoadPct > 0 {
				gpuBar := makeProgressBar(gItem.GPULoadPct, 12)
				txt += fmt.Sprintf("\n[green]Load:[white] %s", gpuBar)
			}
			a.gpuWidget.SetText(txt)
		} else {
			a.gpuWidget.SetText(fmt.Sprintf("[green]Model:[white] %s\n", s.GPUModel))
		}
	} else {
		a.gpuWidget.SetText("GPU data unavailable")
	}

	if m, ok := data["memory"].(*mem.Info); ok {
		ramBar := makeProgressBar(m.UsagePercent, 14)
		specStr := m.Spec
		if specStr == "" && m.Type != "" {
			specStr = m.FormFactor + " " + m.Type
			if m.SpeedMTs > 0 {
				specStr += fmt.Sprintf("-%d", m.SpeedMTs)
			}
		}
		memTxt := fmt.Sprintf(
			"[green]Total:[white] %.1f GB | [green]Used:[white] %.1f GB\n[green]Usage:[white] %s\n",
			m.TotalGB, m.UsedGB, ramBar,
		)
		if specStr != "" {
			memTxt += fmt.Sprintf("[green]Spec:[white] %s\n", specStr)
		}
		a.memoryWidget.SetText(memTxt)
	}

	if b, ok := data["battery"].(*battery.Info); ok {
		if b.Present {
			a.batteryWidget.SetText(fmt.Sprintf(
				"[green]Status:[white] %s\n[green]Charge:[white] %.0f%%\n[green]Health:[white] %.0f%%\n",
				b.Status, b.ChargePct, b.HealthPct,
			))
		} else {
			a.batteryWidget.SetText("No battery present")
		}
	}

	if p, ok := data["processes"].(*processes.Info); ok {
		row, column := a.processesWidget.GetSelection()
		rowOffset, colOffset := a.processesWidget.GetOffset()

		a.processesWidget.Clear()
		a.processesWidget.SetCell(0, 0, tview.NewTableCell("PID").SetTextColor(tcell.ColorYellow).SetSelectable(false))
		a.processesWidget.SetCell(0, 1, tview.NewTableCell("Name").SetTextColor(tcell.ColorYellow).SetSelectable(false))
		a.processesWidget.SetCell(0, 2, tview.NewTableCell("CPU%").SetTextColor(tcell.ColorYellow).SetSelectable(false))
		a.processesWidget.SetCell(0, 3, tview.NewTableCell("MEM%").SetTextColor(tcell.ColorYellow).SetSelectable(false))

		limit := 10
		if a.allProcesses || len(p.Processes) < limit {
			limit = len(p.Processes)
		}

		for i := 0; i < limit; i++ {
			proc := p.Processes[i]
			a.processesWidget.SetCell(i+1, 0, tview.NewTableCell(fmt.Sprintf("%d", proc.PID)))
			a.processesWidget.SetCell(i+1, 1, tview.NewTableCell(truncate(proc.Name, 15)))
			a.processesWidget.SetCell(i+1, 2, tview.NewTableCell(fmt.Sprintf("%.1f", proc.CPU)))
			a.processesWidget.SetCell(i+1, 3, tview.NewTableCell(fmt.Sprintf("%.1f", proc.Memory)))
		}

		a.processesWidget.Select(row, column)
		a.processesWidget.SetOffset(rowOffset, colOffset)
	}

	if st, ok := data["storage"].(*storage.Info); ok {
		a.storageWidget.Clear()
		a.storageWidget.SetCell(0, 0, tview.NewTableCell("Mount").SetTextColor(tcell.ColorYellow))
		a.storageWidget.SetCell(0, 1, tview.NewTableCell("Total").SetTextColor(tcell.ColorYellow))
		a.storageWidget.SetCell(0, 2, tview.NewTableCell("Free").SetTextColor(tcell.ColorYellow))

		for i, d := range st.Disks {
			if i >= 5 {
				break
			} // show max 5
			a.storageWidget.SetCell(i+1, 0, tview.NewTableCell(truncate(d.Model, 10)))
			a.storageWidget.SetCell(i+1, 1, tview.NewTableCell(fmt.Sprintf("%.1f GB", d.SizeGB)))
			a.storageWidget.SetCell(i+1, 2, tview.NewTableCell("-"))
		}
	}

	if nw, ok := data["network"].(*network.Info); ok {
		a.networkWidget.Clear()
		a.networkWidget.SetCell(0, 0, tview.NewTableCell("Interface").SetTextColor(tcell.ColorYellow).SetSelectable(false))
		a.networkWidget.SetCell(0, 1, tview.NewTableCell("IP").SetTextColor(tcell.ColorYellow).SetSelectable(false))
		a.networkWidget.SetCell(0, 2, tview.NewTableCell("Tx/Rx").SetTextColor(tcell.ColorYellow).SetSelectable(false))

		for i, iface := range nw.Interfaces {
			if i >= 6 {
				break
			} // show max 6

			ip := "-"
			if len(iface.IPv4) > 0 {
				ip = iface.IPv4[0]
			}

			txrx := fmt.Sprintf("%s / %s", formatBytes(iface.BytesSent), formatBytes(iface.BytesRecv))

			a.networkWidget.SetCell(i+1, 0, tview.NewTableCell(truncate(iface.Name, 15)))
			a.networkWidget.SetCell(i+1, 1, tview.NewTableCell(ip))
			a.networkWidget.SetCell(i+1, 2, tview.NewTableCell(txrx))
		}
	}

	a.updateGraphWidget()
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
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
