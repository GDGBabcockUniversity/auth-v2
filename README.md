# auth-v2

Authentication service for the GDG Babcock platform, built with Go and Fiber. It provides email/password and Google sign-in, then issues JWTs for platform services using URI-based audiences.

## Setup

### Requirements

- Docker and Docker Compose
- Go 1.27+ (only needed to run the API or tests outside Docker)

### Configure and run locally

1. Create your local environment file:

   ```sh
   cp .env.example .env
   ```

2. Set a secure `JWT_HMAC_SECRET` in `.env` (at least 32 characters). Review `JWT_AUDIENCES` and, if needed, add your Google OAuth credentials.

3. Start PostgreSQL, Mailpit, and the API:

   ```sh
   docker compose up --build
   ```

4. On a new database, apply the migrations from another terminal:

   ```sh
   docker compose run --rm api migrate
   ```

The API is available at `http://localhost:8080`, the health check is at `http://localhost:8080/health`, and Mailpit is available at `http://localhost:8025`.

To run without Docker, make sure `DATABASE_URL` points to a running PostgreSQL instance, then use `go run ./cmd/api`. Run the test suite with `go test ./...`.

## Features

### Done

- Email/password registration and login
- JWT access tokens with refresh-token rotation
- Authenticated user profile and token validation endpoints
- Google OAuth sign-in, including verified-email account linking
- Role claims in JWTs
- PostgreSQL-backed users and refresh tokens

### Upcoming

- Admin dashboard UI for user and account administration
- Password-reset flow for email/password accounts
- Email verification for email/password registration
