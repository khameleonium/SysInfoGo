package tui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/user/sysinfogo/internal/locale"
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

		var out []byte
		var err error

		if runtime.GOOS == "windows" {
			out, err = exec.CommandContext(ctx, "smartctl", "-a", "/dev/"+deviceName).CombinedOutput()
		} else {
			out, err = exec.CommandContext(ctx, "smartctl", "-a", deviceName).CombinedOutput()
		}

		a.app.QueueUpdateDraw(func() {
			if err != nil {
				msg := fmt.Sprintf("[red]%s[white]\n\n", T("Не удалось получить SMART-данные. Убедитесь, что программа запущена с правами администратора, а утилита smartmontools установлена."))

				msg += fmt.Sprintf("[yellow]%s:[white]\n", T("Установка"))
				msg += "Windows: choco install smartmontools\n"
				msg += "Ubuntu/Debian: sudo apt-get install smartmontools\n"
				msg += "macOS: brew install smartmontools\n\n"

				msg += fmt.Sprintf("[gray]Error: %v\n%s", err, string(out))
				tv.SetText(msg)
			} else {
				tv.SetText(string(out))
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
