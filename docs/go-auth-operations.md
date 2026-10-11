# Go auth deployment and cutover

## Configuration

The Go process requires `DATABASE_URL`, `FIREBASE_PROJECT_ID`, `JWT_SECRET` (at least 32 characters), and `CORS_ORIGINS` (a comma-separated list of exact origins). It accepts Firebase Admin credentials from `FIREBASE_SERVICE_ACCOUNT`, `FIREBASE_SERVICE_ACCOUNT_PATH`, or Application Default Credentials. `HTTP_PORT` defaults to `8080`; `JWT_ACCESS_TTL` and `JWT_REFRESH_TTL` default to `24h` and `168h`. Keep the credentials and signing secret outside the repository. Use the existing JWT signing secret during the parallel run so previously issued platform access tokens remain valid.

Local Docker Compose reads `.env`. It uses `FIREBASE_SERVICE_ACCOUNT` when set, or a read-only key file at `.secrets/firebase-service-account.json`. The container path is `/run/secrets/firebase-service-account.json`. ADC from a host login is not automatically available in the container.

The service reads the existing PostgreSQL auth tables through GORM. It never runs `AutoMigrate`. `database/auth.sql` is for a fresh disposable database, not for replay against production. Back up the production database before cutover, and compare counts and representative UUIDs between Express and Go. The four included tables are `users`, `refresh_tokens`, `user_sessions`, and `auth_audit_log`. All other tables remain with Express.

## Routing

Run Go and Express concurrently. Route only `/auth/*` and `/admin/users*` to Go while other paths still reach Express; set real service addresses in the deployment router. `/health` for Go should be exposed on its own internal service address. Do not replace the entire `auth.gdgbabcock.com` backend with the Go container.

Before moving `/auth/me` or `/auth/profile`, provide `activity`, `certificates`, and `team_membership` from another service or accept the resulting website UI change explicitly. Those fields currently require non-auth tables and are not implemented in Go. The current website refresh helper also expects a `refresh_token` in the response; Go includes the still-active refresh token alongside the new access token to preserve the client session. New tokens contain `token_use` to distinguish access from refresh; the verifier also accepts the legacy claim shapes during the migration.

## Staging checks and rollback

1. Use a database copy and a Firebase test project or Auth emulator. Confirm returning users retain `users.id`, new users receive one row, inactive users are denied, and same-email/different-UID logins are rejected for account-linking review.
2. Compare Go and Express responses for login, refresh, logout, profile, verify, and admin user operations. Confirm an old access token works on Go, a Go access token works on Express, and a refresh token cannot call a protected Go route.
3. Confirm `/events`, `/radar`, `/team`, and `/analytics` still reach Express through the router. Check the website's profile activity and certificate displays before moving `/auth/me`.
4. Switch one auth path at a time and observe login failures, identity conflicts, token refreshes, disabled-user denials, and database errors. Keep a route-level rollback to Express during the observation window.

No production routing or data migration is performed by this repository change. The account inventory, staging comparison, external profile enrichment owner, and live cutover remain deployment tasks.
