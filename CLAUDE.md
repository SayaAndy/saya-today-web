# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run

```bash
# Build (CGO required for SQLite)
CGO_ENABLED=1 go build -o sayana-web .

# Run (select config via ENVIRONMENT)
ENVIRONMENT=local ./sayana-web

# Generate Tailwind CSS
npx @tailwindcss/cli -i ./static/input.css -o ./static/output.css

# Docker build
docker build -t sayana-web .
```

Server listens on `:3000`. No Makefile. No test suite currently.

## Environment

Config selected by `ENVIRONMENT` var (`local`/`stage`/`prod`) → `config/config.{env}.yaml`.

Key env vars: `B2_KEY_ID`, `B2_APPLICATION_KEY`, `MAIL_HOST`, `MAIL_ADDRESS`, `MAIL_USERNAME`, `MAIL_PASSWORD`, `MAIL_SALT`. See `.envrc` for local setup.

## Architecture

**Go + Fiber v2** web app serving a multi-language blog/travel site with SSR.

### Request Flow
1. Fiber router with middleware (CORS, compression, ETags, trailing slash)
2. Routes auto-register via `init()` in `internal/router/handlers/` — each handler file adds to `router.Routes` global slice
3. Handlers implement `Route` interface: `Filter()`, `IsTemplated()`, `Render()`, `RenderBody()`
4. Templated routes render via `TemplateManager` (Go `html/template`) with layout composition
5. Pages cached in Ristretto (512MB in-memory) with TTL; segments loadable via `/api/v1/general-page/:part`

### Storage
- **Blog content:** Markdown files with YAML frontmatter on B2 or S3 (configurable via `blog.Client` interface)
- **Database:** SQLite3 via `database/sql` (no ORM). Migrations in `migrations/` via golang-migrate
- **Tables:** `blog_likes`, `blog_views`, `user_email_table`, subscription tables
- **Session cache:** SQLite-backed `ClientCache` for client IDs/preferences

### Key Subsystems
- **Markdown rendering:** Goldmark with custom extensions — `TailwindExtension` (applies Tailwind classes) and `GLightboxExtension` (image galleries)
- **Mailer:** SMTP email with verification codes, subscription management
- **BlogTrigger:** gocron scheduler that detects new posts and sends notification emails
- **i18n:** Language in URL path (`/en/`, `/ru/`), locale YAML files in `locale/`

### Handler Pattern
Handlers in `internal/router/handlers/` follow naming convention matching routes:
- `lang-blog.go` → `/:lang/blog/` pages
- `api-v1-like-put.go` → `/api/v1/like` PUT endpoint
- Each handler self-registers in `init()`, no central route table to maintain

## Deployment

CI/CD via GitHub Actions → Docker image to ghcr.io → Ansible playbook (`deploy/`).
- `stage` branch → stage environment
- Version tags → prod environment
