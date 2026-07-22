package tui

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/user/sysinfogo/internal/locale"
	"github.com/user/sysinfogo/internal/storage"
)

func (a *App) showSmartData(deviceName, model string) {
	T := locale.T

	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true).
		SetWordWrap(true)

	tv.SetBorder(true).
		SetTitle(fmt.Sprintf(" SMART: %s ", model)).
		SetTitleColor(tcell.ColorGreen)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		devArg := deviceName
		if runtime.GOOS == "windows" {
			devArg = "/dev/" + deviceName
		}
		out, status, err := storage.ExecSmartctl(ctx, "-a", devArg)

		a.app.QueueUpdateDraw(func() {
			if err != nil && len(out) == 0 {
				msg := fmt.Sprintf("[red]%s[white]\n\n", T("Не удалось получить SMART-данные. Убедитесь, что программа запущена с правами администратора."))
				msg += fmt.Sprintf("[gray]Error: %v\n", err)
				tv.SetText(msg)
			} else {
				txt := string(out)
				if strings.Contains(status, "системная версия") {
					txt = fmt.Sprintf("[green][%s][white]\n\n%s", status, txt)
				}
				tv.SetText(txt)
			}
		})
	}()

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tv, 0, 1, true).
		AddItem(tview.NewTextView().SetText(" Esc / Enter - "+T("Закрыть")).SetTextAlign(tview.AlignCenter).SetTextColor(tcell.ColorYellow), 1, 1, false)

	popup := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(flex, 0, 8, true).
			AddItem(nil, 0, 1, false), 0, 8, true).
		AddItem(nil, 0, 1, false)

	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyEnter {
			a.pages.RemovePage("smart_modal")
			a.app.SetFocus(a.storageWidget)
			return nil
		}
		return event
	})

	a.pages.AddPage("smart_modal", popup, true, true)
	a.app.SetFocus(flex)
}
