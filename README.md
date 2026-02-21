# resy-scheduler — per-user provider credentials (Go 1.25.5)

This build adds **per-user** Resy/OpenTable credential storage with a simple UI.

## What you get
- Username/password login
- On login, the app checks whether you have provider credentials in the DB
  - If missing, you are redirected to `/credentials`
  - If present, you go to `/`
- Credentials are stored **encrypted in Postgres** (AES-256-GCM) using `CRED_ENC_KEY`

## Run locally (podman)
```bash
cp .env.example .env
podman-compose up --build
```

Create a user (from another terminal):
```bash
make user-add USERNAME=james PASSWORD='changeme-now'
```

Open:
- http://localhost:8080/login

## Getting the tokens (short version shown in the UI too)
- **Resy**: log into resy.com → DevTools → Network → click request to `api.resy.com` → copy `Authorization` token and API key header (often `x-api-key`).
- **OpenTable**: log into opentable.com → DevTools → Network → click request to `/dapi` → copy `x-csrf-token` header.

> You only need to do this once per user; it’s saved in the database and the app only asks again if missing.

## Production notes
- Put `DATABASE_URL`, session keys, and `CRED_ENC_KEY` into Kubernetes Secrets.
- If you change `CRED_ENC_KEY`, previously stored credentials cannot be decrypted.
