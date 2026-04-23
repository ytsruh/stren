# Agent Instructions for Strength Tracker

## Application & Business Logic Instructions
- Weight doesn't have KG or lbs and is stored as a number only. App is currently set to KG but ultimately will let a user determine this via a custom setting in their profile
- Every entry created for a workout is a single set. A set has a number of repititions (reps) & a weight 

## Code & Development Instructions
- Use Makefile scripts where possible
- Add and create unit tests wherever possible
- templ is used for templating lanaguage & must be regenerated before running the application. Generated files: `internal/views/*_templ.go` (don't edit directly)
- Avoid adding new dependancies. If they are needed, discuss with user/developer first. If agreed to use then always wrap dependancies in packages so they are isolated and easily edited, removed, updated without codebase wide changes.
- Add documentation/annotations to code wherever possible to explain what functions, structs etc do.
- Make use of `go doc` Go command to fetch documentation for packages/modules

## Technology & Architecture
- **Tech stack**: Go 1.25+, Echo, Templ, SQLite (modernc.org/sqlite), htmx, Oat CSS
- **Entry point**: `cmd/main.go`
- **HTTP handlers**: `internal/handlers/handlers.go` (Echo framework)
- **Repository**: `internal/models/repository.go` (CRUD operations)
- **Models**: `internal/models/models.go` (ExerciseType, ExerciseEntry)
- **Database**: `internal/db/db.go` (SQLite wrapper + migrations)
- **Templates**: `internal/views/*.templ` (Templ HTML templates)

## Environment Variables

- `PORT` - Server port (default: 8080)
- `DB_PATH` - SQLite file path (default: strength_tracker.db)
