package views

import "time"

// FormatDateTimeLocal formats a time.Time as the value attribute of an
// <input type="datetime-local">. The format is "YYYY-MM-DDTHH:MM" (no
// seconds, no timezone) and matches the pattern the browser round-trips
// when the field is submitted. Used by the entry and weight forms to
// pre-fill the date/time field on both create and edit.
func FormatDateTimeLocal(t time.Time) string {
	return t.Format("2006-01-02T15:04")
}
