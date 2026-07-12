package main

import (
	"os"

	"golang.org/x/sys/windows"
)

var vtEnabled bool

func enableVTProcessing() {
	var mode uint32
	fd := os.Stdout.Fd()
	handle := windows.Handle(fd)
	if windows.GetConsoleMode(handle, &mode) != nil {
		return
	}
	mode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	if err := windows.SetConsoleMode(handle, mode); err != nil {
		return
	}
	vtEnabled = true
}

func vtAvailable() bool {
	return vtEnabled
}
