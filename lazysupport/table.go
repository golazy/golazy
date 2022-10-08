package lazysupport

import (
	"fmt"
	"strconv"
	"strings"
)

type Table struct {
	Header []string
	Values [][]string
}

type tableWidth []int

func (t *Table) updateTableWidth(tw *tableWidth, row []string) {
	for missingWidths := len(row) - len(*tw); missingWidths > 0; missingWidths = missingWidths - 1 {
		*tw = append(*tw, 0)
	}
	for i, col := range row {
		width := len(col)
		if (*tw)[i] < width {
			(*tw)[i] = width
		}
	}
}

func (t *Table) renderRow(tw *tableWidth, row []string) string {
	content := ""
	for i, col := range row {

		format := "%-" + strconv.Itoa((*tw)[i]) + "s "
		content = content + fmt.Sprintf(format, col)
	}

	return strings.TrimSpace(content)

}

func (t *Table) String() string {

	// Calculate columns space
	var tw tableWidth

	t.updateTableWidth(&tw, t.Header)
	for _, row := range t.Values {
		t.updateTableWidth(&tw, row)
	}
	if len(tw) == 0 {
		return ""
	}

	// Render the table
	var lines []string
	if t.Header != nil {
		lines = append(lines, t.renderRow(&tw, t.Header))
	}
	for _, row := range t.Values {
		lines = append(lines, t.renderRow(&tw, row))
	}
	return strings.Join(lines, "\n") + "\n"
}
