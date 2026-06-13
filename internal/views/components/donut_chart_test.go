package components

import (
	"bytes"
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

func TestDonutChartRenders(t *testing.T) {
	props := DonutChartProps{
		ID:     "popular-exercises-chart",
		Title:  "Most Popular Exercises (7d)",
		Labels: []string{"Squat", "Bench Press", "Deadlift"},
		Values: []float64{12, 8, 5},
	}
	var buf bytes.Buffer
	if err := DonutChart(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	out := buf.String()

	// Canvas must use the requested id and the chart wrapper must fill
	// its parent (w-full h-full) with a grid auto/1fr layout so the
	// canvas takes the remaining space after the title.
	if !strings.Contains(out, `<canvas id="popular-exercises-chart">`) {
		t.Error("expected canvas with id popular-exercises-chart")
	}
	if !strings.Contains(out, `w-full h-full`) {
		t.Error("expected chart wrapper to fill its parent (w-full h-full)")
	}
	if !strings.Contains(out, `grid grid-rows-[auto_1fr]`) {
		t.Error("expected chart wrapper to be a grid with auto/1fr rows")
	}
	if !strings.Contains(out, `min-h-0 min-w-0`) {
		t.Error("expected canvas to be wrapped in a min-h-0 min-w-0 div so the 1fr row can shrink")
	}
	if !strings.Contains(out, `Most Popular Exercises (7d)`) {
		t.Error("expected chart title to be rendered above the canvas")
	}

	// The payload must be a real JSON object inside the <script> block
	// with all three labels, three values, three colors (cycled from
	// the default palette), and showLegend=true (3 slices >= 2).
	re := regexp.MustCompile(`<script id="popular-exercises-chart-data" type="application/json">([\s\S]*?)</script>`)
	m := re.FindStringSubmatch(out)
	if len(m) < 2 {
		t.Fatal("could not find data script block")
	}
	var parsed donutPayload
	if err := json.Unmarshal([]byte(m[1]), &parsed); err != nil {
		t.Fatalf("data block is not valid JSON: %v\ncontent: %s", err, m[1])
	}
	if len(parsed.Labels) != 3 || parsed.Labels[0] != "Squat" {
		t.Errorf("expected labels [Squat, Bench Press, Deadlift], got %v", parsed.Labels)
	}
	if len(parsed.Values) != 3 || parsed.Values[0] != 12 {
		t.Errorf("expected values [12, 8, 5], got %v", parsed.Values)
	}
	if len(parsed.Colors) != 3 {
		t.Fatalf("expected 3 colors (cycled from default palette), got %d (%v)", len(parsed.Colors), parsed.Colors)
	}
	if parsed.Colors[0] != donutDefaultPalette[0] {
		t.Errorf("expected first color to be %s, got %s", donutDefaultPalette[0], parsed.Colors[0])
	}
	if !parsed.ShowLegend {
		t.Error("expected showLegend=true when there are 3 slices")
	}
}

func TestDonutChartHiddenWithNoValues(t *testing.T) {
	props := DonutChartProps{
		ID:     "popular-exercises-chart",
		Labels: []string{"Squat"},
		Values: nil, // explicit no-data case
	}
	var buf bytes.Buffer
	if err := DonutChart(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output with no values, got %d bytes: %s", buf.Len(), buf.String())
	}
}

func TestDonutChartHidesLegendForSingleSlice(t *testing.T) {
	// A single-slice donut is a solid disc; a legend with one item is
	// noise, so the payload should report showLegend=false.
	props := DonutChartProps{
		ID:     "popular-exercises-chart",
		Labels: []string{"Squat"},
		Values: []float64{7},
	}
	var buf bytes.Buffer
	if err := DonutChart(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	out := buf.String()
	re := regexp.MustCompile(`<script id="popular-exercises-chart-data" type="application/json">([\s\S]*?)</script>`)
	m := re.FindStringSubmatch(out)
	if len(m) < 2 {
		t.Fatal("could not find data script block")
	}
	var parsed donutPayload
	if err := json.Unmarshal([]byte(m[1]), &parsed); err != nil {
		t.Fatalf("data block is not valid JSON: %v", err)
	}
	if parsed.ShowLegend {
		t.Error("expected showLegend=false for a single-slice donut")
	}
}

func TestDonutChartRespectsExplicitHideLegend(t *testing.T) {
	props := DonutChartProps{
		ID:         "popular-exercises-chart",
		Labels:     []string{"Squat", "Bench"},
		Values:     []float64{3, 2},
		HideLegend: true,
	}
	var buf bytes.Buffer
	if err := DonutChart(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	re := regexp.MustCompile(`<script id="popular-exercises-chart-data" type="application/json">([\s\S]*?)</script>`)
	m := re.FindStringSubmatch(buf.String())
	if len(m) < 2 {
		t.Fatal("could not find data script block")
	}
	var parsed donutPayload
	if err := json.Unmarshal([]byte(m[1]), &parsed); err != nil {
		t.Fatalf("data block is not valid JSON: %v", err)
	}
	if parsed.ShowLegend {
		t.Error("expected showLegend=false when HideLegend is set")
	}
}

func TestDonutChartCyclesDefaultPalettePastSix(t *testing.T) {
	// The default palette has 6 entries, so the 7th slice (index 6)
	// should wrap back to palette[0]. Feed 7 slices and lock in the
	// wrap point so a future palette-length change is caught.
	props := DonutChartProps{
		ID:     "popular-exercises-chart",
		Labels: []string{"a", "b", "c", "d", "e", "f", "g"},
		Values: []float64{1, 1, 1, 1, 1, 1, 1},
	}
	var buf bytes.Buffer
	if err := DonutChart(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	re := regexp.MustCompile(`<script id="popular-exercises-chart-data" type="application/json">([\s\S]*?)</script>`)
	m := re.FindStringSubmatch(buf.String())
	if len(m) < 2 {
		t.Fatal("could not find data script block")
	}
	var parsed donutPayload
	if err := json.Unmarshal([]byte(m[1]), &parsed); err != nil {
		t.Fatalf("data block is not valid JSON: %v", err)
	}
	if len(parsed.Colors) != 7 {
		t.Fatalf("expected 7 colors, got %d", len(parsed.Colors))
	}
	if parsed.Colors[6] != donutDefaultPalette[0] {
		t.Errorf("expected 7th color to wrap to %s, got %s", donutDefaultPalette[0], parsed.Colors[6])
	}
}

func TestDonutChartFillsCallerColorsThenCycles(t *testing.T) {
	// Caller supplies 2 of 5 colors; the remaining 3 should come from
	// the default palette in order.
	props := DonutChartProps{
		ID:     "popular-exercises-chart",
		Labels: []string{"a", "b", "c", "d", "e"},
		Values: []float64{1, 1, 1, 1, 1},
		Colors: []string{"#111111", "#222222"},
	}
	var buf bytes.Buffer
	if err := DonutChart(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	re := regexp.MustCompile(`<script id="popular-exercises-chart-data" type="application/json">([\s\S]*?)</script>`)
	m := re.FindStringSubmatch(buf.String())
	if len(m) < 2 {
		t.Fatal("could not find data script block")
	}
	var parsed donutPayload
	if err := json.Unmarshal([]byte(m[1]), &parsed); err != nil {
		t.Fatalf("data block is not valid JSON: %v", err)
	}
	if len(parsed.Colors) != 5 {
		t.Fatalf("expected 5 colors, got %d", len(parsed.Colors))
	}
	want := []string{"#111111", "#222222", donutDefaultPalette[0], donutDefaultPalette[1], donutDefaultPalette[2]}
	for i, w := range want {
		if parsed.Colors[i] != w {
			t.Errorf("color %d: want %s, got %s", i, w, parsed.Colors[i])
		}
	}
}

func TestDonutChartScriptsQuoteIDs(t *testing.T) {
	// Mirror the chart_test.go contract: the inline script must look up
	// the canvas and data elements via getElementById with quoted string
	// literals, not concatenated variables.
	props := DonutChartProps{
		ID:     "popular-exercises-chart",
		Labels: []string{"Squat", "Bench"},
		Values: []float64{1, 1},
	}
	var buf bytes.Buffer
	if err := DonutChart(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `getElementById("popular-exercises-chart")`) {
		t.Error("expected getElementById to use string literal with quotes for canvas id")
	}
	if !strings.Contains(out, `getElementById("popular-exercises-chart-data")`) {
		t.Error("expected getElementById to use string literal with quotes for data id")
	}
}

func TestDonutChartEmptyStringInColorsFallsBackToDefault(t *testing.T) {
	// Caller passes 3 colors but the middle one is "" — the chart
	// should fill the empty slot from the default palette (the next
	// cycled entry, starting from index 0) and leave the explicit
	// override untouched.
	props := DonutChartProps{
		ID:     "popular-exercises-chart",
		Labels: []string{"a", "b", "c"},
		Values: []float64{1, 1, 1},
		Colors: []string{"", "#ff0000", ""},
	}
	var buf bytes.Buffer
	if err := DonutChart(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	re := regexp.MustCompile(`<script id="popular-exercises-chart-data" type="application/json">([\s\S]*?)</script>`)
	m := re.FindStringSubmatch(buf.String())
	if len(m) < 2 {
		t.Fatal("could not find data script block")
	}
	var parsed donutPayload
	if err := json.Unmarshal([]byte(m[1]), &parsed); err != nil {
		t.Fatalf("data block is not valid JSON: %v", err)
	}
	want := []string{donutDefaultPalette[0], "#ff0000", donutDefaultPalette[1]}
	if len(parsed.Colors) != len(want) {
		t.Fatalf("expected %d colors, got %d (%v)", len(want), len(parsed.Colors), parsed.Colors)
	}
	for i, w := range want {
		if parsed.Colors[i] != w {
			t.Errorf("color %d: want %s, got %s", i, w, parsed.Colors[i])
		}
	}
}

func TestDonutChartEmptyStringFallbackCyclesPastPaletteLength(t *testing.T) {
	// 7 slices all with "" in Colors; the chart should fill them
	// from the default palette cycled, wrapping on index 6.
	props := DonutChartProps{
		ID:     "popular-exercises-chart",
		Labels: []string{"a", "b", "c", "d", "e", "f", "g"},
		Values: []float64{1, 1, 1, 1, 1, 1, 1},
		Colors: []string{"", "", "", "", "", "", ""},
	}
	var buf bytes.Buffer
	if err := DonutChart(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	re := regexp.MustCompile(`<script id="popular-exercises-chart-data" type="application/json">([\s\S]*?)</script>`)
	m := re.FindStringSubmatch(buf.String())
	if len(m) < 2 {
		t.Fatal("could not find data script block")
	}
	var parsed donutPayload
	if err := json.Unmarshal([]byte(m[1]), &parsed); err != nil {
		t.Fatalf("data block is not valid JSON: %v", err)
	}
	for i := 0; i < 7; i++ {
		want := donutDefaultPalette[i%len(donutDefaultPalette)]
		if parsed.Colors[i] != want {
			t.Errorf("color %d: want %s, got %s", i, want, parsed.Colors[i])
		}
	}
}

func TestDonutChartEmptyStringFallbackPreservesExplicitOverrideAtEnd(t *testing.T) {
	// The "Other"-bucket pattern: 5 empty slots + 1 explicit gray.
	// The first 5 slots should be filled with the default palette
	// (positions 0..4) and the gray must survive untouched at index 5.
	props := DonutChartProps{
		ID:     "popular-exercises-chart",
		Labels: []string{"a", "b", "c", "d", "e", "Other"},
		Values: []float64{1, 1, 1, 1, 1, 1},
		Colors: []string{"", "", "", "", "", "#9ca3af"},
	}
	var buf bytes.Buffer
	if err := DonutChart(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	re := regexp.MustCompile(`<script id="popular-exercises-chart-data" type="application/json">([\s\S]*?)</script>`)
	m := re.FindStringSubmatch(buf.String())
	if len(m) < 2 {
		t.Fatal("could not find data script block")
	}
	var parsed donutPayload
	if err := json.Unmarshal([]byte(m[1]), &parsed); err != nil {
		t.Fatalf("data block is not valid JSON: %v", err)
	}
	want := []string{
		donutDefaultPalette[0],
		donutDefaultPalette[1],
		donutDefaultPalette[2],
		donutDefaultPalette[3],
		donutDefaultPalette[4],
		"#9ca3af",
	}
	for i, w := range want {
		if parsed.Colors[i] != w {
			t.Errorf("color %d: want %s, got %s", i, w, parsed.Colors[i])
		}
	}
}

func TestDonutChartJSPaletteMatchesGoPalette(t *testing.T) {
	// Drift guard: the inline JS fallback array must contain every
	// color in donutDefaultPalette. The Go-side resolution is the
	// source of truth at render time, but the JS array is what fires
	// when the server-side payload is missing or malformed — keep
	// them in sync so a brand refresh is a one-file change.
	props := DonutChartProps{
		ID:     "popular-exercises-chart",
		Labels: []string{"a", "b"},
		Values: []float64{1, 1},
	}
	var buf bytes.Buffer
	if err := DonutChart(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	out := buf.String()
	for _, c := range donutDefaultPalette {
		if !strings.Contains(out, c) {
			t.Errorf("expected JS palette to contain %s so the brand ramp stays consistent in both layers", c)
		}
	}
}
