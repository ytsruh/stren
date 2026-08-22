# Strength Tracker

A simple strength tracking application built with Go, Echo, htmx, Tailwind CSS + Basecoat CSS, and SQLite/Turso. HTMX and Basecoat give a simple, clean semantic view layer. Built using an MVC architecture and focused on using as few dependencies as possible.

## Product shape

The iOS app (`client/`) is the primary user surface — logging sets, weigh-ins,
goals, timers, and feedback all happen there over the `/api/v1` JSON namespace.

This web app is a limited companion surface:
- Read-only dashboard (recent activity + 7-day stats)
- Per-exercise history and chart pages
- Profile settings (name, appearance, target weight & unit, weight-reminder preferences)
- Weight data export (`/weight/export`, linked from the profile)
- Admin: exercise management (+ image upload), users, feedback inbox
- Authentication (register / login / password reset)

## Features

- User based authentication
- Read-only exercise dashboard & history views
- Weight reminder notifications (email only, hourly cron)
- Weight data ZIP export

## Environment Variables

All variables are required and validated at startup. See `.env.example`.

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
