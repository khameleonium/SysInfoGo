//go:build windows

package diagnostic

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/user/sysinfogo/internal/cpu"
	"github.com/user/sysinfogo/internal/gpu"
	"github.com/user/sysinfogo/internal/memory"
	"github.com/user/sysinfogo/internal/motherboard"
	"github.com/user/sysinfogo/internal/network"
	"github.com/user/sysinfogo/internal/storage"
	"github.com/user/sysinfogo/internal/wmi"
)

func checkAdminRights() bool {
	mod := syscall.NewLazyDLL("shell32.dll")
	proc := mod.NewProc("IsUserAnAdmin")
	ret, _, _ := proc.Call()
	return ret != 0
}

func runEnvDiagnostics(ctx context.Context, isAdmin bool) ComponentReport {
	report := ComponentReport{ComponentName: "Окружение и Права"}
	
	adminCheck := CheckItem{
		Name:  "Права Администратора (Windows)",
		Value: fmt.Sprintf("Администратор: %v", isAdmin),
	}
	if isAdmin {
		adminCheck.Status = StatusOK
	} else {
		adminCheck.Status = StatusWarn
		adminCheck.ErrorMessage = "Утилита запущены без прав Администратора"
		adminCheck.RootCause = "Windows ограничивает доступ неконсолидированных процессов к кольцу 0 (Ring 0), дескрипторам дисков \\\\.\\PhysicalDrive и регистру MSR процессора."
		adminCheck.Recommendation = "Перезапустите консоль (CMD или PowerShell) от имени Администратора и запустите SysInfoGo повторно."
		report.HasWarnings = true
	}
	report.Checks = append(report.Checks, adminCheck)

	wmiCheck := CheckItem{Name: "Служба Windows WMI (winmgmt)"}
	raw := wmi.QueryList(ctx, "path", "Win32_OperatingSystem", "get", "Caption,Version")
	if raw != "" {
		wmiCheck.Status = StatusOK
		wmiCheck.Value = "WMI подсистема отвечает корректно"
	} else {
		wmiCheck.Status = StatusFail
		wmiCheck.ErrorMessage = "Служба WMI не отвечает на запросы"
		wmiCheck.RootCause = "Служба Windows Management Instrumentation (winmgmt) остановлена или повреждена."
		wmiCheck.Recommendation = "Запустите команду 'net start winmgmt' от имени Администратора для восстановления WMI."
		report.HasErrors = true
	}
	report.Checks = append(report.Checks, wmiCheck)

	return report
}

