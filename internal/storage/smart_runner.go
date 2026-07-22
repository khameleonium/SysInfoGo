package storage

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	resolvedBinary string
	resolvedStatus string
	resolveOnce    sync.Once
)

// parseVersion extracts version tuple e.g. "7.5" -> 7.5 float from `smartctl 7.5 ...`
func parseVersionStr(raw string) float64 {
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "smartctl ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				verStr := parts[1]
				// Handle versions like 7.4 or 7.5 or 7.4.1
				subParts := strings.Split(verStr, ".")
				if len(subParts) >= 2 {
					verStr = subParts[0] + "." + subParts[1]
				}
				if v, err := strconv.ParseFloat(verStr, 64); err == nil {
					return v
				}
			}
		}
	}
	return 0.0
}

func getEmbeddedBinaryBytes() []byte {
	switch runtime.GOOS {
	case "windows":
		return smartctlWindows
	case "linux":
		return smartctlLinux
	case "darwin":
		return smartctlDarwin
	default:
		return nil
	}
}

func getEmbeddedBinaryName() string {
	if runtime.GOOS == "windows" {
		return "sysinfogo_smartctl.exe"
	}
	return "sysinfogo_smartctl"
}

func getSmartctlPath(ctx context.Context) (string, string) {
	resolveOnce.Do(func() {
		embVersion := parseVersionStr("smartctl " + EmbeddedSmartctlVersion)

		// 1. Check system smartctl
		sysPath, err := exec.LookPath("smartctl")
		if err == nil && sysPath != "" {
			cmdCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			out, vErr := exec.CommandContext(cmdCtx, sysPath, "-V").CombinedOutput()
			cancel()
			if vErr == nil {
				sysVer := parseVersionStr(string(out))
				if sysVer > embVersion {
					resolvedBinary = sysPath
					resolvedStatus = fmt.Sprintf("Обнаружена более свежая системная версия smartctl (v%.1f). Используется: %s", sysVer, sysPath)
					return
				}
			}
		}

		// 2. Fallback to embedded smartctl
		embData := getEmbeddedBinaryBytes()
		if len(embData) > 0 {
			tmpDir := os.TempDir()
			binName := getEmbeddedBinaryName()
			targetPath := filepath.Join(tmpDir, binName)

			// Write embedded file if not exists or size differs
			stat, statErr := os.Stat(targetPath)
			if statErr != nil || stat.Size() != int64(len(embData)) {
				_ = os.WriteFile(targetPath, embData, 0755)
			}

			resolvedBinary = targetPath
			resolvedStatus = fmt.Sprintf("Используется встроенный модуль smartctl v%s", EmbeddedSmartctlVersion)
			return
		}

		// Fallback to system smartctl if embedded is empty/unavailable
		if sysPath != "" {
			resolvedBinary = sysPath
			resolvedStatus = "Используется системный smartctl"
			return
		}

		resolvedBinary = "smartctl"
		resolvedStatus = "smartctl не найден"
	})

	return resolvedBinary, resolvedStatus
}

// ExecSmartctl runs smartctl using the best resolved binary (system or embedded).
func ExecSmartctl(ctx context.Context, args ...string) ([]byte, string, error) {
	binPath, status := getSmartctlPath(ctx)
	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, binPath, args...)
	out, err := cmd.CombinedOutput()
	return out, status, err
}
