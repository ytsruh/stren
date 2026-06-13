package components

import (
	"bytes"
	"context"
	"encoding/json"
	"regexp"
	"testing"
)

func TestScatterChartRenders(t *testing.T) {
	props := ScatterChartProps{
		ID:     "scatter-chart",
		Title:  "Reps vs Weight",
		XLabel: "Reps",
		YLabel: "Squat (kg)",
		Points: []ScatterPoint{
			{X: 5, Y: 100, Date: "10 Jun"},
			{X: 5, Y: 102.5, Date: "12 Jun"},
		},
	}
	var buf bytes.Buffer
	if err := ScatterChart(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	out := buf.String()

	if !bytes.Contains([]byte(out), []byte(`<canvas id="scatter-chart">`)) {
		t.Error("expected canvas with id scatter-chart")
	}
	if !bytes.Contains([]byte(out), []byte(`w-full h-full`)) {
		t.Error("expected chart wrapper to fill its parent (w-full h-full)")
	}
	if !bytes.Contains([]byte(out), []byte(`grid grid-rows-[auto_1fr]`)) {
		t.Error("expected chart wrapper to be a grid with auto/1fr rows")
	}
	if !bytes.Contains([]byte(out), []byte(`row-start-2`)) {
		t.Error("expected canvas wrapper to use row-start-2 so the canvas fills the 1fr row even when title is present")
	}
	if !bytes.Contains([]byte(out), []byte(`getElementById("scatter-chart")`)) {
		t.Error("expected getElementById to use string literal with quotes")
	}
	if !bytes.Contains([]byte(out), []byte(`getElementById("scatter-chart-data")`)) {
		t.Error("expected getElementById to use string literal with quotes for data id")
	}
	if !bytes.Contains([]byte(out), []byte(`type: 'scatter'`)) {
		t.Error("expected inline script to construct a scatter-type Chart.js chart")
	}

	// JSON payload shape: xLabel, yLabel, points, color, hideAxes
	re := regexp.MustCompile(`<script id="scatter-chart-data" type="application/json">([\s\S]*?)</script>`)
	m := re.FindStringSubmatch(out)
	if len(m) < 2 {
		t.Fatal("could not find data script block")
	}
	var parsed struct {
		XLabel   string         `json:"xLabel"`
		YLabel   string         `json:"yLabel"`
		Points   []ScatterPoint `json:"points"`
		Color    string         `json:"color"`
		HideAxes bool           `json:"hideAxes"`
	}
	if err := json.Unmarshal([]byte(m[1]), &parsed); err != nil {
		t.Fatalf("data block is not valid JSON: %v\ncontent: %s", err, m[1])
	}
	if parsed.XLabel != "Reps" {
		t.Errorf("expected xLabel 'Reps', got %q", parsed.XLabel)
	}
	if parsed.YLabel != "Squat (kg)" {
		t.Errorf("expected yLabel 'Squat (kg)', got %q", parsed.YLabel)
	}
	if len(parsed.Points) != 2 {
		t.Errorf("expected 2 points, got %d", len(parsed.Points))
	}
	if parsed.Points[0].X != 5 || parsed.Points[0].Y != 100 || parsed.Points[0].Date != "10 Jun" {
		t.Errorf("unexpected first point: %+v", parsed.Points[0])
	}
	if parsed.Color != chartDefaultColor {
		t.Errorf("expected default color %s, got %s", chartDefaultColor, parsed.Color)
	}
	if parsed.HideAxes {
		t.Error("expected default HideAxes=false in payload")
	}
}

func TestScatterChartHiddenBelowTwoPoints(t *testing.T) {
	cases := []struct {
		name   string
		points []ScatterPoint
	}{
		{"no points", nil},
		{"one point", []ScatterPoint{{X: 5, Y: 100, Date: "10 Jun"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			props := ScatterChartProps{
				ID:     "scatter-chart",
				XLabel: "Reps",
				YLabel: "Squat (kg)",
				Points: tc.points,
			}
			var buf bytes.Buffer
			if err := ScatterChart(props).Render(context.Background(), &buf); err != nil {
				t.Fatalf("render failed: %v", err)
			}
			if buf.Len() != 0 {
				t.Errorf("expected empty output, got %d bytes", buf.Len())
			}
		})
	}
}

func TestScatterChartFillsParent(t *testing.T) {
	props := ScatterChartProps{
		ID:     "scatter-chart",
		XLabel: "Reps",
		YLabel: "Squat (kg)",
		Points: []ScatterPoint{
			{X: 5, Y: 100, Date: "10 Jun"},
			{X: 5, Y: 102.5, Date: "12 Jun"},
		},
	}
	var buf bytes.Buffer
	if err := ScatterChart(props).Render(context.Background(), &buf); err != nil {
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

func TestScatterChartHideAxes(t *testing.T) {
	props := ScatterChartProps{
		ID:       "scatter-chart",
		XLabel:   "Reps",
		YLabel:   "Squat (kg)",
		Points:   []ScatterPoint{{X: 5, Y: 100, Date: "10 Jun"}, {X: 5, Y: 102.5, Date: "12 Jun"}},
		HideAxes: true,
	}
	var buf bytes.Buffer
	if err := ScatterChart(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	out := buf.String()

	re := regexp.MustCompile(`<script id="scatter-chart-data" type="application/json">([\s\S]*?)</script>`)
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

	// The script should reference data.hideAxes on both axes so the tick
	// labels can be toggled off independently.
	if got := bytes.Count([]byte(out), []byte("data.hideAxes")); got != 2 {
		t.Errorf("expected data.hideAxes to be referenced on both axes, got %d occurrences", got)
	}
}

func TestScatterChartCustomColor(t *testing.T) {
	props := ScatterChartProps{
		ID:     "scatter-chart",
		XLabel: "Reps",
		YLabel: "Squat (kg)",
		Points: []ScatterPoint{
			{X: 5, Y: 100, Date: "10 Jun"},
			{X: 5, Y: 102.5, Date: "12 Jun"},
		},
		Color: "#336699",
	}
	var buf bytes.Buffer
	if err := ScatterChart(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte(`#336699`)) {
		t.Error("expected custom color #336699 to be embedded in the payload")
	}
}

func TestScatterChartDateInTooltip(t *testing.T) {
	props := ScatterChartProps{
		ID:     "scatter-chart",
		XLabel: "Reps",
		YLabel: "Squat (kg)",
		Points: []ScatterPoint{
			{X: 5, Y: 100, Date: "10 Jun"},
			{X: 5, Y: 102.5, Date: "12 Jun"},
		},
	}
	var buf bytes.Buffer
	if err := ScatterChart(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	out := buf.String()
	// The tooltip label callback must surface the date field of each point.
	if !bytes.Contains([]byte(out), []byte("'Date: ' + p.date")) {
		t.Error("expected tooltip callback to render the date field of each point")
	}
	if !bytes.Contains([]byte(out), []byte("'Reps: ' + p.x")) {
		t.Error("expected tooltip callback to render the reps (x) field of each point")
	}
}

func TestScatterChartKeepsAllPoints(t *testing.T) {
	// Three identical (reps, weight) sets must each appear as a separate
	// point — the chart deliberately does not aggregate so overlapping sets
	// stay visible (alpha 0.6 fill).
	props := ScatterChartProps{
		ID:     "scatter-chart",
		XLabel: "Reps",
		YLabel: "Squat (kg)",
		Points: []ScatterPoint{
			{X: 5, Y: 100, Date: "10 Jun"},
			{X: 5, Y: 100, Date: "10 Jun"},
			{X: 5, Y: 100, Date: "10 Jun"},
		},
	}
	var buf bytes.Buffer
	if err := ScatterChart(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	out := buf.String()
	re := regexp.MustCompile(`<script id="scatter-chart-data" type="application/json">([\s\S]*?)</script>`)
	m := re.FindStringSubmatch(out)
	if len(m) < 2 {
		t.Fatal("could not find data script block")
	}
	var parsed struct {
		Points []ScatterPoint `json:"points"`
	}
	if err := json.Unmarshal([]byte(m[1]), &parsed); err != nil {
		t.Fatalf("data block is not valid JSON: %v", err)
	}
	if len(parsed.Points) != 3 {
		t.Errorf("expected all 3 overlapping points to be preserved, got %d", len(parsed.Points))
	}
}

func TestScatterChartRendersWithoutTitle(t *testing.T) {
	// No title — the canvas should still be in the 1fr row because the
	// canvas wrapper carries row-start-2 (this regression-locks the
	// previous fix against accidentally removing row-start-2).
	props := ScatterChartProps{
		ID:     "scatter-chart",
		XLabel: "Reps",
		YLabel: "Squat (kg)",
		Points: []ScatterPoint{
			{X: 5, Y: 100, Date: "10 Jun"},
			{X: 5, Y: 102.5, Date: "12 Jun"},
		},
	}
	var buf bytes.Buffer
	if err := ScatterChart(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte(`<canvas id="scatter-chart">`)) {
		t.Error("expected canvas to render even when title is empty")
	}
	if !bytes.Contains([]byte(out), []byte(`row-start-2`)) {
		t.Error("expected canvas wrapper to keep row-start-2 so the 1fr row holds the canvas without a title row")
	}
}

func TestScatterChartXAxsisPadding(t *testing.T) {
	// The x-axis (reps) must use the same min/max + 10% padding pattern
	// as the y-axis (weight) so the leftmost/rightmost dots aren't flush
	// with the chart edges. beginAtZero is removed because forcing the
	// axis to start at 0 leaves a large empty area when the user only
	// does high-rep sets.
	props := ScatterChartProps{
		ID:     "scatter-chart",
		XLabel: "Reps",
		YLabel: "Squat (kg)",
		Points: []ScatterPoint{
			{X: 5, Y: 100, Date: "10 Jun"},
			{X: 10, Y: 80, Date: "12 Jun"},
		},
	}
	var buf bytes.Buffer
	if err := ScatterChart(props).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}
	out := buf.String()

	// xMin/xMax are computed from the data with 10% padding, mirroring
	// the y-axis. xMin is floored at 1 because reps are always >= 1.
	if !bytes.Contains([]byte(out), []byte("var xMin, xMax")) {
		t.Error("expected x-axis to compute its own min/max from the data")
	}
	if !bytes.Contains([]byte(out), []byte("Math.max(1, xmn - xpad)")) {
		t.Error("expected xMin to be floored at 1 so a high-rep user doesn't see an axis starting at 0")
	}
	if !bytes.Contains([]byte(out), []byte("(xmx - xmn) * 0.10")) {
		t.Error("expected x-axis to use the same 10% padding as the y-axis")
	}
	// The x scale uses suggestedMin/suggestedMax from the computed
	// values, not beginAtZero (which would re-introduce the empty area).
	if !bytes.Contains([]byte(out), []byte("suggestedMin: xMin")) {
		t.Error("expected x scale to use suggestedMin: xMin from the padding calculation")
	}
	if !bytes.Contains([]byte(out), []byte("suggestedMax: xMax")) {
		t.Error("expected x scale to use suggestedMax: xMax from the padding calculation")
	}
	if bytes.Contains([]byte(out), []byte("beginAtZero: true")) {
		t.Error("x scale should not use beginAtZero — it forces a 0 origin even when the user's data starts higher")
	}
}
