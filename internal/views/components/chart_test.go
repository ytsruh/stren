package components

import (
	"bytes"
	"context"
	"encoding/json"
	"regexp"
	"testing"
)

func TestChartRenders(t *testing.T) {
	props := ChartProps{
		ID:     "weight-chart",
		Title:  "Weight Over Time",
		Labels: []string{"10 Jun", "11 Jun"},
		Datasets: []ChartDataset{
			{Label: "Weight (kg)", Values: []float64{82.4, 82.1}},
		},
	}
	var buf bytes.Buffer
	if err := Chart(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	out := buf.String()

	if !bytes.Contains([]byte(out), []byte(`<canvas id="weight-chart">`)) {
		t.Error("expected canvas with id weight-chart")
	}
	if !bytes.Contains([]byte(out), []byte(`w-full h-full`)) {
		t.Error("expected chart wrapper to fill its parent (w-full h-full)")
	}
	// The wrapper must be a CSS grid with two rows (title + canvas) so the
	// canvas fills the *remaining* space after the title rather than the
	// full wrapper height. Without this, height:100% on the canvas causes
	// it to overflow the wrapper (and the card) by the title's height.
	if !bytes.Contains([]byte(out), []byte(`grid grid-rows-[auto_1fr]`)) {
		t.Error("expected chart wrapper to be a grid with auto/1fr rows so the canvas fills remaining space")
	}
	// Grid items default to min-height:auto (canvas intrinsic 150px) AND
	// min-width:auto (canvas intrinsic 300px). The min-h-0/min-w-0 wrapper
	// around the canvas opts out of both so the 1fr row can shrink to
	// small layouts and the canvas tracks the real container width on
	// resize (which also unblocks Chart.js's ResizeObserver).
	if !bytes.Contains([]byte(out), []byte(`<div class="min-h-0 min-w-0"><canvas id="weight-chart">`)) {
		t.Error("expected canvas to be wrapped in a min-h-0 min-w-0 div so the 1fr row can shrink and the canvas can resize")
	}
	if !bytes.Contains([]byte(out), []byte(`getElementById("weight-chart")`)) {
		t.Error("expected getElementById to use string literal with quotes")
	}
	if !bytes.Contains([]byte(out), []byte(`getElementById("weight-chart-data")`)) {
		t.Error("expected getElementById to use string literal with quotes for data id")
	}

	// The JSON in the data element must be a real object, not a string-of-JSON.
	re := regexp.MustCompile(`<script id="weight-chart-data" type="application/json">([\s\S]*?)</script>`)
	m := re.FindStringSubmatch(out)
	if len(m) < 2 {
		t.Fatal("could not find data script block")
	}
	var parsed struct {
		Labels   []string `json:"labels"`
		Datasets []struct {
			Label  string    `json:"label"`
			Values []float64 `json:"values"`
			Color  string    `json:"color"`
		} `json:"datasets"`
		HideAxes bool `json:"hideAxes"`
	}
	if err := json.Unmarshal([]byte(m[1]), &parsed); err != nil {
		t.Fatalf("data block is not valid JSON: %v\ncontent: %s", err, m[1])
	}
	if len(parsed.Labels) != 2 || parsed.Labels[0] != "10 Jun" {
		t.Errorf("expected labels [10 Jun, 11 Jun], got %v", parsed.Labels)
	}
	if len(parsed.Datasets) != 1 || parsed.Datasets[0].Label != "Weight (kg)" {
		t.Errorf("expected one dataset 'Weight (kg)', got %+v", parsed.Datasets)
	}
	if len(parsed.Datasets[0].Values) != 2 || parsed.Datasets[0].Values[0] != 82.4 {
		t.Errorf("expected values [82.4, 82.1], got %v", parsed.Datasets[0].Values)
	}
	if parsed.Datasets[0].Color != chartDefaultColor {
		t.Errorf("expected default color %s, got %s", chartDefaultColor, parsed.Datasets[0].Color)
	}
	if parsed.HideAxes {
		t.Error("expected default HideAxes=false in payload")
	}
	if !bytes.Contains([]byte(out), []byte("labelColor")) {
		t.Error("expected tooltip labelColor callback to set box color")
	}
	if bytes.Contains([]byte(out), []byte("bodyColor")) {
		t.Error("tooltip body text color should not be overridden")
	}
}

func TestChartHiddenBelowTwoLabels(t *testing.T) {
	props := ChartProps{
		ID:     "weight-chart",
		Labels: []string{"10 Jun"},
		Datasets: []ChartDataset{
			{Label: "Weight (kg)", Values: []float64{82.4}},
		},
	}
	var buf bytes.Buffer
	if err := Chart(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output with 1 label, got %d bytes", buf.Len())
	}
}

func TestChartFillsParent(t *testing.T) {
	// ContainerClass is intentionally absent from ChartProps now: the chart
	// fills its parent and the caller owns sizing. This test locks the
	// wrapper contract in so a future refactor can't accidentally reintroduce
	// a hardcoded chart height.
	props := ChartProps{
		ID:       "history-chart",
		Labels:   []string{"10 Jun", "11 Jun"},
		Datasets: []ChartDataset{{Label: "Squat (kg)", Values: []float64{80, 85}}},
	}
	var buf bytes.Buffer
	if err := Chart(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte(`w-full h-full`)) {
		t.Errorf("expected wrapper to fill parent (w-full h-full), got: %s", out)
	}
	if bytes.Contains([]byte(out), []byte(`h-44 sm:h-52 md:h-60`)) {
		t.Errorf("chart wrapper must not impose a default height; caller owns sizing")
	}
}

func TestChartHideAxes(t *testing.T) {
	props := ChartProps{
		ID:       "history-chart",
		Labels:   []string{"10 Jun", "11 Jun"},
		Datasets: []ChartDataset{{Label: "Squat (kg)", Values: []float64{80, 85}}},
		HideAxes: true,
	}
	var buf bytes.Buffer
	if err := Chart(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	out := buf.String()

	// Payload flag must round-trip into the JSON data block so the inline
	// script can read it.
	re := regexp.MustCompile(`<script id="history-chart-data" type="application/json">([\s\S]*?)</script>`)
	m := re.FindStringSubmatch(out)
	if len(m) < 2 {
		t.Fatal("could not find data script block")
	}
	var parsed struct {
		HideAxes bool `json:"hideAxes"`
	}
	if err := json.Unmarshal([]byte(m[1]), &parsed); err != nil {
		t.Fatalf("data block is not valid JSON: %v\ncontent: %s", err, m[1])
	}
	if !parsed.HideAxes {
		t.Error("expected hideAxes=true in payload when HideAxes is set")
	}

	// Inline script must reference data.hideAxes in both the x scale ternary
	// and the y scale ticks config. (We can't substring-match the final
	// `ticks: { display: false }` value on the y axis because the y branch
	// is written as a ternary whose result is only known at JS runtime.)
	if got := bytes.Count([]byte(out), []byte("data.hideAxes")); got != 2 {
		t.Errorf("expected data.hideAxes to be referenced on both axes, got %d occurrences", got)
	}
}
