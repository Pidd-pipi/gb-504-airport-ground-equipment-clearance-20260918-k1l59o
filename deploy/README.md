# Deployment

Run `docker compose up -d --build` from the repository root. The stack waits for PostgreSQL and Redis health checks before starting the API, then waits for `/healthz` before exposing the frontend.

Use `docker compose down` to stop the stack, or `docker compose down -v` to remove local demo data as well.
