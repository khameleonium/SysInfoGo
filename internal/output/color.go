package output

var (
	ColorReset   = "\033[0m"
	ColorBold    = "\033[1m"
	ColorDim     = "\033[2m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
)

func DisableColors() {
	ColorReset = ""
	ColorBold = ""
	ColorDim = ""
	ColorRed = ""
	ColorGreen = ""
	ColorYellow = ""
	ColorBlue = ""
	ColorMagenta = ""
	ColorCyan = ""
}

func Red(s string) string    { return ColorRed + s + ColorReset }
func Green(s string) string  { return ColorGreen + s + ColorReset }
func Yellow(s string) string { return ColorYellow + s + ColorReset }
func Cyan(s string) string   { return ColorCyan + s + ColorReset }
func Bold(s string) string   { return ColorBold + s + ColorReset }
func Dim(s string) string    { return ColorDim + s + ColorReset }

func TempColor(temp float64, s string) string {
	switch {
	case temp > 80:
		return Red(s)
	case temp > 60:
		return Yellow(s)
	default:
		return Green(s)
	}
}

func StatusColor(ok bool, s string) string {
	if ok {
		return Green(s)
	}
	return Red(s)
}