func runCPUDiagnostics(ctx context.Context, isAdmin bool) ComponentReport {
	report := ComponentReport{ComponentName: "Процессор (CPU)"}

	cpuInfo, _, err := cpu.Collect(ctx)
	if err != nil || cpuInfo == nil {
		report.Checks = append(report.Checks, CheckItem{
			Name:           "Сбор основной информации CPU",
			Status:         StatusFail,
			ErrorMessage:   "Не удалось получить модель и ядра процессора",
			RootCause:      "Сбой библиотеки gopsutil / WMI Win32_Processor",
			Recommendation: "Проверьте целостность системы Windows (sfc /scannow).",
		})
		report.HasErrors = true
		return report
	}

	report.Checks = append(report.Checks, CheckItem{
		Name:   "Модель и конфигурация CPU",
		Status: StatusOK,
		Value:  fmt.Sprintf("%s (%d ядер / %d потоков)", cpuInfo.Model, cpuInfo.PhysicalCores, cpuInfo.LogicalCores),
	})

	// Temp check
	tempCheck := CheckItem{Name: "Датчик температуры процессора (CPU Temp)"}
	if cpuInfo.PackageTemp > 0 {
		tempCheck.Status = StatusOK
		tempCheck.Value = fmt.Sprintf("%.1f °C", cpuInfo.PackageTemp)
	} else {
		tempCheck.Status = StatusWarn
		tempCheck.ErrorMessage = "Температура недоступна (N/A)"
		if !isAdmin {
			tempCheck.RootCause = "Windows ограничивает чтение регистров MSR процессора для пользователей без прав Администратора."
			tempCheck.Recommendation = "1. Запустите SysInfoGo от имени Администратора.\n2. В качестве альтернативы запустите утилиту LibreHardwareMonitor в фоновом режиме."
		} else {
			tempCheck.RootCause = "Материнская плата или ACPI-драйвер Windows не транслируют датчик MSAcpi_ThermalZoneTemperature в WMI."
			tempCheck.Recommendation = "Запустите утилиту LibreHardwareMonitor или OpenHardwareMonitor в фоне — SysInfoGo автоматически подтянет с неё данные."
		}
		report.HasWarnings = true
	}
	report.Checks = append(report.Checks, tempCheck)

	// L3 Cache check
	cacheCheck := CheckItem{Name: "Кэш-память L3"}
	if cpuInfo.CacheL3KB > 0 {
		cacheCheck.Status = StatusOK
		cacheCheck.Value = fmt.Sprintf("%d KB (%.1f MB)", cpuInfo.CacheL3KB, float64(cpuInfo.CacheL3KB)/1024.0)
	} else {
		cacheCheck.Status = StatusWarn
		cacheCheck.ErrorMessage = "Размер L3-кэша не определён"
		cacheCheck.RootCause = "BIOS процессора не сообщил значение L3CacheSize в WMI таблицу Win32_Processor."
		cacheCheck.Recommendation = "Обновите BIOS материнской платы до актуальной версии."
		report.HasWarnings = true
	}
	report.Checks = append(report.Checks, cacheCheck)

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
			RootCause:      "Система использует базовый видеоадаптер Microsoft или виртуальный дисплей.",
			Recommendation: "Убедитесь, что графический драйвер видеокарты установлен.",
		})
		report.HasWarnings = true
		return report
	}

	for _, g := range gpuInfo.GPUs {
		gCheck := CheckItem{
			Name:   fmt.Sprintf("GPU: %s", g.Name),
			Status: StatusOK,
			Value:  fmt.Sprintf("VRAM: %d MB | Temp: %.0f°C | Load: %.0f%%", g.VRAMMB, g.TempC, g.GPULoadPct),
		}
		if g.VRAMMB <= 0 {
			gCheck.Status = StatusWarn
			gCheck.ErrorMessage = "Не удалось определить точно объём VRAM"
			gCheck.RootCause = "Драйвер видеокарты не передаёт размер видеопамяти через WMI или NVML."
			gCheck.Recommendation = "Установите или обновите фирменные драйверы производители видеокарты (NVIDIA / AMD / Intel)."
			report.HasWarnings = true
		}
		report.Checks = append(report.Checks, gCheck)

		if g.Vendor == "NVIDIA" {
			nvCheck := CheckItem{Name: fmt.Sprintf("NVIDIA Management Interface (%s)", g.Name)}
			smartCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			out, err := exec.CommandContext(smartCtx, "nvidia-smi", "--query-gpu=driver_version", "--format=csv,noheader").Output()
			cancel()
			if err == nil {
				nvCheck.Status = StatusOK
				nvCheck.Value = fmt.Sprintf("nvidia-smi доступен, вер. драйвера: %s", strings.TrimSpace(string(out)))
			} else {
				nvCheck.Status = StatusWarn
				nvCheck.ErrorMessage = "Утилита nvidia-smi недоступна или вербовала ошибку"
				nvCheck.RootCause = "Утилита nvidia-smi отсутствует в системном PATH или служба NVIDIA Display Driver не отвечает."
				nvCheck.Recommendation = "Добавьте путь 'C:\\Program Files\\NVIDIA Corporation\\NVSMI' в системную переменную PATH или переустановите драйвер NVIDIA."
				report.HasWarnings = true
			}
			report.Checks = append(report.Checks, nvCheck)
		}
	}

	return report
}

func runRAMDiagnostics(ctx context.Context, isAdmin bool) ComponentReport {
	report := ComponentReport{ComponentName: "Оперативная память (RAM)"}

	memInfo, _, err := memory.Collect(ctx)
	if err != nil || memInfo == nil {
		report.Checks = append(report.Checks, CheckItem{
			Name:           "Сбор данных RAM",
			Status:         StatusFail,
			ErrorMessage:   "Не удалось прочитать объём ОЗУ",
			RootCause:      "Сбой вызова GlobalMemoryStatusEx",
			Recommendation: "Проверьте стабильность системы.",
		})
		report.HasErrors = true
		return report
	}

	report.Checks = append(report.Checks, CheckItem{
		Name:   "Общий объём ОЗУ",
		Status: StatusOK,
		Value:  fmt.Sprintf("%.2f GB (Использовано: %.2f GB / %.0f%%)", memInfo.TotalGB, memInfo.UsedGB, memInfo.UsagePercent),
	})

	specCheck := CheckItem{Name: "Спецификация модулей ОЗУ (Тип/Частота/Модель)"}
	if memInfo.Type != "" || memInfo.SpeedMTs > 0 {
		specCheck.Status = StatusOK
		specCheck.Value = fmt.Sprintf("%s %s %d MT/s (Планка: %s %s)", memInfo.FormFactor, memInfo.Type, memInfo.SpeedMTs, memInfo.Manufacturer, memInfo.Model)
	} else {
		specCheck.Status = StatusWarn
		specCheck.ErrorMessage = "Подробные характеристики планок памяти не получены"
		specCheck.RootCause = "SMBIOS/DMI таблицы материнской платы не содержат данных о типе и частоте в классе Win32_PhysicalMemory."
		specCheck.Recommendation = "Обновите BIOS материнской платы до свежей версии."
		report.HasWarnings = true
	}
	report.Checks = append(report.Checks, specCheck)

	return report
}

