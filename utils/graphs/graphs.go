// Package graphs renders charts from explicitly mapped tabular data.
package graphs

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"

	chartlib "github.com/go-analyze/charts"
	"github.com/lmorg/ttyphoon/types"
)

// DefaultWidth and DefaultHeight are the fixed output dimensions.
const (
	DefaultWidth  = 800
	DefaultHeight = 600
)

// Type selects a table-compatible chart style.
type Type string

const (
	Line          Type = "line"
	Area          Type = "area"
	Bar           Type = "bar"
	HorizontalBar Type = "horizontal-bar"
	Scatter       Type = "scatter"
	Pie           Type = "pie"
	Doughnut      Type = "doughnut"
)

// Format selects the encoded output image format.
type Format string

const (
	SVG Format = "svg"
	PNG Format = "png"
)

// Table is a header row and string-valued records, as used by CSV and Markdown tables.
type Table struct {
	Headings []string
	Rows     [][]string
}

// Spec maps table columns to chart labels and numeric series.
type Spec struct {
	Type Type
	// Format defaults to SVG when empty.
	Format Format
	Title  string
	// Column indices are zero-based. CategoryColumns, when nonempty, overrides
	// CategoryColumn and joins the selected cells with ", ".
	CategoryColumn  int
	CategoryColumns []int
	ValueColumns    []int
	SeriesNames     []string
	// RowIndices selects zero-based source rows; empty means all rows.
	RowIndices []int
	// CategoryAxisTitle and ValueAxisTitle follow the data roles for horizontal bars.
	CategoryAxisTitle string
	ValueAxisTitle    string
}

// RenderTable validates a table mapping, renders it, and returns SVG or PNG data.
func RenderTable(table Table, spec Spec) ([]byte, error) {
	if len(table.Headings) == 0 {
		return nil, fmt.Errorf("graphs: table has no headings")
	}
	rows := table.Rows
	if len(spec.RowIndices) > 0 {
		selected := make(map[int]struct{}, len(spec.RowIndices))
		for _, index := range spec.RowIndices {
			selected[index] = struct{}{}
		}
		rows = make([][]string, 0, len(selected))
		for index, row := range table.Rows {
			if _, ok := selected[index]; ok {
				rows = append(rows, row)
			}
		}
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("graphs: table has no data rows")
	}
	categoryColumns := spec.CategoryColumns
	if len(categoryColumns) == 0 {
		categoryColumns = []int{spec.CategoryColumn}
	}
	for _, column := range categoryColumns {
		if column < 0 || column >= len(table.Headings) {
			return nil, fmt.Errorf("graphs: category column %d is out of range", column)
		}
	}
	if len(spec.ValueColumns) == 0 {
		return nil, fmt.Errorf("graphs: select at least one value column")
	}
	if (spec.Type == Pie || spec.Type == Doughnut) && len(spec.ValueColumns) != 1 {
		return nil, fmt.Errorf("graphs: %s charts require exactly one value column", spec.Type)
	}
	if spec.Type != Line && spec.Type != Area && spec.Type != Bar && spec.Type != HorizontalBar && spec.Type != Scatter && spec.Type != Pie && spec.Type != Doughnut {
		return nil, fmt.Errorf("graphs: unsupported chart type %q", spec.Type)
	}

	for _, column := range spec.ValueColumns {
		if column < 0 || column >= len(table.Headings) {
			return nil, fmt.Errorf("graphs: value column %d is out of range", column)
		}
		for _, category := range categoryColumns {
			if column == category {
				return nil, fmt.Errorf("graphs: category column %d cannot also be a value column", column)
			}
		}
	}

	format, err := normalizeFormat(spec.Format)
	if err != nil {
		return nil, err
	}
	categories := make([]string, len(rows))
	values := make([][]float64, len(spec.ValueColumns))
	for seriesIndex := range values {
		values[seriesIndex] = make([]float64, len(rows))
	}
	for rowIndex, row := range rows {
		labels := make([]string, len(categoryColumns))
		for i, column := range categoryColumns {
			labels[i] = cell(row, column)
		}
		categories[rowIndex] = strings.Join(labels, ", ")
		for seriesIndex, column := range spec.ValueColumns {
			number, err := parseNumber(cell(row, column))
			if err != nil {
				return nil, fmt.Errorf("graphs: row %d, column %q: %w", rowIndex+2, table.Headings[column], err)
			}
			values[seriesIndex][rowIndex] = number
		}
	}

	chartType, fillArea, err := libraryType(spec.Type)
	if err != nil {
		return nil, err
	}
	seriesNames := make([]string, len(values))
	for i := range values {
		if i < len(spec.SeriesNames) && strings.TrimSpace(spec.SeriesNames[i]) != "" {
			seriesNames[i] = strings.TrimSpace(spec.SeriesNames[i])
		} else {
			seriesNames[i] = table.Headings[spec.ValueColumns[i]]
		}
	}

	series := make(chartlib.GenericSeriesList, 0, len(values))
	if spec.Type == Pie || spec.Type == Doughnut {
		for rowIndex, value := range values[0] {
			series = append(series, chartlib.GenericSeries{
				Type:   chartType,
				Name:   categories[rowIndex],
				Values: []float64{value},
			})
		}
	} else {
		for i, data := range values {
			series = append(series, chartlib.GenericSeries{
				Type:   chartType,
				Name:   seriesNames[i],
				Values: data,
			})
		}
	}

	option := chartlib.ChartOption{
		OutputFormat: string(format),
		Width:        DefaultWidth,
		Height:       DefaultHeight,
		Theme:        DefaultTheme(),
		Title:        chartlib.TitleOption{Text: spec.Title},
		SeriesList:   series,
	}
	if spec.Type == HorizontalBar {
		option.YAxis = []chartlib.YAxisOption{{
			Title:  spec.CategoryAxisTitle,
			Labels: categories,
		}}
		if spec.ValueAxisTitle != "" {
			option.XAxis.Title = spec.ValueAxisTitle
		}
	} else if spec.Type != Pie && spec.Type != Doughnut {
		option.XAxis = chartlib.XAxisOption{
			Title:  spec.CategoryAxisTitle,
			Labels: categories,
		}
		option.YAxis = []chartlib.YAxisOption{{Title: spec.ValueAxisTitle}}
	}
	if fillArea {
		option.FillArea = chartlib.Ptr(true)
		option.FillOpacity = 96
	}

	painter, err := chartlib.Render(option)
	if err != nil {
		return nil, fmt.Errorf("graphs: render %s chart: %w", spec.Type, err)
	}
	data, err := painter.Bytes()
	if err != nil {
		return nil, fmt.Errorf("graphs: encode %s chart: %w", spec.Type, err)
	}
	return data, nil
}

