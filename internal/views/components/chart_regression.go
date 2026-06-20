package components

// LinearRegression fits a straight line y = a*x + b to the given y-values
// where x is the index (0, 1, 2, ...). It returns the fitted y-value at
// each input index, in input order. The output slice has the same length
// as values.
//
// Behaviour:
//   - len < 2: returns a copy of values (constant line, slope 0).
//   - zero variance (all values equal): returns a copy of values (slope 0).
//   - NaN/Inf inputs are ignored when computing the fit; the output slot
//     at the matching index is left as 0 so callers can detect gaps.
//
// This is a simple ordinary-least-squares fit — good enough for visual
// trendlines. It's intentionally not a streaming / numerically-stable
// implementation; weight/exercise series are short (tens to hundreds of
// points at most).
func LinearRegression(values []float64) []float64 {
	out := make([]float64, len(values))
	if len(values) == 0 {
		return out
	}
	copy(out, values)

	// Collect valid (x, y) pairs.
	var sumX, sumY, sumXY, sumXX float64
	n := 0
	for i, y := range values {
		if y != y || y > 1e308 || y < -1e308 { // NaN or ±Inf
			out[i] = 0
			continue
		}
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
		n++
	}
	if n < 2 {
		return out
	}

	denom := float64(n)*sumXX - sumX*sumX
	if denom == 0 {
		// All x are equal — only possible when n==1; n>=2 here means
		// pathological input, fall back to a constant line.
		return out
	}
	slope := (float64(n)*sumXY - sumX*sumY) / denom
	intercept := (sumY - slope*sumX) / float64(n)

	for i := range out {
		// Only fill slots that came from valid inputs.
		y := values[i]
		if y != y || y > 1e308 || y < -1e308 {
			continue
		}
		out[i] = slope*float64(i) + intercept
	}
	return out
}
