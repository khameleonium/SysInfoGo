package tui

import (
	"context"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/user/sysinfogo/internal/network"
	"github.com/user/sysinfogo/internal/output"
)

type CollectFn func(ctx context.Context) (map[string]any, []output.Warning)

type App struct {
	app              *tview.Application
	pages            *tview.Pages

	collectFn        CollectFn
	interval         time.Duration
	allProcesses     bool
	lastData         map[string]any
	selectedPID      int32
	focusRing        []tview.Primitive
	focusIdx         int
	bgNetHistory     bool
	netHistory       map[string][]float64
	activeGraphIface string
	lastNetData      map[string]network.InterfaceInfo
	lastNetTime      time.Time

	// Widgets
	killModal       *tview.Modal
	graphWidget     *tview.TextView
	headerWidget    *tview.TextView
	summaryWidget   *tview.TextView
	cpuWidget       *tview.TextView
	memoryWidget    *tview.TextView
	processesWidget *tview.Table
	storageWidget   *tview.Table
	networkWidget   *tview.Table
	batteryWidget   *tview.TextView
	footerWidget    *tview.TextView
}

func NewApp(interval time.Duration, allProcesses bool, bgNetHistory bool, collectFn CollectFn) *App {
	a := &App{
		app:          tview.NewApplication(),
		pages:        tview.NewPages(),
		collectFn:    collectFn,
		interval:     interval,
		allProcesses: allProcesses,
		bgNetHistory: bgNetHistory,
		netHistory:   make(map[string][]float64),
		lastNetData:  make(map[string]network.InterfaceInfo),
		lastNetTime:  time.Now(),
	}

	a.initWidgets()
	a.setupLayout()
	a.setupBindings()

	a.app.SetRoot(a.pages, true).EnableMouse(true)
	return a
}

func (a *App) Run(parentCtx context.Context) error {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	go func() {
		<-ctx.Done()
		a.app.Stop()
	}()

	go func() {
		// Initial fetch
		a.updateData(ctx)

		ticker := time.NewTicker(a.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.updateData(ctx)
			}
		}
	}()

	return a.app.Run()
}

func (a *App) updateData(ctx context.Context) {
	data, _ := a.collectFn(ctx)

	// Record network history if available
	if nwInfo, ok := data["network"].(*network.Info); ok {
		now := time.Now()
		elapsed := now.Sub(a.lastNetTime).Seconds()
		if elapsed > 0 {
			for i, iface := range nwInfo.Interfaces {
				if last, exists := a.lastNetData[iface.Name]; exists {
					// Convert bytes diff to Mbps: (bytes * 8) / (1024*1024) / seconds
					recvDiff := iface.BytesRecv - last.BytesRecv
					sentDiff := iface.BytesSent - last.BytesSent
					iface.SpeedRecvMbps = float64(recvDiff*8) / 1000000.0 / elapsed
					iface.SpeedSentMbps = float64(sentDiff*8) / 1000000.0 / elapsed
				}
				a.lastNetData[iface.Name] = iface
				nwInfo.Interfaces[i] = iface // update the original slice with calculated speeds

				// We only keep the last 50 points
				hist := a.netHistory[iface.Name]

				// Total throughput for this interface (Tx + Rx)
				val := iface.SpeedRecvMbps + iface.SpeedSentMbps
				hist = append(hist, val)

				if len(hist) > 50 {
					hist = hist[1:]
				}
				a.netHistory[iface.Name] = hist
			}
		}
		a.lastNetTime = now
	}

	// Update widgets in the main thread
	a.app.QueueUpdateDraw(func() {
		a.lastData = data
		a.updateWidgets(data)
	})
}

func (a *App) setupBindings() {
	var lastToggle time.Time
	a.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		frontName, _ := a.pages.GetFrontPage()
		isModalOpen := frontName != "main" && frontName != ""

		switch event.Key() {
		case tcell.KeyRune:
			switch event.Rune() {
			case 'q', 'Q', 'й', 'Й':
				if !isModalOpen {
					a.app.Stop()
					return nil
				}
			case 'p', 'P', 'з', 'З':
				if !isModalOpen {
					if time.Since(lastToggle) < 300*time.Millisecond {
						return event
					}
					lastToggle = time.Now()
					a.allProcesses = !a.allProcesses
					a.updateWidgets(a.lastData)
					return nil
				}
			}
		case tcell.KeyRight, tcell.KeyTab:
			if !isModalOpen {
				a.focusIdx = (a.focusIdx + 1) % len(a.focusRing)
				a.app.SetFocus(a.focusRing[a.focusIdx])
				return nil
			}
		case tcell.KeyLeft, tcell.KeyBacktab:
			if !isModalOpen {
				a.focusIdx = (a.focusIdx - 1 + len(a.focusRing)) % len(a.focusRing)
				a.app.SetFocus(a.focusRing[a.focusIdx])
				return nil
			}
		case tcell.KeyEscape:
			// can be used to close modals if any
		}
		return event
	})
}
