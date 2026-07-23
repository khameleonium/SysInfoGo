//go:build linux

package diagnostic

import (
	"context"
	"fmt"
	"os"
	"os/exec"

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
	report := ComponentReport{ComponentName: "Окружение и Права"}

	adminCheck := CheckItem{
		Name:  "Права суперпользователя Root (Linux)",
		Value: fmt.Sprintf("Root: %v", isAdmin),
	}
	if isAdmin {
		adminCheck.Status = StatusOK
	} else {
		adminCheck.Status = StatusWarn
		adminCheck.ErrorMessage = "Утилита запущены без привилегий root (sudo)"
		adminCheck.RootCause = "Linux ограничивает доступ обычным пользователям к прямым устройствам MSR, /dev/mem, /dev/sd* и /dev/nvme*."
		adminCheck.Recommendation = "Запустите утилиту с правами суперпользователя: 'sudo sysinfogo -d'."
		report.HasWarnings = true
	}
	report.Checks = append(report.Checks, adminCheck)

	sysfsCheck := CheckItem{Name: "Подсистема Sysfs (/sys)"}
	if _, err := os.Stat("/sys/class"); err == nil {
		sysfsCheck.Status = StatusOK
		sysfsCheck.Value = "/sys/class смонтирован и доступен"
	} else {
		sysfsCheck.Status = StatusFail
		sysfsCheck.ErrorMessage = "Виртуальная файловая система /sys не найдена"
		sysfsCheck.RootCause = "Среда контейнера (Docker/LXC) смонтирована без доступа к псевдо-ФС ядра."
		sysfsCheck.Recommendation = "При запуске контейнера передайте флаги '--privileged -v /sys:/sys:ro'."
		report.HasErrors = true
	}
	report.Checks = append(report.Checks, sysfsCheck)

	return report
}

func runCPUDiagnostics(ctx context.Context, isAdmin bool) ComponentReport {
	report := ComponentReport{ComponentName: "Процессор (CPU)"}

	cpuInfo, _, err := cpu.Collect(ctx)
	if err != nil || cpuInfo == nil {
		report.Checks = append(report.Checks, CheckItem{
			Name:           "Сбор информации CPU",
			Status:         StatusFail,
			ErrorMessage:   "Не удалось прочитать /proc/cpuinfo",
			RootCause:      "Сбой чтения /proc/cpuinfo",
			Recommendation: "Убедитесь, что /proc смонтирован.",
		})
		report.HasErrors = true
		return report
	}

	report.Checks = append(report.Checks, CheckItem{
		Name:   "Модель и конфигурация CPU",
		Status: StatusOK,
		Value:  fmt.Sprintf("%s (%d ядер / %d потоков)", cpuInfo.Model, cpuInfo.PhysicalCores, cpuInfo.LogicalCores),
	})

	tempCheck := CheckItem{Name: "Датчик температуры процессора (Linux hwmon)"}
	if cpuInfo.PackageTemp > 0 {
		tempCheck.Status = StatusOK
		tempCheck.Value = fmt.Sprintf("%.1f °C", cpuInfo.PackageTemp)
	} else {
		tempCheck.Status = StatusWarn
		tempCheck.ErrorMessage = "Температура недоступна (N/A)"
		tempCheck.RootCause = "Модули ядра для чтения датчиков (coretemp / k10temp) не загружены, либо lm-sensors не настроен."
		tempCheck.Recommendation = "1. Запустите 'sudo modprobe coretemp' (для Intel) или 'sudo modprobe k10temp' (для AMD).\n2. Установите lm-sensors: 'sudo apt install lm-sensors && sudo sensors-detect'."
		report.HasWarnings = true
	}
	fanCheck := CheckItem{Name: "Датчик кулера CPU (Linux sysfs hwmon)"}
	if cpuInfo.FanSpeedRPM > 0 {
		fanCheck.Status = StatusOK
		fanCheck.Value = fmt.Sprintf("%d RPM", cpuInfo.FanSpeedRPM)
	} else {
		fanCheck.Status = StatusWarn
		fanCheck.ErrorMessage = "Скорость кулера CPU недоступна (N/A / 0 RPM)"
		fanCheck.RootCause = "Модули ядра для чтения тахометров вентиляторов (nuiton, nct6775, it87) не загружены или /sys/class/hwmon недоступен."
		fanCheck.Recommendation = "1. Запустите 'sudo sensors-detect' для автоматической настройки модулей hwmon.\n2. Выполните 'sudo modprobe nct6775' (для большинства десктопных плат)."
		report.HasWarnings = true
	}
	report.Checks = append(report.Checks, fanCheck)

	return report
}

