package storage

import (
	_ "embed"
)

// EmbeddedSmartctlVersion is the version of the bundled smartctl tool.
const EmbeddedSmartctlVersion = "7.5"

//go:embed bin/smartctl_windows.exe
var smartctlWindows []byte

//go:embed bin/smartctl_linux
var smartctlLinux []byte

//go:embed bin/smartctl_darwin
var smartctlDarwin []byte