func DefaultTheme() chartlib.ColorPalette {
	name := chartlib.ThemeDark
	if types.THEME_LIGHT {
		name = chartlib.ThemeLight
	}
	palette := chartlib.GetTheme(name)
	foreground := chartColor(types.SGR_COLOR_FOREGROUND, chartlib.ColorWhite)
	background := chartColor(types.SGR_COLOR_BACKGROUND, chartlib.ColorBlack)
	series := []chartlib.Color{
		chartColor(types.SGR_COLOR_ACCENT, chartlib.ColorBlue),
		chartColor(types.SGR_COLOR_CYAN, chartlib.ColorRGB(0, 217, 210)),
		chartColor(types.SGR_COLOR_GREEN, chartlib.ColorGreen),
		chartColor(types.SGR_COLOR_RED, chartlib.ColorRed),
		chartColor(types.SGR_COLOR_YELLOW, chartlib.ColorYellow),
		chartColor(types.SGR_COLOR_MAGENTA, chartlib.ColorRGB(217, 0, 116)),
	}
	return palette.
		WithBackgroundColor(background).
		WithTextColor(foreground).
		WithXAxisColor(foreground).
		WithYAxisColor(foreground).
		WithSeriesColors(series)
}

func chartColor(color *types.Colour, fallback chartlib.Color) chartlib.Color {
	if color == nil {
		return fallback
	}
	alpha := color.Alpha
	if alpha == 0 {
		alpha = 255
	}
	return chartlib.Color{R: color.Red, G: color.Green, B: color.Blue, A: alpha}
}

func normalizeFormat(format Format) (Format, error) {
	if format == "" {
		return SVG, nil
	}
	switch Format(strings.ToLower(strings.TrimSpace(string(format)))) {
	case SVG, PNG:
		return Format(strings.ToLower(strings.TrimSpace(string(format)))), nil
	default:
		return "", fmt.Errorf("graphs: unsupported output format %q; use svg or png", format)
	}
}

func libraryType(kind Type) (string, bool, error) {
	switch kind {
	case Line:
		return chartlib.ChartTypeLine, false, nil
	case Scatter:
		return chartlib.ChartTypeScatter, false, nil
	case Area:
		return chartlib.ChartTypeLine, true, nil
	case Bar:
		return chartlib.ChartTypeBar, false, nil
	case HorizontalBar:
		return chartlib.ChartTypeHorizontalBar, false, nil
	case Pie:
		return chartlib.ChartTypePie, false, nil
	case Doughnut:
		return chartlib.ChartTypeDoughnut, false, nil
	default:
		return "", false, fmt.Errorf("graphs: unsupported chart type %q", kind)
	}
}

func cell(row []string, column int) string {
	if column >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[column])
}

func parseNumber(value string) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return chartlib.GetNullValue(), nil
	}
	var normalized strings.Builder
	normalized.Grow(len(value))
	for _, char := range value {
		if char == ',' || unicode.Is(unicode.Sc, char) {
			continue
		}
		normalized.WriteRune(char)
	}
	number, err := strconv.ParseFloat(strings.TrimSpace(normalized.String()), 64)
	if err != nil {
		return 0, fmt.Errorf("%q is not numeric", value)
	}
	if math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, fmt.Errorf("%q is not a finite number", value)
	}
	return number, nil
}
