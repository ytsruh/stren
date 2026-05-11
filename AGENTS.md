# Agent Instructions for Strength Tracker

## Application & Business Logic Instructions
- Weight doesn't have KG or lbs and is stored as a number only. App is currently set to KG but ultimately will let a user determine this via a custom setting in their profile
- Every entry created for a workout is a single set. A set has a number of repititions (reps) & a weight 

## Code & Development Instructions
- Use Makefile scripts where possible
- Add and create unit tests wherever possible
- Follow go best practices at all time. Particulary using interfaces, packages should create interfaces that they accept.
- templ is used for templating lanaguage & must be regenerated before running the application. Generated files: `internal/views/*_templ.go` (don't edit directly)
- Avoid adding new dependancies. If they are needed, discuss with user/developer first. If agreed to use then always wrap dependancies in packages so they are isolated and easily edited, removed, updated without codebase wide changes.
- Add documentation/annotations to code wherever possible to explain what functions, structs etc do.
- Make use of `go doc` Go command to fetch documentation for packages/modules

## Technology & Architecture
- **Tech stack**: Go 1.25+, Echo, Templ, Turso Sync (turso.tech/database/tursogo), htmx, Basecoat CSS
- **Entry point**: `cmd/main.go`
- **HTTP routes**: `internal/routes/*.go` (Echo framework — request parsing, rendering, redirects, middleware)
- **Controllers**: `internal/controllers/*.go` (business logic — auth, entry CRUD orchestration)
- **Repository interfaces**: `internal/models/repository.go` (CRUD operations)
- **Models**: `internal/models/models.go` (ExerciseType, ExerciseEntry, User)
- **Database**: `internal/db/db.go` (Turso Sync wrapper with local-first reads/writes and background cloud sync)
- **Migrations**: `internal/db/migrations/*.sql` managed by [goose](https://github.com/pressly/goose) and embedded in the binary via `//go:embed`
- **Templates**: `internal/views/*.templ` (Templ HTML templates)

## Environment Variables

All environment variables are strictly required and validated on server startup.
Create a `.env` file based on `.env.example` before running the application.

- `PORT` - Server port
- `DB_PATH` - Local database file path (e.g., `/data/strength_tracker.db`)
- `TURSO_DATABASE_URL` - Turso Cloud database URL (`libsql://...`)
- `TURSO_AUTH_TOKEN` - Turso Cloud authentication token
