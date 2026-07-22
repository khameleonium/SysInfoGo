# Инструкция по сборке SysInfoGo и обновлению встроенных бинарников smartctl

Проект **SysInfoGo** использует встроенные автоблоки `smartctl` (из состава открытого проекта [smartmontools](https://github.com/smartmontools/smartmontools)), запекаемые в итоговый исполняемый файл с помощью механизма Go `//go:embed`.

Это позволяет утилите работать полностью из коробки ("Zero Dependencies") без необходимости для конечного пользователя устанавливать какие-либо сторонние пакеты.

---

## 📂 Структура встроенных бинарников

Все бинарные файлы `smartctl` для разных операционных систем располагаются в директории:
```text
internal/storage/bin/
```

Файлы в этой директории должны иметь следующие точные имена:

| Имя файла | Операционная система | Описание |
| :--- | :--- | :--- |
| `smartctl_windows.exe` | **Windows** (x86_64 / amd64) | Официальный `smartctl.exe` для Windows |
| `smartctl_linux` | **Linux** (x86_64 / amd64) | Скомпилированный бинарный файл `smartctl` под Linux |
| `smartctl_darwin` | **macOS** (Apple Silicon / Intel) | Скомпилированный бинарный файл `smartctl` под macOS |

---

## 🛠 Замена и обновление бинарников smartctl

Если вы хотите обновить версию `smartctl` до более свежей версии или собрать под свою специфическую архитектуру:

### 1. Получение бинарников smartctl
Скачайте скомпилированные утилиты или исходный код с официальной страницы релизов `smartmontools`:
👉 **[GitHub Releases smartmontools](https://github.com/smartmontools/smartmontools/releases)**

* **Для Windows:** Скачайте `smartmontools-X.Y.win32-setup.exe`. Распакуйте установочный файл (например, через [7-Zip](https://www.7-zip.org/) или запустив установщик с параметром `/S /D=C:\temp\smart`) и скопируйте `smartctl.exe` из папки `bin/` в:
  `internal/storage/bin/smartctl_windows.exe`

* **Для Linux:** Возьмите скомпилированный бинарник `smartctl` из системы (`/usr/sbin/smartctl`) или скачайте официальный статический бинарник и скопируйте в:
  `internal/storage/bin/smartctl_linux`

* **Для macOS:** Скопируйте файл `smartctl` (установленный, например, через `brew install smartmontools`) в:
  `internal/storage/bin/smartctl_darwin`

---

### 2. Указание версии в коде
Если вы заменили бинарник на новую версию (например, `7.6`), откройте файл:
[`internal/storage/embedded.go`](file:///D:/Project/SysInfoGo/internal/storage/embedded.go)

И обновите константу `EmbeddedSmartctlVersion`:
```go
// EmbeddedSmartctlVersion is the version of the bundled smartctl tool.
const EmbeddedSmartctlVersion = "7.6"
```

Это необходимо для системы умного сравнения версий: если у пользователя в системе будет найден системный `smartctl` более свежей версии, SysInfoGo автоматически использует системный вместо встроенного.

---

## 🏗 Сборка исполняемых файлов

Для сборки утилиты необходим установленный компилятор **Go (1.21+)**.

### Локальная сборка под текущую ОС:
```bash
go build -ldflags "-s -w" -o sysinfogo.exe ./cmd/sysinfogo
```

### Кросс-компиляция для всех платформ:

#### Windows (amd64):
```bash
env GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o Releases/sysinfogo_windows_amd64.exe ./cmd/sysinfogo
```

#### Linux (amd64):
```bash
env GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o Releases/sysinfogo_linux ./cmd/sysinfogo
```

#### macOS (Apple Silicon arm64 / Intel amd64):
```bash
# Для Apple Silicon (M1/M2/M3/M4):
env GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o Releases/sysinfogo_darwin_arm64 ./cmd/sysinfogo

# Для Intel Mac:
env GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o Releases/sysinfogo_darwin_amd64 ./cmd/sysinfogo
```

*(На Windows PowerShell переключение ОС выполняется через `$env:GOOS="linux"; $env:GOARCH="amd64"`, после чего запускается `go build`)*.
