package tui

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/user/sysinfogo/internal/battery"
	"github.com/user/sysinfogo/internal/cpu"
	mem "github.com/user/sysinfogo/internal/memory"
	"github.com/user/sysinfogo/internal/network"
	"github.com/user/sysinfogo/internal/processes"
	"github.com/user/sysinfogo/internal/storage"
	"github.com/user/sysinfogo/internal/summary"
)

func (a *App) updateWidgets(data map[string]any) {
	if s, ok := data["summary"].(*summary.Info); ok {
		a.headerWidget.SetText(fmt.Sprintf("[green::b]SysInfoGo[white] | %s | %s | %s | Uptime: %s",
			s.Hostname, s.OS, time.Now().Format("15:04:05"), s.Uptime))

		a.summaryWidget.SetText(fmt.Sprintf(
			"[green]OS:[white] %s\n[green]Kernel:[white] %s\n[green]Arch:[white] %s\n[green]Boot:[white] %s\n",
			s.OS, s.Kernel, s.Arch, s.BootTime,
		))
	}

	if c, ok := data["cpu"].(*cpu.Info); ok {
		a.cpuWidget.SetText(fmt.Sprintf(
			"[green]Model:[white] %s\n[green]Usage:[white] %.1f%%\n[green]Temp:[white] %.1f°C\n[green]Cores:[white] %d Physical / %d Logical\n",
			c.Model, c.UsagePercent, c.PackageTemp, c.PhysicalCores, c.LogicalCores,
		))
	}

	if m, ok := data["memory"].(*mem.Info); ok {
		a.memoryWidget.SetText(fmt.Sprintf(
			"[green]Total:[white] %.1f GB\n[green]Used:[white] %.1f GB (%.1f%%)\n[green]Free:[white] %.1f GB\n",
			m.TotalGB, m.UsedGB, m.UsagePercent, m.FreeGB,
		))
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
