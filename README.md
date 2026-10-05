# auth-v2

Go/Fiber replacement for the GDG Babcock platform authentication backend. It removes Firebase and issues platform JWTs with URI audiences.

Copy `.env.example` to `.env`, set a real `JWT_HMAC_SECRET`, then run `docker compose up --build`. The API listens on `http://localhost:8080`; Mailpit is at `http://localhost:8025`.

Endpoints currently provided: `POST /auth/register`, `POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout`, `GET /auth/me`, `GET /auth/verify`, and `GET /health`.

Tokens use `iss=JWT_ISSUER` and an audience chosen from `JWT_AUDIENCES`. The receiving service must validate its own URI. Firebase tokens are intentionally not accepted. Existing users retain their UUIDs through the migration schema; users must establish a native password through the reset flow or link Google during cutover.