func runStorageDiagnostics(ctx context.Context, isAdmin bool) ComponentReport {
	report := ComponentReport{ComponentName: "Накопители и SMART"}

	stInfo, _, err := storage.Collect(ctx)
	if err != nil || stInfo == nil || len(stInfo.Disks) == 0 {
		report.Checks = append(report.Checks, CheckItem{
			Name:           "Перечисление дисков",
			Status:         StatusFail,
			ErrorMessage:   "Не удалось обнаружить диски",
			RootCause:      "Сбой WMI Win32_DiskDrive",
			Recommendation: "Проверьте подключение накопителей.",
		})
		report.HasErrors = true
		return report
	}

	report.Checks = append(report.Checks, CheckItem{
		Name:   "Физические накопители",
		Status: StatusOK,
		Value:  fmt.Sprintf("Обнаружено дисков: %d", len(stInfo.Disks)),
	})

	for _, d := range stInfo.Disks {
		if d.IsRAMDisk {
			continue
		}
		smCheck := CheckItem{Name: fmt.Sprintf("SMART для диска: %s (Disk %d)", d.Model, d.DiskNumber)}
		if d.HealthPct > 0 && d.Health != "Unknown" {
			smCheck.Status = StatusOK
			smCheck.Value = fmt.Sprintf("Здоровье: %s (%.0f%%) | Температура: %.0f°C", d.Health, d.HealthPct, d.TempC)
		} else {
			smCheck.Status = StatusWarn
			smCheck.ErrorMessage = "SMART атрибуты не считываются"
			if !isAdmin {
				smCheck.RootCause = "Windows запрещает прямой доступ к сырым секторам дисков \\\\.\\PhysicalDrive без прав Администратора."
				smCheck.Recommendation = "Запустите SysInfoGo от имени Администратора для снятия показаний SMART."
			} else {
				smCheck.RootCause = "Накопитель или контроллер RAID/NVMe не поддерживает стандартные команды ATA/NVMe SMART через smartctl."
				smCheck.Recommendation = "Проверьте режим работы контроллера в BIOS (AHCI/NVMe вместо proprietary RAID)."
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
	if netInfo == nil || len(netInfo.Interfaces) == 0 {
		report.Checks = append(report.Checks, CheckItem{
			Name:           "Сетевые интерфейсы",
			Status:         StatusWarn,
			ErrorMessage:   "Активные сетевые адаптеры не найдены",
			RootCause:      "Сетевые интерфейсы отключены или отсутствует драйвер сети.",
			Recommendation: "Включите сетевой адаптер в Диспетчере устройств.",
		})
		report.HasWarnings = true
		return report
	}

	report.Checks = append(report.Checks, CheckItem{
		Name:   "Сетевые адаптеры",
		Status: StatusOK,
		Value:  fmt.Sprintf("Активных интерфейсов: %d", len(netInfo.Interfaces)),
	})
	return report
}

func runMotherboardDiagnostics(ctx context.Context, isAdmin bool) ComponentReport {
	report := ComponentReport{ComponentName: "Материнская плата и BIOS"}

	mbInfo, _, _ := motherboard.Collect(ctx)
	mbCheck := CheckItem{Name: "Информация о материнской плате"}
	if mbInfo != nil && (mbInfo.Manufacturer != "" || mbInfo.Model != "") {
		mbCheck.Status = StatusOK
		mbCheck.Value = fmt.Sprintf("%s %s (BIOS: %s %s)", mbInfo.Manufacturer, mbInfo.Model, mbInfo.BiosVendor, mbInfo.BiosVersion)
	} else {
		mbCheck.Status = StatusWarn
		mbCheck.ErrorMessage = "Сведения о плате не прочитаны"
		mbCheck.RootCause = "WMI Win32_BaseBoard не возвращает данные DMI."
		mbCheck.Recommendation = "Обновите BIOS платы."
		report.HasWarnings = true
	}
	report.Checks = append(report.Checks, mbCheck)

	return report
}
