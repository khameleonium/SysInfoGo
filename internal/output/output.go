package output

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

type Warning struct {
	Section string `json:"section"`
	Message string `json:"message"`
	OSHint  string `json:"os_hint"`
}

type AggregatedData struct {
	Timestamp    string         `json:"timestamp"`
	Hostname     string         `json:"hostname"`
	OS           string         `json:"os"`
	IsAdmin      bool           `json:"is_admin"`
	SectionOrder []string       `json:"-"`
	Sections     map[string]any `json:"sections"`
	Warnings     []Warning      `json:"warnings"`
}

type SectionInfo interface {
	SectionName() string
}

type Formatter interface {
	Format(data *AggregatedData) string
}

func IsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func FormatMB(mb float64, unit string) string {
	switch unit {
	case "mb":
		return fmt.Sprintf("%.0f MB", mb)
	case "gb":
		return fmt.Sprintf("%.1f GB", mb/1024)
	default:
		if mb >= 1024 {
			return fmt.Sprintf("%.1f GB", mb/1024)
		}
		return fmt.Sprintf("%.0f MB", mb)
	}
}
