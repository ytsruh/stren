package goals

import "time"

// timeT is an alias for time.Time used by the goals view package
// so the templ expressions don't have to import "time" directly.
// Kept as a named type so tests can substitute a fixed clock in
// the future if needed (e.g. for asserting on the "Due in N days"
// chip with a deterministic reference time).
type timeT = time.Time

// timeNow returns the current time. Wrapped as a package-level
// var so the templ expression in list.templ can call it without
// importing "time" inline. Currently delegates to time.Now; kept
// as an indirection so it can be swapped in tests.
func timeNow() time.Time {
	return time.Now()
}
