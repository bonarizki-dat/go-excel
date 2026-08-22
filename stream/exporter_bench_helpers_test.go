package stream

import (
	"fmt"
)

// generateHeadings returns numCols column heading strings for
// benchmarks, named "Column1", "Column2", and so on.
func generateHeadings(numCols int) []string {
	headings := make([]string, numCols)
	for i := range numCols {
		headings[i] = fmt.Sprintf("Column%d", i+1)
	}
	return headings
}

// generateRow returns a numCols-length benchmark row for rowNum, with
// values cycling through string, int, float64, and bool across
// columns to resemble a realistic mixed-type dataset.
func generateRow(rowNum, numCols int) []any {
	row := make([]any, numCols)
	for col := range numCols {
		switch col % 4 {
		case 0:
			row[col] = fmt.Sprintf("Row%d_Col%d", rowNum, col)
		case 1:
			row[col] = rowNum*numCols + col
		case 2:
			row[col] = float64(rowNum) + float64(col)/100.0
		case 3:
			row[col] = rowNum%2 == 0
		}
	}
	return row
}
