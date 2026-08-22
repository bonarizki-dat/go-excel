// Command events demonstrates registering lifecycle hooks (BeforeExport,
// AfterExport, BeforeRowImport, OnError, and so on) on exporters and
// importers.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/bonarizki-dat/go-excel/excel"
	"github.com/bonarizki-dat/go-excel/excel/exporter"
	"github.com/bonarizki-dat/go-excel/excel/importer"
)

func main() {
	ctx := context.Background()

	// Example 1: Export events
	fmt.Println("=== Example 1: Export Events ===")
	if err := demonstrateExportEvents(ctx); err != nil {
		log.Fatal(err)
	}
	fmt.Println()

	// Example 2: Import events
	fmt.Println("=== Example 2: Import Events ===")
	if err := demonstrateImportEvents(ctx); err != nil {
		log.Fatal(err)
	}
	fmt.Println()

	// Example 3: Multiple hooks per event
	fmt.Println("=== Example 3: Multiple Hooks Per Event ===")
	if err := demonstrateMultipleHooks(ctx); err != nil {
		log.Fatal(err)
	}
	fmt.Println()

	// Example 4: Error handling in hooks
	fmt.Println("=== Example 4: Error Handling in Hooks ===")
	if err := demonstrateErrorHandling(ctx); err != nil {
		log.Fatal(err)
	}
}

// demonstrateExportEvents shows how to use export events.
func demonstrateExportEvents(ctx context.Context) error {
	// Prepare data
	data := [][]any{
		{"Name", "Age", "Email"},
		{"John Doe", 30, "john@example.com"},
		{"Jane Smith", 25, "jane@example.com"},
		{"Bob Johnson", 35, "bob@example.com"},
	}

	// Create exporter
	exp := exporter.NewExporter(data, excel.WithHeadings(true))

	// Track events
	eventsCalled := make(map[excel.EventType]int)

	// Register BeforeExport hook
	exp.On(excel.BeforeExport, func(_ context.Context, _ *excel.Event) error {
		eventsCalled[excel.BeforeExport]++
		fmt.Println("  🔔 BeforeExport: Starting export...")
		return nil
	})

	// Register AfterExport hook
	exp.On(excel.AfterExport, func(_ context.Context, _ *excel.Event) error {
		eventsCalled[excel.AfterExport]++
		fmt.Println("  🔔 AfterExport: Export completed!")
		return nil
	})

	// Register BeforeRowExport hook
	exp.On(excel.BeforeRowExport, func(_ context.Context, e *excel.Event) error {
		eventsCalled[excel.BeforeRowExport]++
		row := e.Data.([]any)
		fmt.Printf("  🔔 BeforeRowExport: Processing row: %v\n", row)
		return nil
	})

	// Register AfterRowExport hook
	exp.On(excel.AfterRowExport, func(_ context.Context, e *excel.Event) error {
		eventsCalled[excel.AfterRowExport]++
		row := e.Data.([]any)
		fmt.Printf("  🔔 AfterRowExport: Completed row: %v\n", row)
		return nil
	})

	// Perform export
	if err := exp.Export(ctx); err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	// Save to file
	filename := "events_demo.xlsx"
	if err := exp.GetFile().SaveAs(filename); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}
	defer func() { _ = os.Remove(filename) }()

	// Print summary
	fmt.Printf("  ✅ Events called:\n")
	fmt.Printf("    - BeforeExport: %d\n", eventsCalled[excel.BeforeExport])
	fmt.Printf("    - AfterExport: %d\n", eventsCalled[excel.AfterExport])
	fmt.Printf("    - BeforeRowExport: %d\n", eventsCalled[excel.BeforeRowExport])
	fmt.Printf("    - AfterRowExport: %d\n", eventsCalled[excel.AfterRowExport])

	return nil
}

