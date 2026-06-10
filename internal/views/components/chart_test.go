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
	if !bytes.Contains([]byte(out), []byte(`h-44 sm:h-52 md:h-60`)) {
		t.Error("expected responsive height classes")
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
