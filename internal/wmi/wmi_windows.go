package wmi

import (
	"bufio"
	"context"
	"os/exec"
	"strings"
	"sync"
)

var (
	cacheMutex sync.RWMutex
	cacheMap   = make(map[string]string)
)

func isDynamicQuery(args []string) bool {
	for _, arg := range args {
		argLower := strings.ToLower(arg)
		if strings.Contains(argLower, "battery") || strings.Contains(argLower, "thermal") || strings.Contains(argLower, "temperature") {
			return true
		}
	}
	return false
}

// QueryList executes wmic with the given args and /format:list, returning the raw output string.
func QueryList(ctx context.Context, args ...string) string {
	key := strings.Join(args, " ")
	isDynamic := isDynamicQuery(args)

	if !isDynamic {
		cacheMutex.RLock()
		if val, ok := cacheMap[key]; ok {
			cacheMutex.RUnlock()
			return val
		}
		cacheMutex.RUnlock()
	}

	cmd := exec.CommandContext(ctx, "wmic", args...)
	cmd.Args = append(cmd.Args, "/format:list")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	res := string(out)

	if !isDynamic && res != "" {
		cacheMutex.Lock()
		cacheMap[key] = res
		cacheMutex.Unlock()
	}

	return res
}

// ExtractTag extracts the value for a specific tag (e.g. "Name=") from a WMI list output line.
func ExtractTag(line, prefix string) string {
	idx := strings.Index(line, prefix)
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(line[idx+len(prefix):])
}

// ParseList parses a block of /format:list output into a map of maps or list of maps.
func ParseList(raw string) []map[string]string {
	var results []map[string]string
	var current map[string]string

	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			if len(current) > 0 {
				results = append(results, current)
				current = nil
			}
			continue
		}
		if current == nil {
			current = make(map[string]string)
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			current[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	if len(current) > 0 {
		results = append(results, current)
	}
	return results
}
