// Command web_export demonstrates streaming an exported xlsx file directly
// to an HTTP response instead of a local file.
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
)

// exportHandler demonstrates exporting Excel file directly to browser
// File will automatically download to user's computer.
func exportHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context() // Get context from request

	// Prepare data (could come from a database, API, etc.)
	data := [][]any{
		{"Name", "Age", "Email", "Department"},
		{"John Doe", 30, "john@example.com", "Engineering"},
		{"Jane Smith", 25, "jane@example.com", "Marketing"},
		{"Bob Johnson", 35, "bob@example.com", "Sales"},
		{"Alice Williams", 28, "alice@example.com", "HR"},
	}

	// Create exporter with headings
	exp := exporter.NewExporter(data, excel.WithHeadings(true))

	// These headers are required for the browser to trigger a download
	// instead of trying to render the response inline.
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=employees.xlsx")

	// Stream the export directly to the response writer.
	if err := exp.ExportToWriter(ctx, w); err != nil {
		http.Error(w, fmt.Sprintf("Export failed: %v", err), http.StatusInternalServerError)
		return
	}
}

func main() {
	// Setup HTTP server
	http.HandleFunc("/export", exportHandler)

	fmt.Println("🚀 Server running on http://localhost:8080")
	fmt.Println("📥 Visit http://localhost:8080/export to download Excel file")
	fmt.Println("   File will automatically download to your Downloads folder")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server failed:", err)
	}
}
