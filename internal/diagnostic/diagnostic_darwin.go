//go:build darwin

package diagnostic

import (
	"context"
	"fmt"
	"os"

	"github.com/user/sysinfogo/internal/cpu"
	"github.com/user/sysinfogo/internal/gpu"
	"github.com/user/sysinfogo/internal/memory"
	"github.com/user/sysinfogo/internal/motherboard"
	"github.com/user/sysinfogo/internal/network"
	"github.com/user/sysinfogo/internal/storage"
)

func checkAdminRights() bool {
	return os.Geteuid() == 0
}

func runEnvDiagnostics(ctx context.Context, isAdmin bool) ComponentReport {
	report := ComponentReport{ComponentName: "Окружение и Права (macOS)"}

	adminCheck := CheckItem{
		Name:  "Права суперпользователя Root (macOS)",
		Value: fmt.Sprintf("Root: %v", isAdmin),
	}
	if isAdmin {
		adminCheck.Status = StatusOK
	} else {
		adminCheck.Status = StatusWarn
		adminCheck.ErrorMessage = "Утилита запущены без привилегий sudo"
		adminCheck.RootCause = "macOS блокирует сырой доступ к дисковым контроллерам IOKit без прав root."
		adminCheck.Recommendation = "Запустите утилиту через 'sudo sysinfogo -d'."
		report.HasWarnings = true
	}
	report.Checks = append(report.Checks, adminCheck)

	return report
}

func runCPUDiagnostics(ctx context.Context, isAdmin bool) ComponentReport {
	report := ComponentReport{ComponentName: "Процессор (macOS CPU)"}
	cpuInfo, _, _ := cpu.Collect(ctx)
	if cpuInfo != nil {
		report.Checks = append(report.Checks, CheckItem{
			Name:   "Конфигурация Apple Silicon / Intel",
			Status: StatusOK,
			Value:  fmt.Sprintf("%s (%d ядер)", cpuInfo.Model, cpuInfo.LogicalCores),
		})
	}
	return report
}

func runGPUDiagnostics(ctx context.Context, isAdmin bool) ComponentReport {
	report := ComponentReport{ComponentName: "Графика (macOS GPU)"}
	gpuInfo, _, _ := gpu.Collect(ctx)
	if gpuInfo != nil {
		report.Checks = append(report.Checks, CheckItem{
			Name:   "Графический адаптер",
			Status: StatusOK,
			Value:  fmt.Sprintf("Обнаружено GPU: %d", len(gpuInfo.GPUs)),
		})
	}
	return report
}

func runRAMDiagnostics(ctx context.Context, isAdmin bool) ComponentReport {
	report := ComponentReport{ComponentName: "Оперативная память"}
	memInfo, _, _ := memory.Collect(ctx)
	if memInfo != nil {
		report.Checks = append(report.Checks, CheckItem{
			Name:   "Объём RAM",
			Status: StatusOK,
			Value:  fmt.Sprintf("%.2f GB", memInfo.TotalGB),
		})
	}
	return report
}

func runStorageDiagnostics(ctx context.Context, isAdmin bool) ComponentReport {
	report := ComponentReport{ComponentName: "Накопители (macOS)"}
	stInfo, _, _ := storage.Collect(ctx)
	if stInfo != nil {
		report.Checks = append(report.Checks, CheckItem{
			Name:   "Диски APFS / NVMe",
			Status: StatusOK,
			Value:  fmt.Sprintf("Дисков: %d", len(stInfo.Disks)),
		})
	}
	return report
}

func runNetworkDiagnostics(ctx context.Context, isAdmin bool) ComponentReport {
	report := ComponentReport{ComponentName: "Сеть (macOS)"}
	netInfo, _, _ := network.Collect(ctx)
	if netInfo != nil {
		report.Checks = append(report.Checks, CheckItem{
			Name:   "Сетевые адаптеры",
			Status: StatusOK,
			Value:  fmt.Sprintf("Интерфейсов: %d", len(netInfo.Interfaces)),
		})
	}
	return report
}

func runMotherboardDiagnostics(ctx context.Context, isAdmin bool) ComponentReport {
	report := ComponentReport{ComponentName: "Материнская плата (Mac)"}
	mbInfo, _, _ := motherboard.Collect(ctx)
	if mbInfo != nil {
		report.Checks = append(report.Checks, CheckItem{
			Name:   "Apple System Model",
			Status: StatusOK,
			Value:  fmt.Sprintf("%s %s", mbInfo.Manufacturer, mbInfo.Model),
		})
	}
	return report
}
