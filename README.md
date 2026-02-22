# resy-scheduler — per-user credentials (build-fixed)

This build implements:
- username/password login
- per-user provider credentials stored in Postgres (encrypted at rest)
- redirect to `/credentials` only when missing

It intentionally **does not** compile provider HTTP clients yet; those will be added back once wired to per-user creds.

## Run (podman)
```bash
cp .env.example .env
podman-compose up --build
make user-add USERNAME=james PASSWORD='changeme'
```
Then visit http://localhost:8080/login
