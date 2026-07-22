package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/user/sysinfogo/internal/locale"
	"github.com/user/sysinfogo/internal/network"
	"github.com/user/sysinfogo/internal/storage"
)

func createBox(title string) *tview.TextView {
	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWrap(true)
	tv.SetBorder(true).
		SetTitle(fmt.Sprintf(" %s ", title)).
		SetTitleColor(tcell.ColorGreen)

	tv.SetFocusFunc(func() {
		tv.SetBorderColor(tcell.ColorYellow)
	})
	tv.SetBlurFunc(func() {
		tv.SetBorderColor(tcell.ColorWhite)
	})
	return tv
}

func createTable(title string) *tview.Table {
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false)
	table.SetBorder(true).
		SetTitle(fmt.Sprintf(" %s ", title)).
		SetTitleColor(tcell.ColorGreen)

	table.SetFocusFunc(func() {
		table.SetBorderColor(tcell.ColorYellow)
	})
	table.SetBlurFunc(func() {
		table.SetBorderColor(tcell.ColorWhite)
	})
	return table
}

func (a *App) initWidgets() {
	T := locale.T

	a.headerWidget = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	a.summaryWidget = createBox(T("Сводка"))
	a.cpuWidget = createBox(T("Процессор"))
	a.gpuWidget = createBox(T("Видеокарта"))
	a.gpuWidget.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			a.showDisplaysData()
			return nil
		}
		return event
	})
	a.memoryWidget = createBox(T("Оперативная память"))
	a.batteryWidget = createBox(T("Батарея"))

	a.processesWidget = createTable(T("Процессы"))
	a.processesWidget.SetSelectedFunc(func(row, column int) {
		if row == 0 {
			return // header
		}
		cell := a.processesWidget.GetCell(row, 0) // PID is in column 0
		if cell == nil || cell.Text == "" {
			return
		}
		var pid int32
		fmt.Sscanf(cell.Text, "%d", &pid)

		nameCell := a.processesWidget.GetCell(row, 1) // Name is in column 1
		name := ""
		if nameCell != nil {
			name = nameCell.Text
		}

		a.selectedPID = pid
		a.killModal.SetText(fmt.Sprintf("%s %s (PID: %d)?", T("Завершить процесс"), name, pid))

		a.killModal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonIndex == 0 {
				proc, err := process.NewProcess(a.selectedPID)
				if err == nil {
					proc.Kill() // This calls TerminateProcess on Windows, which is a hard kill
				}
			}
			a.pages.HidePage("kill_modal")
			a.app.SetFocus(a.processesWidget)
		})
		a.pages.ShowPage("kill_modal")
	})

	a.storageWidget = createTable(T("Накопители"))
	a.storageWidget.SetSelectedFunc(func(row, column int) {
		if row == 0 {
			return
		}
		if a.lastData != nil {
			if sInfo, ok := a.lastData["storage"].(*storage.Info); ok {
				idx := row - 1
				if idx >= 0 && idx < len(sInfo.Disks) {
					disk := sInfo.Disks[idx]
					if disk.DeviceName != "" {
						a.showSmartData(disk.DeviceName, disk.Model)
					}
				}
			}
		}
	})

	a.networkWidget = createTable(T("Сеть"))
	a.networkWidget.SetSelectedFunc(func(row, column int) {
		if row == 0 {
			return
		}
		if a.lastData != nil {
			if nInfo, ok := a.lastData["network"].(*network.Info); ok {
				idx := row - 1
				if idx >= 0 && idx < len(nInfo.Interfaces) {
					iface := nInfo.Interfaces[idx]
					a.showNetworkGraph(iface.Name)
				}
			}
		}
	})

	a.footerWidget = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText(fmt.Sprintf("[yellow]%s:[white] q - %s | p - %s | Enter - %s", T("Горячие клавиши"), T("Выход"), T("Отобразить все процессы (вместо топ 10)"), T("Завершить процесс")))

	a.killModal = tview.NewModal().
		AddButtons([]string{T("Да"), T("Нет")})
}

func (a *App) setupLayout() {
	// Grid layout
	grid := tview.NewGrid().
		SetRows(3, 0, 0, 1). // Header, Middle1, Middle2, Footer
		SetColumns(0, 0, 0). // 3 columns
		SetBorders(false)

	// Header spanning 3 columns
	grid.AddItem(a.headerWidget, 0, 0, 1, 3, 0, 0, false)

	// Middle1: Summary (col 0), CPU + GPU flex (col 1), Processes (col 2, span 2 rows)
	cpuGpuFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.cpuWidget, 0, 1, false).
		AddItem(a.gpuWidget, 0, 1, false)

	grid.AddItem(a.summaryWidget, 1, 0, 1, 1, 0, 0, false)
	grid.AddItem(cpuGpuFlex, 1, 1, 1, 1, 0, 0, false)
	grid.AddItem(a.processesWidget, 1, 2, 2, 1, 0, 0, true) // Processes is selectable

	// Middle2: Memory (col 0), Network/Storage (col 1)
	// Actually, let's split col 0 into Memory + Battery
	leftSplit := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.memoryWidget, 0, 1, false).
		AddItem(a.batteryWidget, 0, 1, false)

	midSplit := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.storageWidget, 0, 1, false).
		AddItem(a.networkWidget, 0, 1, false)

	grid.AddItem(leftSplit, 2, 0, 1, 1, 0, 0, false)
	grid.AddItem(midSplit, 2, 1, 1, 1, 0, 0, false)

	// Footer spanning 3 columns
	grid.AddItem(a.footerWidget, 3, 0, 1, 3, 0, 0, false)

	a.focusRing = []tview.Primitive{
		a.processesWidget,
		a.storageWidget,
		a.networkWidget,
		a.summaryWidget,
		a.cpuWidget,
		a.gpuWidget,
		a.memoryWidget,
		a.batteryWidget,
	}
	a.focusIdx = 0

	a.pages.AddPage("main", grid, true, true)
	a.pages.AddPage("kill_modal", a.killModal, false, false)
}
