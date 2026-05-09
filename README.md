# Strength Tracker

A simple strength tracking application built with Go, Echo, htmx, Tailwind CSS + Basecoat CSS, and SQLite. HTMX and Basecoat give a simple, clean semantic view layer. Built using an MVC architecture and focused on using as few dependencies as possible.

## Features

- Track individual exercise sets with weight (in kg), set number, and notes
- User based authentication
- Exercise history view showing all sets for a specific exercise

## Styling

The app uses **Tailwind CSS standalone CLI + Basecoat CSS** for styling. Tailwind provides utility classes while Basecoat provides component classes.

### Setup

CSS is compiled from source using the Tailwind standalone CLI:

```bash
# Build CSS
make css-build

# Watch for changes during development
make css-watch
```

### Key Classes

- `btn`, `btn-primary`, `btn-outline`, `btn-destructive` - buttons
- `card`, `card-body` - cards
- `table` - tables
- `input`, `select`, `textarea` - form inputs
- `alert`, `alert-destructive` - alerts/toasts
- `dialog` - modal dialogs
- `dropdown-menu` - dropdown menus

Dark mode is handled via the `.dark` class on `<html>`, controlled by localStorage and `prefers-color-scheme`.

## Environment Variables

- `PORT` - Server port (default: 8080)
- `JWT_SECRET` - Secret used to sign auth info
- `DB_PATH` - SQLite/Turso database file path (default: strength_tracker.db)
- `TURSO_DATABASE_URL` - Remote URL from Turso
- `TURSO_AUTH_TOKEN` - Auth token for Turso
