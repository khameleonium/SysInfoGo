package tui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/user/sysinfogo/internal/locale"
)

var blocks = []rune(" ▂▃▄▅▆▇█")

func generateBarChart(data []float64, width, height int) string {
	if len(data) == 0 || width <= 0 || height <= 0 {
		return ""
	}

	var max float64
	for _, v := range data {
		if v > max {
			max = v
		}
	}

	if max == 0 {
		max = 1
	}

	cols := make([]float64, width)
	startIdx := len(data) - width
	if startIdx < 0 {
		startIdx = 0
	}
	offset := width - (len(data) - startIdx)
	for i := startIdx; i < len(data); i++ {
		cols[offset+i-startIdx] = data[i]
	}

	var sb strings.Builder
	for y := height - 1; y >= 0; y-- {
		for x := 0; x < width; x++ {
			val := cols[x]
			levels := int((val / max) * float64(height*8))
			levelHere := levels - y*8
			if levelHere <= 0 {
				sb.WriteRune(' ')
			} else if levelHere >= 8 {
				sb.WriteRune(blocks[7])
			} else {
				sb.WriteRune(blocks[levelHere-1])
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (a *App) showNetworkGraph(ifaceName string) {
	T := locale.T

	if !a.bgNetHistory {
		a.netHistory[ifaceName] = nil
	}
	a.activeGraphIface = ifaceName

	if a.graphWidget == nil {
		a.graphWidget = tview.NewTextView().
			SetDynamicColors(true).
			SetWrap(false)
	}

	a.graphWidget.SetBorder(true).
		SetTitle(fmt.Sprintf(" %s: %s ", T("Активность сети"), ifaceName)).
		SetTitleColor(tcell.ColorGreen)

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.graphWidget, 0, 1, false).
		AddItem(tview.NewTextView().SetText(" Esc / Enter - "+T("Закрыть")).SetTextAlign(tview.AlignCenter).SetTextColor(tcell.ColorYellow), 1, 1, false)

	popup := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(flex, 15, 1, true).
			AddItem(nil, 0, 1, false), 0, 8, true).
		AddItem(nil, 0, 1, false)

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyEnter {
			a.activeGraphIface = ""
			a.pages.RemovePage("graph_modal")
			a.app.SetFocus(a.networkWidget)
			return nil
		}
		return event
	})

	a.pages.AddPage("graph_modal", popup, true, true)
	a.app.SetFocus(flex)

	a.updateGraphWidget()
}

func (a *App) updateGraphWidget() {
	if a.activeGraphIface == "" || a.graphWidget == nil {
		return
	}

	hist := a.netHistory[a.activeGraphIface]
	_, _, width, height := a.graphWidget.GetInnerRect()
	if width <= 0 || height <= 0 {
		return
	}

	chart := generateBarChart(hist, width, height-1)

	var current float64
	if len(hist) > 0 {
		current = hist[len(hist)-1]
	}

	text := fmt.Sprintf("[cyan]Total Traffic:[white] %.1f Mbps\n%s", current, chart)
	a.graphWidget.SetText(text)
}
