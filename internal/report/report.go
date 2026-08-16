// Package report lays numbers out for a terminal.
//
// A table of numbers is only readable if the columns line up and the numbers are
// printed with a fixed number of significant digits, so that the eye can compare
// magnitudes down a column instead of parsing each entry. Everything here is about
// that, and nothing here knows what the numbers mean.
package report

import (
	"fmt"
	"math"
	"strings"
)

// Alignment says how a column is padded.
type Alignment int

// The alignments.
const (
	AlignLeft Alignment = iota
	AlignRight
)

// Table is a set of rows laid out in aligned columns.
type Table struct {
	Header     []string
	Rows       [][]string
	Alignments []Alignment
	Gap        int
}

// AddRow appends one row.
func (t *Table) AddRow(cells ...string) {
	t.Rows = append(t.Rows, cells)
}

// widths returns the width of every column.
func (t Table) widths() []int {
	columns := len(t.Header)
	for _, row := range t.Rows {
		if len(row) > columns {
			columns = len(row)
		}
	}
	out := make([]int, columns)
	for index, cell := range t.Header {
		if length := len([]rune(cell)); length > out[index] {
			out[index] = length
		}
	}
	for _, row := range t.Rows {
		for index, cell := range row {
			if length := len([]rune(cell)); length > out[index] {
				out[index] = length
			}
		}
	}
	return out
}

// alignment returns the alignment of one column, defaulting to the left.
func (t Table) alignment(index int) Alignment {
	if index < len(t.Alignments) {
		return t.Alignments[index]
	}
	return AlignLeft
}

// pad fits a cell to a width on the requested side.
func pad(cell string, width int, alignment Alignment) string {
	missing := width - len([]rune(cell))
	if missing <= 0 {
		return cell
	}
	if alignment == AlignRight {
		return strings.Repeat(" ", missing) + cell
	}
	return cell + strings.Repeat(" ", missing)
}

// Render lays the table out, underlining the header and trimming the padding at the end
// of every line.
func (t Table) Render() string {
	gap := t.Gap
	if gap <= 0 {
		gap = 2
	}
	widths := t.widths()
	separator := strings.Repeat(" ", gap)
	var builder strings.Builder
	writeRow := func(cells []string) {
		parts := make([]string, 0, len(widths))
		for index := range widths {
			cell := ""
			if index < len(cells) {
				cell = cells[index]
			}
			parts = append(parts, pad(cell, widths[index], t.alignment(index)))
		}
		builder.WriteString(strings.TrimRight(strings.Join(parts, separator), " "))
		builder.WriteString("\n")
	}
	if len(t.Header) > 0 {
		writeRow(t.Header)
		rule := make([]string, len(widths))
		for index, width := range widths {
			rule[index] = strings.Repeat("-", width)
		}
		writeRow(rule)
	}
	for _, row := range t.Rows {
		writeRow(row)
	}
	return builder.String()
}

// Float renders a number with a fixed number of significant digits.
func Float(value float64) string {
	switch {
	case value == 0:
		return "0"
	case math.IsInf(value, 1):
		return "+inf"
	case math.IsInf(value, -1):
		return "-inf"
	case math.IsNaN(value):
		return "nan"
	}
	return fmt.Sprintf("%.6g", value)
}

// Sci renders a number in scientific notation, which is what an error column wants.
func Sci(value float64) string {
	if value == 0 {
		return "0"
	}
	if math.IsNaN(value) {
		return "nan"
	}
	return fmt.Sprintf("%.3e", value)
}

// Bool renders a flag as yes or no.
func Bool(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

// Int renders a whole number.
func Int(value int) string { return fmt.Sprintf("%d", value) }

// Ints renders a list of whole numbers.
func Ints(values []int) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%d", value))
	}
	return strings.Join(parts, " ")
}

// Bar renders a count as a row of blocks, scaled so that the largest count fills the
// width, which is what makes a histogram readable without a plot.
func Bar(count, largest, width int) string {
	if count <= 0 || largest <= 0 || width <= 0 {
		return ""
	}
	length := count * width / largest
	if length < 1 {
		length = 1
	}
	return strings.Repeat("#", length)
}
