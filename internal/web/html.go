package web

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// GenerateHTML creates a standalone HTML file containing all styles, scripts, and data.
func GenerateHTML(data map[string]any, outFilename string) error {
	indexBytes, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		return fmt.Errorf("failed to read index.html: %w", err)
	}

	cssBytes, err := staticFS.ReadFile("static/style.css")
	if err != nil {
		return fmt.Errorf("failed to read style.css: %w", err)
	}

	jsBytes, err := staticFS.ReadFile("static/app.js")
	if err != nil {
		return fmt.Errorf("failed to read app.js: %w", err)
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	htmlContent := string(indexBytes)

	// Inject CSS
	cssTag := fmt.Sprintf("<style>\n%s\n</style>", string(cssBytes))
	htmlContent = strings.Replace(htmlContent, `<link rel="stylesheet" href="style.css">`, cssTag, 1)

	// Inject JS
	jsTag := fmt.Sprintf("<script>\n%s\n</script>", string(jsBytes))
	htmlContent = strings.Replace(htmlContent, `<script src="app.js"></script>`, jsTag, 1)

	// Inject JSON Data
	jsonTag := fmt.Sprintf(`<script id="injected-data" type="application/json">%s</script>`, string(jsonData))
	htmlContent = strings.Replace(htmlContent, `<script id="injected-data" type="application/json"></script>`, jsonTag, 1)

	return os.WriteFile(outFilename, []byte(htmlContent), 0644)
}
