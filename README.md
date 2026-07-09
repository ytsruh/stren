# Strength Tracker

A simple strength tracking application built with Go, Echo, htmx, Tailwind CSS + Basecoat CSS, and SQLite/Turso. HTMX and Basecoat give a simple, clean semantic view layer. Built using an MVC architecture and focused on using as few dependencies as possible.

## Features

- Installable PWA
- User based authentication
- Track individual exercise sets with weight (in kg), set number, and notes
- Exercise history view showing all sets for a specific exercise
- Rest/countdown timer & an EMOM timer
- Weight tracking
- User notifications (push & email)

## Environment Variables

- `PORT` - Server port (default: 8080)
- `JWT_SECRET` - Secret used to sign auth info
- `DB_PATH` - SQLite/Turso database file path (default: strength_tracker.db)
- `TURSO_DATABASE_URL` - Remote URL from Turso
- `TURSO_AUTH_TOKEN` - Auth token for Turso
- `STORAGE_ENDPOINT` - Storage endpoint URL
- `STORAGE_ACCESS_KEY` - Storage access key
- `STORAGE_SECRET_KEY` - Storage secret key
- `STORAGE_BUCKET` - Storage bucket name
- `STORAGE_PUBLIC_URL` - Public URL for the storage bucket
- `CLOUDFLARE_EMAIL_TOKEN` - Cloudflare email token
- `PUBLIC_URL` - Public URL for the app
