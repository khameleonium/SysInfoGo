package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/user/sysinfogo/internal/battery"
	"github.com/user/sysinfogo/internal/cpu"
	"github.com/user/sysinfogo/internal/diagnostic"
	"github.com/user/sysinfogo/internal/gpu"
	"github.com/user/sysinfogo/internal/locale"
	"github.com/user/sysinfogo/internal/memory"
	"github.com/user/sysinfogo/internal/motherboard"
	"github.com/user/sysinfogo/internal/network"
	"github.com/user/sysinfogo/internal/output"
	"github.com/user/sysinfogo/internal/processes"
	"github.com/user/sysinfogo/internal/render"
	"github.com/user/sysinfogo/internal/storage"
	"github.com/user/sysinfogo/internal/summary"
	"github.com/user/sysinfogo/internal/tui"
	"github.com/user/sysinfogo/internal/watch"
	"github.com/user/sysinfogo/internal/web"
)

const version = "1.7.0"

var (
	flagCPU          bool
	flagRAM          bool
	flagStorage      bool
	flagGPU          bool
	flagNetwork      bool
	flagMotherboard  bool
	flagProcesses    bool
	flagBattery      bool
	flagAll          bool
	flagAllProcesses bool
	flagSummary      bool
	flagWatch        bool
	flagTUI          bool
	flagInterval     time.Duration
	flagNoColor      bool
	flagJSON         bool
	flagVerbose      bool
	flagHelp         bool
	flagVersion      bool
	flagUnits        string
	flagLog          string
	flagLogAppend    bool
	flagWeb          bool
	flagPort         string
	flagHost         string
	flagLocal        bool
	flagHTML         string
	flagInitConfig   bool
	flagInitLocale   bool
	flagSmart        string
	flagDiagnostic   bool
)

func init() {
	cfg := loadConfig()

	// Load locale silently (optional file)
	_ = locale.Load()

	flag.BoolVar(&flagDiagnostic, "diagnostic", false, "")
	flag.BoolVar(&flagDiagnostic, "d", false, "")

	flag.StringVar(&flagSmart, "smart", "", "")
	flag.StringVar(&flagSmart, "sm", "", "")

	flag.BoolVar(&flagCPU, "cpu", false, "")
	flag.BoolVar(&flagCPU, "c", false, "")

	flag.BoolVar(&flagRAM, "ram", false, "")
	flag.BoolVar(&flagRAM, "r", false, "")

	flag.BoolVar(&flagStorage, "storage", false, "")
	flag.BoolVar(&flagStorage, "s", false, "")

	flag.BoolVar(&flagGPU, "gpu", false, "")
	flag.BoolVar(&flagGPU, "g", false, "")

	flag.BoolVar(&flagNetwork, "network", false, "")
	flag.BoolVar(&flagNetwork, "n", false, "")

	flag.BoolVar(&flagMotherboard, "motherboard", false, "")
	flag.BoolVar(&flagMotherboard, "m", false, "")

	flag.BoolVar(&flagProcesses, "processes", false, "")
	flag.BoolVar(&flagProcesses, "p", false, "")

	flag.BoolVar(&flagBattery, "battery", false, "")
	flag.BoolVar(&flagBattery, "b", false, "")

	flag.BoolVar(&flagAll, "all", false, "")
	flag.BoolVar(&flagAll, "a", false, "")

	flag.BoolVar(&flagAllProcesses, "all-processes", false, "")

	flag.BoolVar(&flagSummary, "summary", false, "")

	flag.BoolVar(&flagWatch, "watch", false, "")
	flag.BoolVar(&flagWatch, "w", false, "")
	flag.BoolVar(&flagTUI, "tui", false, "")
	flag.BoolVar(&flagTUI, "t", false, "")
	flag.DurationVar(&flagInterval, "interval", time.Duration(cfg.WatchInterval)*time.Second, "")

	flag.BoolVar(&flagNoColor, "no-color", cfg.NoColor, "")
	flag.BoolVar(&flagJSON, "json", false, "")
	flag.BoolVar(&flagJSON, "j", false, "")
	flag.BoolVar(&flagVerbose, "verbose", false, "")
	flag.BoolVar(&flagVerbose, "v", false, "")
	flag.StringVar(&flagUnits, "units", cfg.Units, "")
	flag.StringVar(&flagLog, "log", "", "")
	flag.BoolVar(&flagLogAppend, "log-append", false, "")
	flag.BoolVar(&flagWeb, "web", false, "")
	flag.StringVar(&flagPort, "port", cfg.WebPort, "")
	flag.StringVar(&flagPort, "P", cfg.WebPort, "")
	flag.StringVar(&flagHost, "host", "0.0.0.0", "")
	flag.StringVar(&flagHost, "bind", "0.0.0.0", "")
	flag.BoolVar(&flagLocal, "local", false, "")
	flag.BoolVar(&flagLocal, "l", false, "")
	flag.StringVar(&flagHTML, "html", "", "")
	flag.BoolVar(&flagHelp, "help", false, "")
	flag.BoolVar(&flagHelp, "h", false, "")
	flag.BoolVar(&flagVersion, "version", false, "")
	flag.BoolVar(&flagInitConfig, "init-config", false, "")
	flag.BoolVar(&flagInitLocale, "init-locale", false, "")

	flag.Usage = usage
}