func runGPUDiagnostics(ctx context.Context, isAdmin bool) ComponentReport {
	report := ComponentReport{ComponentName: "Видеокарта (GPU)"}

	gpuInfo, _, _ := gpu.Collect(ctx)
	if gpuInfo == nil || len(gpuInfo.GPUs) == 0 {
		report.Checks = append(report.Checks, CheckItem{
			Name:           "Обнаружение видеокарт",
			Status:         StatusWarn,
			ErrorMessage:   "Дискретные физические GPU не обнаружены",
			RootCause:      "Система использует виртуальный framebuffer или встроенную графику без проброса драйвера.",
			Recommendation: "Установите драйверы видеокарты (nvidia-driver / mesa-vulkan-drivers).",
		})
		report.HasWarnings = true
		return report
	}

	for _, g := range gpuInfo.GPUs {
		gCheck := CheckItem{
			Name:   fmt.Sprintf("GPU: %s", g.Name),
			Status: StatusOK,
			Value:  fmt.Sprintf("VRAM: %d MB | Temp: %.0f°C", g.VRAMMB, g.TempC),
		}
		report.Checks = append(report.Checks, gCheck)

		gpuFanCheck := CheckItem{Name: fmt.Sprintf("Кулер GPU: %s", g.Name)}
		if g.FanSpeedRPM > 0 || g.FanSpeedPct > 0 {
			gpuFanCheck.Status = StatusOK
			if g.FanSpeedRPM > 0 && g.FanSpeedPct > 0 {
				gpuFanCheck.Value = fmt.Sprintf("%d RPM (%.0f%%)", g.FanSpeedRPM, g.FanSpeedPct)
			} else if g.FanSpeedRPM > 0 {
				gpuFanCheck.Value = fmt.Sprintf("%d RPM", g.FanSpeedRPM)
			} else {
				gpuFanCheck.Value = fmt.Sprintf("%.0f%%", g.FanSpeedPct)
			}
		} else {
			gpuFanCheck.Status = StatusWarn
			gpuFanCheck.ErrorMessage = "Скорость кулера GPU недоступна (N/A / 0 RPM)"
			gpuFanCheck.RootCause = "Кулеры GPU могут быть остановлены в режиме бездействия (Zero RPM Mode), либо отсутствует nvidia-smi / rocm-smi."
			gpuFanCheck.Recommendation = "1. Проверьте nvidia-smi / rocm-smi под 3D-нагрузкой.\n2. Установите проприетарные драйверы GPU."
			report.HasWarnings = true
		}
		report.Checks = append(report.Checks, gpuFanCheck)

		if g.Vendor == "NVIDIA" {
			nvCheck := CheckItem{Name: "NVIDIA Utility (nvidia-smi)"}
			if _, err := exec.LookPath("nvidia-smi"); err == nil {
				nvCheck.Status = StatusOK
				nvCheck.Value = "nvidia-smi установлен в PATH"
			} else {
				nvCheck.Status = StatusWarn
				nvCheck.ErrorMessage = "nvidia-smi не найден"
				nvCheck.RootCause = "Проприетарные драйверы NVIDIA не установлены или отсутствует пакет nvidia-utils."
				nvCheck.Recommendation = "Установите драйверы: 'sudo apt install nvidia-utils' или 'sudo dnf install xorg-x11-drv-nvidia-cuda'."
				report.HasWarnings = true
			}
			report.Checks = append(report.Checks, nvCheck)
		}
	}

	return report
}

func runRAMDiagnostics(ctx context.Context, isAdmin bool) ComponentReport {
	report := ComponentReport{ComponentName: "Оперативная память (RAM)"}
	memInfo, _, _ := memory.Collect(ctx)
	if memInfo == nil {
		report.Checks = append(report.Checks, CheckItem{Name: "Сбор ОЗУ", Status: StatusFail})
		return report
	}
	report.Checks = append(report.Checks, CheckItem{
		Name:   "Общий объём ОЗУ",
		Status: StatusOK,
		Value:  fmt.Sprintf("%.2f GB (Использовано: %.2f GB)", memInfo.TotalGB, memInfo.UsedGB),
	})
	return report
}

func runStorageDiagnostics(ctx context.Context, isAdmin bool) ComponentReport {
	report := ComponentReport{ComponentName: "Накопители и SMART"}
	stInfo, _, _ := storage.Collect(ctx)
	if stInfo == nil || len(stInfo.Disks) == 0 {
		report.Checks = append(report.Checks, CheckItem{Name: "Диски", Status: StatusFail})
		return report
	}
	report.Checks = append(report.Checks, CheckItem{
		Name:   "Накопители Linux",
		Status: StatusOK,
		Value:  fmt.Sprintf("Обнаружено дисков: %d", len(stInfo.Disks)),
	})
	for _, d := range stInfo.Disks {
		if d.IsRAMDisk {
			continue
		}
		smCheck := CheckItem{Name: fmt.Sprintf("SMART: %s", d.Model)}
		if d.HealthPct > 0 {
			smCheck.Status = StatusOK
			smCheck.Value = fmt.Sprintf("Здоровье: %s (%d%%)", d.Health, d.HealthPct)
		} else {
			smCheck.Status = StatusWarn
			smCheck.ErrorMessage = "SMART недоступен"
			if !isAdmin {
				smCheck.RootCause = "Чтение /dev/sd* и /dev/nvme* заблокировано для пользователей без прав root."
				smCheck.Recommendation = "Запустите 'sudo sysinfogo -d'."
			} else {
				smCheck.RootCause = "Утилита smartctl не установлена."
				smCheck.Recommendation = "Установите smartmontools: 'sudo apt install smartmontools'."
			}
			report.HasWarnings = true
		}
		report.Checks = append(report.Checks, smCheck)
	}
	return report
}

func runNetworkDiagnostics(ctx context.Context, isAdmin bool) ComponentReport {
	report := ComponentReport{ComponentName: "Сеть"}
	netInfo, _, _ := network.Collect(ctx)
	if netInfo != nil {
		report.Checks = append(report.Checks, CheckItem{
			Name:   "Сетевые адаптеры",
			Status: StatusOK,
			Value:  fmt.Sprintf("Активных интерфейсов: %d", len(netInfo.Interfaces)),
		})
	}
	return report
}

func runMotherboardDiagnostics(ctx context.Context, isAdmin bool) ComponentReport {
	report := ComponentReport{ComponentName: "Материнская плата"}
	mbInfo, _, _ := motherboard.Collect(ctx)
	if mbInfo != nil {
		report.Checks = append(report.Checks, CheckItem{
			Name:   "Плата / DMI",
			Status: StatusOK,
			Value:  fmt.Sprintf("%s %s", mbInfo.Manufacturer, mbInfo.Model),
		})
	}
	return report
}
