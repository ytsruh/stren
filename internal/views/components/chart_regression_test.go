package components

import (
	"math"
	"testing"
)

func TestLinearRegression_TwoPoints(t *testing.T) {
	// y = 2x + 1 → [1, 3]
	got := LinearRegression([]float64{1, 3})
	want := []float64{1, 3}
	if !floatSliceEqual(got, want, 1e-9) {
		t.Errorf("two-point exact fit: got %v, want %v", got, want)
	}
}

func TestLinearRegression_PerfectLine(t *testing.T) {
	// Perfect y = 0.5x + 10 over 5 points.
	in := []float64{10, 10.5, 11, 11.5, 12}
	got := LinearRegression(in)
	if !floatSliceEqual(got, in, 1e-9) {
		t.Errorf("perfect line should fit exactly: got %v, want %v", got, in)
	}
}

func TestLinearRegression_NoisyData(t *testing.T) {
	// Linear data with small noise — fitted line should pass close to mean.
	in := []float64{82.0, 81.8, 81.6, 81.4, 81.2, 81.0, 80.8}
	got := LinearRegression(in)

	// Slope should be ~-0.2 per index.
	slope := got[len(got)-1] - got[0]
	wantSlope := -0.2 * float64(len(got)-1)
	if math.Abs(slope-wantSlope) > 1e-6 {
		t.Errorf("expected slope*range ≈ %v, got %v", wantSlope, slope)
	}
}

func TestLinearRegression_Constant(t *testing.T) {
	// All values equal — slope is 0, output is the constant.
	in := []float64{80, 80, 80, 80}
	got := LinearRegression(in)
	if !floatSliceEqual(got, in, 1e-9) {
		t.Errorf("constant input should pass through: got %v, want %v", got, in)
	}
}

func TestLinearRegression_SinglePoint(t *testing.T) {
	in := []float64{82.4}
	got := LinearRegression(in)
	if !floatSliceEqual(got, in, 1e-9) {
		t.Errorf("single point should be returned as-is: got %v, want %v", got, in)
	}
}

func TestLinearRegression_Empty(t *testing.T) {
	got := LinearRegression(nil)
	if len(got) != 0 {
		t.Errorf("empty input should return empty output, got %v", got)
	}
}

func TestLinearRegression_PreservesLength(t *testing.T) {
	in := []float64{1, 2, 3, 4, 5, 6, 7}
	got := LinearRegression(in)
	if len(got) != len(in) {
		t.Fatalf("output length %d != input length %d", len(got), len(in))
	}
}

func floatSliceEqual(a, b []float64, tol float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if math.Abs(a[i]-b[i]) > tol {
			return false
		}
	}
	return true
}