// demonstrateImportEvents shows how to use import events.
func demonstrateImportEvents(ctx context.Context) error {
	// First, create a file to import
	data := [][]any{
		{"Name", "Age"},
		{"Alice", 30},
		{"Bob", 25},
	}
	exp := exporter.NewExporter(data, excel.WithHeadings(true))
	if err := exp.Export(ctx); err != nil {
		return fmt.Errorf("failed to create test file: %w", err)
	}
	filename := "import_events_demo.xlsx"
	if err := exp.GetFile().SaveAs(filename); err != nil {
		return fmt.Errorf("failed to save test file: %w", err)
	}
	defer func() { _ = os.Remove(filename) }()

	// Create importer
	imp := importer.NewImporter(excel.WithHeadings(true))

	// Track events
	eventsCalled := make(map[excel.EventType]int)

	// Register BeforeImport hook
	imp.On(excel.BeforeImport, func(_ context.Context, _ *excel.Event) error {
		eventsCalled[excel.BeforeImport]++
		fmt.Println("  🔔 BeforeImport: Starting import...")
		return nil
	})

	// Register AfterImport hook
	imp.On(excel.AfterImport, func(_ context.Context, _ *excel.Event) error {
		eventsCalled[excel.AfterImport]++
		fmt.Println("  🔔 AfterImport: Import completed!")
		return nil
	})

	// Register BeforeRowImport hook
	imp.On(excel.BeforeRowImport, func(_ context.Context, e *excel.Event) error {
		eventsCalled[excel.BeforeRowImport]++
		row := e.Data.([]any)
		fmt.Printf("  🔔 BeforeRowImport: Processing row: %v\n", row)
		return nil
	})

	// Register AfterRowImport hook
	imp.On(excel.AfterRowImport, func(_ context.Context, e *excel.Event) error {
		eventsCalled[excel.AfterRowImport]++
		row := e.Data.([]any)
		fmt.Printf("  🔔 AfterRowImport: Completed row: %v\n", row)
		return nil
	})

	// Perform import
	err := imp.ImportFromFile(ctx, filename)
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	// Print summary
	fmt.Printf("  ✅ Events called:\n")
	fmt.Printf("    - BeforeImport: %d\n", eventsCalled[excel.BeforeImport])
	fmt.Printf("    - AfterImport: %d\n", eventsCalled[excel.AfterImport])
	fmt.Printf("    - BeforeRowImport: %d\n", eventsCalled[excel.BeforeRowImport])
	fmt.Printf("    - AfterRowImport: %d\n", eventsCalled[excel.AfterRowImport])

	return nil
}

// demonstrateMultipleHooks shows that multiple hooks can be registered for the same event.
func demonstrateMultipleHooks(ctx context.Context) error {
	data := [][]any{
		{"Name", "Value"},
		{"Item 1", 100},
		{"Item 2", 200},
	}

	exp := exporter.NewExporter(data, excel.WithHeadings(true))

	hook1Called := false
	hook2Called := false
	hook3Called := false

	// Register multiple hooks for BeforeExport
	exp.On(excel.BeforeExport, func(_ context.Context, _ *excel.Event) error {
		hook1Called = true
		fmt.Println("  🔔 Hook 1: First BeforeExport hook")
		return nil
	})

	exp.On(excel.BeforeExport, func(_ context.Context, _ *excel.Event) error {
		hook2Called = true
		fmt.Println("  🔔 Hook 2: Second BeforeExport hook")
		return nil
	})

	exp.On(excel.BeforeExport, func(_ context.Context, _ *excel.Event) error {
		hook3Called = true
		fmt.Println("  🔔 Hook 3: Third BeforeExport hook")
		return nil
	})

	// Perform export
	if err := exp.Export(ctx); err != nil {
		return fmt.Errorf("export failed: %w", err)
	}

	// Verify all hooks were called
	if !hook1Called || !hook2Called || !hook3Called {
		return fmt.Errorf("not all hooks were called")
	}

	fmt.Println("  ✅ All hooks executed in order")

	return nil
}

// demonstrateErrorHandling shows how errors in hooks can cancel operations.
func demonstrateErrorHandling(ctx context.Context) error {
	data := [][]any{
		{"Name", "Value"},
		{"Item 1", 100},
		{"Item 2", 200},
	}

	exp := exporter.NewExporter(data, excel.WithHeadings(true))

	// Register error hook
	exp.On(excel.OnError, func(_ context.Context, e *excel.Event) error {
		fmt.Printf("  ⚠️  OnError: %v\n", e.Error)
		return nil // Return nil to handle error, don't propagate
	})

	// Register a hook that returns an error to cancel operation
	exp.On(excel.BeforeRowExport, func(_ context.Context, e *excel.Event) error {
		row := e.Data.([]any)
		// Cancel on specific condition
		if len(row) > 0 && row[0] == "Item 2" {
			return fmt.Errorf("canceling export on Item 2")
		}
		return nil
	})

	// Perform export - should fail due to hook error
	err := exp.Export(ctx)
	if err == nil {
		return fmt.Errorf("expected export to fail but it succeeded")
	}

	fmt.Printf("  ✅ Export correctly canceled: %v\n", err)

	return nil
}
