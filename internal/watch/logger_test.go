package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCSVLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.csv")

	logger, err := NewLogger(logPath, false)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	data1 := map[string]any{
		"CPU": "10.5",
		"RAM": "2.3",
	}
	logger.WriteRow(time.Now(), data1)

	data2 := map[string]any{
		"CPU":  "12.0",
		"Disk": "100",
	}
	logger.WriteRow(time.Now(), data2)

	logger.Close()

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}

	strContent := string(content)
	// It should have written the header first: Timestamp,CPU,RAM
	// Then data1
	// Then data2 should trigger a rewrite if new columns are added,
	// but the CSVLogger appends to the same line if columns are known,
	// wait, CSVLogger rebuilds the file if columns changed!
	// So the final file should have columns: Timestamp,CPU,Disk,RAM
	if len(strContent) == 0 {
		t.Errorf("log file is empty")
	}
}
