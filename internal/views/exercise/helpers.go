package exercise

import (
	"sort"
	"time"

	"stren/internal/models"
)

// chartDay holds the max weight recorded for a single calendar day,
// used by chartDataForExercise to collapse multiple sets into a single
// line point. Shared between the history and chart views.
type chartDay struct {
	date   time.Time
	weight float64
}

// aggregateEntriesByDay groups a slice of ExerciseEntry by calendar
// day and reduces each day to the heaviest weight recorded. Days are
// returned in ascending order so the resulting line runs left-to-right.
// The two chart views (history and full-width) both wrap this helper
// to build their own ChartProps.
func aggregateEntriesByDay(entries []models.ExerciseEntry) []chartDay {
	byDay := make(map[string]chartDay)
	for _, e := range entries {
		key := e.CreatedAt.Format("2006-01-02")
		existing, ok := byDay[key]
		if !ok || e.Weight > existing.weight {
			byDay[key] = chartDay{date: e.CreatedAt, weight: e.Weight}
		}
	}
	days := make([]chartDay, 0, len(byDay))
	for _, d := range byDay {
		days = append(days, d)
	}
	sort.Slice(days, func(i, j int) bool {
		return days[i].date.Before(days[j].date)
	})
	return days
}

// dayLabelsAndValues turns a []chartDay into the parallel labels /
// values slices the chart props expect. format controls how the date
// is rendered (e.g. "02 Jan" for the history view).
func dayLabelsAndValues(days []chartDay, format string) ([]string, []float64) {
	labels := make([]string, len(days))
	values := make([]float64, len(days))
	for i, d := range days {
		labels[i] = d.date.Format(format)
		values[i] = d.weight
	}
	return labels, values
}