func main() {
	enableVTProcessing()

	newArgs := make([]string, 0, len(os.Args))
	for i := 0; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "-smart" || arg == "--smart" || arg == "-sm" {
			if i+1 >= len(os.Args) || strings.HasPrefix(os.Args[i+1], "-") {
				newArgs = append(newArgs, "-smart=all")
				continue
			}
		}
		newArgs = append(newArgs, arg)
	}
	os.Args = newArgs

	flag.Parse()

	if flagHelp {
		usage()
		return
	}
	if flagVersion {
		fmt.Printf("sysinfogo v%s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
		return
	}
	if flagInitConfig {
		if err := SaveDefaultConfig("sysinfogo_config.json"); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Created sysinfogo_config.json")
		return
	}
	if flagInitLocale {
		if err := locale.SaveDefault("sysinfogo_locale.json"); err != nil {
			fmt.Printf("Error saving locale: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Created sysinfogo_locale.json")
		return
	}

	noColor := flagNoColor || flagJSON || !output.IsTTY() || !vtAvailable()
	if noColor {
		output.DisableColors()
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if flagDiagnostic {
		runDiagnostic(ctx, noColor)
		return
	}

	if flagSmart != "" {
		runSmart(ctx, flagSmart)
		return
	}

	if flagWatch {
		runWatch(ctx, noColor)
		return
	}
	if flagTUI {
		runTUI(ctx)
		return
	}
	if flagWeb {
		runWeb(ctx)
		return
	}

	sections := selectedSections()
	results, warnings := collectSections(ctx, sections)

	aggr := &output.AggregatedData{
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Hostname:     hostname(),
		OS:           fmt.Sprintf("%s %s", runtime.GOOS, runtime.GOARCH),
		IsAdmin:      checkAdmin(),
		SectionOrder: sections,
		Sections:     results,
		Warnings:     warnings,
	}

	var formatter output.Formatter
	if flagJSON {
		formatter = &output.JSONFormatter{}
	} else {
		formatter = render.NewTextFormatter(!noColor, flagVerbose, flagUnits, flagAllProcesses)
	}

	if flagHTML != "" {
		if err := web.GenerateHTML(aggr.Sections, flagHTML); err != nil {
			fmt.Fprintf(os.Stderr, "html export error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("HTML report exported to %s\n", flagHTML)
		return
	}

	fmt.Print(formatter.Format(aggr))
}

func runWeb(ctx context.Context) {
	cfg := loadConfig()
	sections := append([]string{"summary"}, allSections()...)

	hostAddr := flagHost
	if flagLocal {
		hostAddr = "127.0.0.1"
	}
	if hostAddr == "" {
		hostAddr = "0.0.0.0"
	}

	portStr := flagPort
	if portStr == "" {
		portStr = cfg.WebPort
	}
	if portStr == "" {
		portStr = "8080"
	}

	server := web.NewServer(hostAddr, portStr, cfg.BackgroundNetworkHistory, func(reqCtx context.Context) (map[string]any, []output.Warning) {
		return collectSections(reqCtx, sections)
	})
	if err := server.Start(ctx, flagInterval); err != nil {
		fmt.Fprintf(os.Stderr, "Web server error: %v\n", err)
		os.Exit(1)
	}
}

func selectedSections() []string {
	if flagAll {
		return allSections()
	}
	if flagSummary || (!flagCPU && !flagRAM && !flagStorage && !flagGPU &&
		!flagNetwork && !flagMotherboard && !flagProcesses && !flagBattery) {
		return []string{"summary"}
	}

	var sections []string
	if flagCPU {
		sections = append(sections, "cpu")
	}
	if flagRAM {
		sections = append(sections, "memory")
	}
	if flagStorage {
		sections = append(sections, "storage")
	}
	if flagGPU {
		sections = append(sections, "gpu")
	}
	if flagNetwork {
		sections = append(sections, "network")
	}
	if flagMotherboard {
		sections = append(sections, "motherboard")
	}
	if flagProcesses {
		sections = append(sections, "processes")
	}
	if flagBattery {
		sections = append(sections, "battery")
	}
	return sections
}

func allSections() []string {
	return []string{"summary", "cpu", "memory", "storage", "gpu", "network", "motherboard", "processes", "battery"}
}

func collectSections(ctx context.Context, sections []string) (map[string]any, []output.Warning) {
	results := make(map[string]any)
	var warnings []output.Warning
	var mu sync.Mutex

	var wg sync.WaitGroup
	for _, name := range sections {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			info, warns := collectSection(ctx, n)

			mu.Lock()
			results[n] = info
			warnings = append(warnings, warns...)
			mu.Unlock()
		}(name)
	}
	wg.Wait()
	return results, warnings
}

func collectSection(ctx context.Context, name string) (any, []output.Warning) {
	switch name {
	case "summary":
		info, warns, _ := summary.Collect(ctx)
		return info, warns
	case "cpu":
		info, warns, _ := cpu.Collect(ctx)
		return info, warns
	case "memory":
		info, warns, _ := memory.Collect(ctx)
		return info, warns
	case "storage":
		info, warns, _ := storage.Collect(ctx)
		return info, warns
	case "gpu":
		info, warns, _ := gpu.Collect(ctx)
		return info, warns
	case "network":
		info, warns, _ := network.Collect(ctx)
		return info, warns
	case "motherboard":
		info, warns, _ := motherboard.Collect(ctx)
		return info, warns
	case "processes":
		info, warns, _ := processes.Collect(ctx)
		return info, warns
	case "battery":
		info, warns, _ := battery.Collect(ctx)
		return info, warns
	default:
		return nil, nil
	}
}

func runTUI(ctx context.Context) {
	sections := allSections() // TUI dashboard needs all sections
	cfg := loadConfig()
	app := tui.NewApp(flagInterval, flagAllProcesses, cfg.BackgroundNetworkHistory, func(reqCtx context.Context) (map[string]any, []output.Warning) {
		return collectSections(reqCtx, sections)
	})
	if err := app.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}

func runWatch(ctx context.Context, noColor bool) {
	sections := watchSections()
	w, err := watch.New(flagInterval, flagJSON, noColor, flagVerbose, flagUnits, flagLog, flagLogAppend, sections, flagAllProcesses)
	if err != nil {
		fmt.Fprintf(os.Stderr, "watch: %v\n", err)
		os.Exit(1)
	}

	w.Run(ctx,
		func(reqCtx context.Context) (map[string]any, []output.Warning) {
			return collectSections(reqCtx, sections)
		},
	)
}

func watchSections() []string {
	if flagAll {
		return allSections()
	}

	var sections []string
	if flagCPU {
		sections = append(sections, "cpu")
	}
	if flagRAM {
		sections = append(sections, "memory")
	}
	if flagStorage {
		sections = append(sections, "storage")
	}
	if flagGPU {
		sections = append(sections, "gpu")
	}
	if flagNetwork {
		sections = append(sections, "network")
	}
	if flagMotherboard {
		sections = append(sections, "motherboard")
	}
	if flagProcesses {
		sections = append(sections, "processes")
	}
	if flagBattery {
		sections = append(sections, "battery")
	}

	if len(sections) == 0 {
		return []string{"cpu", "memory", "processes"}
	}
	return sections
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}

func checkAdmin() bool {
	return isAdmin()
}

func usage() {
	var b strings.Builder
	t := locale.T

	b.WriteString(t("SysInfoGo — кроссплатформенная утилита мониторинга системы") + "\n\n")
	b.WriteString(t("Использование: sysinfogo [флаги]") + "\n\n")
	b.WriteString(t("Флаги:") + "\n")
	b.WriteString("  -a, --all          " + t("Вывести все разделы") + "\n")
	b.WriteString("  -c, --cpu          " + t("Информация о процессоре") + "\n")
	b.WriteString("  -r, --ram          " + t("Информация об оперативной памяти") + "\n")
	b.WriteString("  -s, --storage      " + t("Накопители и S.M.A.R.T.") + "\n")
	b.WriteString("  -sm, --smart [id]  " + t("Полная SMART-информация накопителя (например: -smart a, -smart 0)") + "\n")
	b.WriteString("  -g, --gpu          " + t("Информация о видеоадаптерах") + "\n")
	b.WriteString("  -n, --network      " + t("Сетевые интерфейсы и статистика") + "\n")
	b.WriteString("  -m, --motherboard  " + t("Материнская плата и BIOS/UEFI") + "\n")
	b.WriteString("  -p, --processes    " + t("Топ процессов по CPU и RAM") + "\n")
	b.WriteString("      --all-processes " + t("Отобразить все процессы (вместо топ 10)") + "\n")
	b.WriteString("  --init-config      " + t("Создать файл sysinfogo_config.json со значениями по умолчанию") + "\n")
	b.WriteString("  --init-locale      " + t("Создать файл sysinfogo_locale.json со словарём (опционально)") + "\n")
	b.WriteString("  -b, --battery      " + t("Информация о батарее") + "\n")
	b.WriteString("  -w, --watch        " + t("Режим реального времени") + "\n")
	b.WriteString("  -t, --tui          " + t("Интерактивный TUI дашборд") + "\n")
	b.WriteString("      --web          " + t("Запустить веб-сервер дашборда") + "\n")
	b.WriteString("  -P, --port         " + t("Порт веб-сервера (по умолчанию 8080)") + "\n")
	b.WriteString("      --host         " + t("Хост связывания веб-сервера (по умолчанию 0.0.0.0)") + "\n")
	b.WriteString("  -l, --local        " + t("Запуск веб-сервера чисто локально (127.0.0.1)") + "\n")
	b.WriteString("      --interval     " + t("Интервал обновления (по умолчанию 2s)") + "\n")
	b.WriteString("  -j, --json         " + t("Вывод в формате JSON") + "\n")
	b.WriteString("  -v, --verbose      " + t("Подробный вывод с диагностикой") + "\n")
	b.WriteString("      --no-color     " + t("Отключить цветной вывод") + "\n")
	b.WriteString("      --units        " + t("Единицы измерения: auto, mb, gb (по умолчанию auto)") + "\n")
	b.WriteString("      --log          " + t("Путь к CSV-файлу для записи метрик") + "\n")
	b.WriteString("      --log-append   " + t("Дописывать в CSV-файл вместо перезаписи") + "\n")
	b.WriteString("      --summary      " + t("Общая сводка (по умолчанию)") + "\n")
	b.WriteString("  -h, --help         " + t("Справка") + "\n")
	b.WriteString("      --version      " + t("Версия утилиты") + "\n\n")
	b.WriteString(t("Примеры:") + "\n")
	b.WriteString("  sysinfogo                    " + t("# Сводка") + "\n")
	b.WriteString("  sysinfogo -s                 " + t("# Список накопителей с индексами [a], [b]") + "\n")
	b.WriteString("  sysinfogo -smart a           " + t("# SMART-отчёт для накопителя [a]") + "\n")
	b.WriteString("  sysinfogo --cpu --ram        " + t("# CPU и RAM") + "\n")
	b.WriteString("  sysinfogo -a --json          " + t("# Все разделы в JSON") + "\n")
	b.WriteString("  sysinfogo -w --interval 5s   " + t("# Watch каждые 5с") + "\n")
	b.WriteString("  sysinfogo -c -g -w --log s.log  " + t("# CPU+GPU в watch с логом") + "\n")
	fmt.Print(b.String())
}

func runSmart(ctx context.Context, target string) {
	sInfo, _, err := storage.Collect(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка сбора информации о накопителях: %v\n", err)
		os.Exit(1)
	}

	var nonRAMDisks []storage.DiskInfo
	for _, d := range sInfo.Disks {
		if !d.IsRAMDisk {
			nonRAMDisks = append(nonRAMDisks, d)
		}
	}

	if len(nonRAMDisks) == 0 {
		fmt.Println("Накопители не обнаружены.")
		return
	}

	t := strings.ToLower(strings.TrimSpace(target))

	if t == "all" {
		for i, d := range nonRAMDisks {
			letter := fmt.Sprintf("[%c]", 'a'+i)
			fmt.Printf("=== Накопитель %s: %s ===\n", letter, d.Model)
			fmt.Println(storage.GetSmartReport(ctx, d))
		}
		return
	}

	var targetDisk *storage.DiskInfo

	if len(t) == 1 && t[0] >= 'a' && t[0] <= 'z' {
		idx := int(t[0] - 'a')
		if idx >= 0 && idx < len(nonRAMDisks) {
			targetDisk = &nonRAMDisks[idx]
		}
	}

	if targetDisk == nil {
		if idx, err := strconv.Atoi(t); err == nil {
			if idx >= 0 && idx < len(nonRAMDisks) {
				targetDisk = &nonRAMDisks[idx]
			}
		}
	}

	if targetDisk == nil {
		for i, d := range nonRAMDisks {
			if strings.EqualFold(d.DeviceName, t) || strings.EqualFold("/dev/"+d.DeviceName, t) {
				targetDisk = &nonRAMDisks[i]
				break
			}
			for _, p := range d.Partitions {
				pName := strings.TrimRight(strings.TrimRight(p.MountPoint, `\`), `:`)
				if strings.EqualFold(pName, t) || strings.EqualFold(p.MountPoint, t) {
					targetDisk = &nonRAMDisks[i]
					break
				}
			}
		}
	}

	if targetDisk == nil {
		fmt.Printf("Накопитель \"%s\" не найден.\n\n", target)
		fmt.Println("Доступные накопители:")
		for i, d := range nonRAMDisks {
			fmt.Printf("  [%c] %s (%.1f GB)\n", 'a'+i, d.Model, d.SizeGB)
		}
		fmt.Println("\nПример использования: sysinfogo -smart a")
		return
	}

	fmt.Println(storage.GetSmartReport(ctx, *targetDisk))
}

func runDiagnostic(ctx context.Context, noColor bool) {
	fmt.Println("Запуск глубокой диагностики датчиков, подсистем и прав SysInfoGo...")
	diagResult := diagnostic.Run(ctx)
	formatter := render.NewTextFormatter(!noColor, flagVerbose, flagUnits, flagAllProcesses)
	fmt.Print(formatter.FormatDiagnostic(diagResult))
}
