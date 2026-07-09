package views

// MaxSetsPerExerciseEntry caps the number of sets accepted in a single
// create-exercise-entry submission. Enforced both server-side (in
// routes.parseExerciseEntrySets) and client-side (the form's "Add Set"
// button disables at this count). Lives here in the views package because
// the templ template embeds the value into its inline script, and views is
// a leaf of the dependency graph so nothing else can import it without
// creating a cycle.
const MaxSetsPerExerciseEntry = 10
