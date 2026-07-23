package diagnostic

import (
	"context"
	"os"
	"runtime"
	"time"
)

type Status string

const (
	StatusOK   Status = "OK"
	StatusWarn Status = "WARN"
	StatusFail Status = "FAIL"
)

type CheckItem struct {
	Name           string `json:"name"`
	Status         Status `json:"status"`
	Value          string `json:"value,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
	RootCause      string `json:"root_cause,omitempty"`
	Recommendation string `json:"recommendation,omitempty"`
}

type ComponentReport struct {
	ComponentName string      `json:"component_name"`
	Checks        []CheckItem `json:"checks"`
	HasWarnings   bool        `json:"has_warnings"`
	HasErrors     bool        `json:"has_errors"`
}

type DiagnosticResult struct {
	OS          string            `json:"os"`
	Kernel      string            `json:"kernel"`
	Hostname    string            `json:"hostname"`
	IsAdmin     bool              `json:"is_admin"`
	Timestamp   time.Time         `json:"timestamp"`
	Reports     []ComponentReport `json:"reports"`
	SummaryText string            `json:"summary_text"`
}

func Run(ctx context.Context) *DiagnosticResult {
	hostname, _ := os.Hostname()
	result := &DiagnosticResult{
		OS:        runtime.GOOS,
		Kernel:    runtime.GOARCH,
		Hostname:  hostname,
		Timestamp: time.Now(),
	}

	result.IsAdmin = checkAdminRights()

	var reports []ComponentReport
	reports = append(reports, runEnvDiagnostics(ctx, result.IsAdmin))
	reports = append(reports, runCPUDiagnostics(ctx, result.IsAdmin))
	reports = append(reports, runGPUDiagnostics(ctx, result.IsAdmin))
	reports = append(reports, runRAMDiagnostics(ctx, result.IsAdmin))
	reports = append(reports, runStorageDiagnostics(ctx, result.IsAdmin))
	reports = append(reports, runNetworkDiagnostics(ctx, result.IsAdmin))
	reports = append(reports, runMotherboardDiagnostics(ctx, result.IsAdmin))

	result.Reports = reports
	return result
}
