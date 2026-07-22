package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/user/sysinfogo/internal/locale"
	"github.com/user/sysinfogo/internal/network"
	"github.com/user/sysinfogo/internal/output"
	"github.com/user/sysinfogo/internal/processes"
	"github.com/user/sysinfogo/internal/render"
	"github.com/user/sysinfogo/internal/storage"
	"github.com/user/sysinfogo/internal/summary"
)

//go:embed static/*
var staticFS embed.FS

type Server struct {
	host      string
	port      string
	collect   func(ctx context.Context) (map[string]any, []output.Warning)
	dataMutex sync.RWMutex
	lastData  map[string]any

	bgNetHistory bool
	netHistory   map[string][]float64

	lastState *serverState
}

type serverState struct {
	Timestamp time.Time
	NetSent   map[string]uint64
	NetRecv   map[string]uint64
	DiskRead  map[string]uint64
	DiskWrite map[string]uint64
}

func NewServer(host string, port string, bgNetHistory bool, collect func(ctx context.Context) (map[string]any, []output.Warning)) *Server {
	return &Server{
		host:         host,
		port:         port,
		collect:      collect,
		bgNetHistory: bgNetHistory,
		netHistory:   make(map[string][]float64),
		lastState: &serverState{
			NetSent:   make(map[string]uint64),
			NetRecv:   make(map[string]uint64),
			DiskRead:  make(map[string]uint64),
			DiskWrite: make(map[string]uint64),
		},
	}
}

func (s *Server) Start(ctx context.Context, interval time.Duration) error {
	// Start collector loop
	go s.collectorLoop(ctx, interval)

	// Setup HTTP Handlers
	mux := http.NewServeMux()

	// Serve static files
	staticContent, err := fs.Sub(staticFS, "static")
	if err != nil {
		return err
	}
	mux.Handle("/", http.FileServer(http.FS(staticContent)))

	// API endpoints
	mux.HandleFunc("/api/info", s.handleAPI)
	mux.HandleFunc("/api/locale", s.handleLocale)
	mux.HandleFunc("/api/process/terminate", s.handleProcessKill)
	mux.HandleFunc("/api/smart", s.handleSmart)
	mux.HandleFunc("/api/network/history", s.handleNetworkHistory)
	mux.HandleFunc("/api/export/txt", s.handleExportTXT)
	mux.HandleFunc("/api/export/csv", s.handleExportCSV)
	mux.HandleFunc("/api/export/html", s.handleExportHTML)

	bindAddr := s.host + ":" + s.port
	if s.host == "" {
		bindAddr = ":" + s.port
	}

	server := &http.Server{
		Addr:    bindAddr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	displayHost := s.host
	if displayHost == "0.0.0.0" || displayHost == "" {
		displayHost = "localhost"
	}
	fmt.Printf("Web Dashboard is running at http://%s:%s\n", displayHost, s.port)
	if s.host == "127.0.0.1" {
		fmt.Println("(Режим: Чисто локальный доступ / Localhost only)")
	}
	fmt.Println("Press Ctrl+C to stop.")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) collectorLoop(ctx context.Context, interval time.Duration) {
	// Initial collection
	s.updateData(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.updateData(ctx)
		}
	}
}

func (s *Server) updateData(ctx context.Context) {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	data, warnings := s.collect(ctxTimeout)

	now := time.Now()
	if s.lastState.Timestamp.IsZero() {
		s.lastState.Timestamp = now
	} else {
		elapsed := now.Sub(s.lastState.Timestamp).Seconds()
		if elapsed > 0 {
			if netInfo, ok := data["network"].(*network.Info); ok {
				for i, iface := range netInfo.Interfaces {
					if prev, ok := s.lastState.NetRecv[iface.Name]; ok && iface.BytesRecv >= prev {
						netInfo.Interfaces[i].SpeedRecvMbps = float64(iface.BytesRecv-prev) * 8 / (1024 * 1024) / elapsed
					}
					if prev, ok := s.lastState.NetSent[iface.Name]; ok && iface.BytesSent >= prev {
						netInfo.Interfaces[i].SpeedSentMbps = float64(iface.BytesSent-prev) * 8 / (1024 * 1024) / elapsed
					}
					s.lastState.NetRecv[iface.Name] = iface.BytesRecv
					s.lastState.NetSent[iface.Name] = iface.BytesSent

					if s.bgNetHistory {
						val := netInfo.Interfaces[i].SpeedRecvMbps + netInfo.Interfaces[i].SpeedSentMbps
						hist := s.netHistory[iface.Name]
						hist = append(hist, val)
						if len(hist) > 50 {
							hist = hist[1:]
						}
						s.netHistory[iface.Name] = hist
					}
				}
			}
			if storageInfo, ok := data["storage"].(*storage.Info); ok {
				for i, disk := range storageInfo.Disks {
					id := fmt.Sprintf("%d_%s", disk.DiskNumber, disk.Model)
					if prev, ok := s.lastState.DiskRead[id]; ok && disk.ReadBytes >= prev {
						storageInfo.Disks[i].ReadMBps = float64(disk.ReadBytes-prev) / (1024 * 1024) / elapsed
					}
					if prev, ok := s.lastState.DiskWrite[id]; ok && disk.WriteBytes >= prev {
						storageInfo.Disks[i].WriteMBps = float64(disk.WriteBytes-prev) / (1024 * 1024) / elapsed
					}
					s.lastState.DiskRead[id] = disk.ReadBytes
					s.lastState.DiskWrite[id] = disk.WriteBytes
				}
			}
		}
		s.lastState.Timestamp = now
	}

	data["warnings"] = warnings
	data["timestamp"] = now.Format("15:04:05")

	s.dataMutex.Lock()
	s.lastData = data
	s.dataMutex.Unlock()
}

