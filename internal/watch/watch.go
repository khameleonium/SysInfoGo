package watch

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/user/sysinfogo/internal/network"
	"github.com/user/sysinfogo/internal/output"
	"github.com/user/sysinfogo/internal/render"
	"github.com/user/sysinfogo/internal/storage"
)

type Watch struct {
	interval     time.Duration
	jsonMode     bool
	noColor      bool
	verbose      bool
	units        string
	logger       *CSVLogger
	sectionOrder []string
	lastState    *watchState
	allProcesses bool
}

type watchState struct {
	Timestamp time.Time
	NetSent   map[string]uint64
	NetRecv   map[string]uint64
	DiskRead  map[string]uint64
	DiskWrite map[string]uint64
}

func New(interval time.Duration, jsonMode, noColor, verbose bool, units, logPath string, logAppend bool, sectionOrder []string, allProcesses bool) (*Watch, error) {
	w := &Watch{interval: interval, jsonMode: jsonMode, noColor: noColor, verbose: verbose, units: units, sectionOrder: sectionOrder, allProcesses: allProcesses}

	if logPath != "" {
		l, err := NewLogger(logPath, logAppend)
		if err != nil {
			return nil, fmt.Errorf("log file %s: %w", logPath, err)
		}
		w.logger = l
	}

	return w, nil
}

type CollectFn func(ctx context.Context) (map[string]any, []output.Warning)

func (w *Watch) Run(ctx context.Context, collect CollectFn) {
	defer w.cleanup()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	w.render(ctx, collect)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			return
		case <-ticker.C:
			w.render(ctx, collect)
		}
	}
}

func (w *Watch) render(parentCtx context.Context, collect CollectFn) {
	timeout := w.interval - 100*time.Millisecond
	if timeout <= 0 {
		timeout = w.interval / 2
	}
	ctx, cancel := context.WithTimeout(parentCtx, timeout)
	defer cancel()

	data, warnings := collect(ctx)

	if !w.jsonMode && !w.noColor {
		clearScreen()
		fmt.Print("\033[?25l")
	}

	now := time.Now()

	if w.lastState == nil {
		w.lastState = &watchState{
			Timestamp: now,
			NetSent:   make(map[string]uint64),
			NetRecv:   make(map[string]uint64),
			DiskRead:  make(map[string]uint64),
			DiskWrite: make(map[string]uint64),
		}
	} else {
		elapsed := now.Sub(w.lastState.Timestamp).Seconds()
		if elapsed > 0 {
			if netInfo, ok := data["network"].(*network.Info); ok {
				for i, iface := range netInfo.Interfaces {
					if prev, ok := w.lastState.NetRecv[iface.Name]; ok {
						if iface.BytesRecv >= prev {
							netInfo.Interfaces[i].SpeedRecvMbps = float64(iface.BytesRecv-prev) * 8 / (1024 * 1024) / elapsed
						}
					}
					if prev, ok := w.lastState.NetSent[iface.Name]; ok {
						if iface.BytesSent >= prev {
							netInfo.Interfaces[i].SpeedSentMbps = float64(iface.BytesSent-prev) * 8 / (1024 * 1024) / elapsed
						}
					}
					w.lastState.NetRecv[iface.Name] = iface.BytesRecv
					w.lastState.NetSent[iface.Name] = iface.BytesSent
				}
			}
			if storageInfo, ok := data["storage"].(*storage.Info); ok {
				for i, disk := range storageInfo.Disks {
					id := fmt.Sprintf("%d_%s", disk.DiskNumber, disk.Model)
					if prev, ok := w.lastState.DiskRead[id]; ok {
						if disk.ReadBytes >= prev {
							storageInfo.Disks[i].ReadMBps = float64(disk.ReadBytes-prev) / (1024 * 1024) / elapsed
						}
					}
					if prev, ok := w.lastState.DiskWrite[id]; ok {
						if disk.WriteBytes >= prev {
							storageInfo.Disks[i].WriteMBps = float64(disk.WriteBytes-prev) / (1024 * 1024) / elapsed
						}
					}
					w.lastState.DiskRead[id] = disk.ReadBytes
					w.lastState.DiskWrite[id] = disk.WriteBytes
				}
			}
		}
		w.lastState.Timestamp = now
	}

	if w.logger != nil {
		if err := w.logger.WriteRow(now, data); err != nil {
			fmt.Fprintf(os.Stderr, "log error: %v\n", err)
		}
	}

	aggr := &output.AggregatedData{
		Timestamp:    now.UTC().Format(time.RFC3339),
		SectionOrder: w.sectionOrder,
		Sections:     data,
		Warnings:     warnings,
	}

	var formatter output.Formatter
	if w.jsonMode {
		formatter = output.NewJSONFormatter(false)
	} else {
		formatter = render.NewTextFormatter(!w.noColor, w.verbose, w.units, w.allProcesses)
	}

	fmt.Print(formatter.Format(aggr))
}

func (w *Watch) cleanup() {
	if !w.jsonMode && !w.noColor {
		fmt.Print("\033[?25h")
	}

	if w.logger != nil {
		w.logger.Close()
	}
}
