# Strength Tracker

A simple strength tracking application built with Go, Echo, htmx, Tailwind CSS + Basecoat CSS, and SQLite. HTMX and Basecoat give a simple, clean semantic view layer. Built using an MVC architecture and focused on using as few dependencies as possible.

## Features

- User based authentication
- Track individual exercise sets with weight (in kg), set number, and notes
- Exercise history view showing all sets for a specific exercise

## Environment Variables

- `PORT` - Server port (default: 8080)
- `JWT_SECRET` - Secret used to sign auth info
- `DB_PATH` - SQLite/Turso database file path (default: strength_tracker.db)
- `TURSO_DATABASE_URL` - Remote URL from Turso
- `TURSO_AUTH_TOKEN` - Auth token for Turso

## To Do List
- [ ] Add a HTMX Spinner for form submissions
- [ ] Add in toasts for success/failure feedback
- [ ] Add a list of users in admin section
- [ ] Add a feedback form for users and then list it in the admin section
- [ ] 404 & 500 errors are returning JSON