func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
	s.dataMutex.RLock()
	data := s.lastData
	s.dataMutex.RUnlock()

	if data == nil {
		http.Error(w, "Data not ready", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *Server) handleLocale(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(locale.GetDictionary())
}

func (s *Server) getAggrData() *output.AggregatedData {
	s.dataMutex.RLock()
	data := s.lastData
	s.dataMutex.RUnlock()

	if data == nil {
		return nil
	}

	warns := []output.Warning{}
	if w, ok := data["warnings"].([]output.Warning); ok {
		warns = w
	}

	host := "unknown"
	if sum, ok := data["summary"].(*summary.Info); ok {
		host = sum.Hostname
	}

	return &output.AggregatedData{
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Hostname:     host,
		OS:           fmt.Sprintf("%s %s", runtime.GOOS, runtime.GOARCH),
		IsAdmin:      false,
		SectionOrder: []string{"summary", "cpu", "memory", "storage", "gpu", "network", "motherboard", "processes", "battery"},
		Sections:     data,
		Warnings:     warns,
	}
}

func (s *Server) handleExportTXT(w http.ResponseWriter, r *http.Request) {
	aggr := s.getAggrData()
	if aggr == nil {
		http.Error(w, "Data not ready", http.StatusServiceUnavailable)
		return
	}
	formatter := render.NewTextFormatter(false, true, "auto", true) // verbose=false, noColor=true, allProcesses=true
	txt := formatter.Format(aggr)

	w.Header().Set("Content-Disposition", "attachment; filename=sysinfo_report.txt")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(txt))
}

func (s *Server) handleExportCSV(w http.ResponseWriter, r *http.Request) {
	// Simple CSV generation
	aggr := s.getAggrData()
	if aggr == nil {
		http.Error(w, "Data not ready", http.StatusServiceUnavailable)
		return
	}
	// For CSV, we can just use JSON to Map and flatten, or just return JSON and let user convert it.
	// But let's create a basic CSV for top CPU processes.
	w.Header().Set("Content-Disposition", "attachment; filename=sysinfo_processes.csv")
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Write([]byte("PID,Name,CPU_Pct,Mem_Pct,User\n"))

	if proc, ok := aggr.Sections["processes"].(*processes.Info); ok {
		for _, p := range proc.Processes {
			w.Write([]byte(fmt.Sprintf("%d,\"%s\",%.1f,%.1f,\"%s\"\n", p.PID, p.Name, p.CPU, p.Memory, p.User)))
		}
	}
}

func (s *Server) handleExportHTML(w http.ResponseWriter, r *http.Request) {
	aggr := s.getAggrData()
	if aggr == nil {
		http.Error(w, "Data not ready", http.StatusServiceUnavailable)
		return
	}

	// Create temp file for GenerateHTML
	tmpFile := filepath.Join(os.TempDir(), "sysinfo_tmp.html")
	if err := GenerateHTML(aggr.Sections, tmpFile); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpFile)

	content, err := os.ReadFile(tmpFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=sysinfo_report.html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(content)
}

func (s *Server) handleProcessKill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pidStr := r.URL.Query().Get("pid")
	if pidStr == "" {
		http.Error(w, "pid is required", http.StatusBadRequest)
		return
	}
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		http.Error(w, "invalid pid", http.StatusBadRequest)
		return
	}
	if pid == os.Getpid() {
		http.Error(w, "Cannot kill sysinfogo process", http.StatusForbidden)
		return
	}
	p, err := os.FindProcess(pid)
	if err == nil {
		err = p.Kill()
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleSmart(w http.ResponseWriter, r *http.Request) {
	device := r.URL.Query().Get("device")
	if device == "" {
		http.Error(w, "device is required", http.StatusBadRequest)
		return
	}

	devArg := device
	if runtime.GOOS == "windows" {
		devArg = "/dev/" + device
	}

	out, status, err := storage.ExecSmartctl(r.Context(), "-a", devArg)
	if err != nil && len(out) == 0 {
		http.Error(w, "smartctl failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if strings.Contains(status, "системная версия") {
		w.Write([]byte("[" + status + "]\n\n"))
	}
	w.Write(out)
}

func (s *Server) handleNetworkHistory(w http.ResponseWriter, r *http.Request) {
	s.dataMutex.RLock()
	defer s.dataMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	if !s.bgNetHistory {
		w.Write([]byte("{}"))
		return
	}
	json.NewEncoder(w).Encode(s.netHistory)
}
