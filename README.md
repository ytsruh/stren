# Strength Tracker

A simple strength tracking application built with Go, Echo, htmx, Oat UI, and SQLite. HTMX and OatCSS give a simple, clean semantic view layer. Built using an MVC architecture and focussed on using as few dependancies as possible.

## Features

- Track individual exercise sets with weight (in kg), set number, and notes
- User based authentication
- Exercise history view showing all sets for a specific exercise

## Environment Variables

- `PORT` - Server port (default: 8080)
- `JWT_SECRET` - Secret used to sign auth info
- `DB_PATH` - SQLite/Turso database file path (default: strength_tracker.db)
- `TURSO_DATABASE_URL` - Remote URL from Turso
- `TURSO_AUTH_TOKEN` - Auth token for Turso

## To Do List

#### Server
- [x] Setup air
- [x] Add sqlc as DB lib 
- [x] Add goose as Migration lib
- [x] Add unit tests
- [x] Turn into PWA
- [x] Add Dockerfile
- [x] Add Auth and scope data to users
- [x] Switch to Turso
- [x] Make a utils file to check envs at startup
- [x] Add validation library for user input

#### UI
- [x] Better buttons. Some buttons are just links, not actual buttons
- [x] Delete entry modal instead of confirmation
- [ ] Date formats
- [ ] After entry redirect to dashbaord
- [ ] Exercises pages have cards that are not formatted as Cards
