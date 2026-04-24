# Kai

Local-first productivity command layer. Quickest, Fastest, Easiest way to access your Local AI Agent. 

In Progress: Google, Outlook, Apple Integrations

## Structure

- `apps/desktop`: React + TypeScript desktop UI prototype
- `services/backend`: thin Go backend skeleton for auth/health infrastructure

## Run

```bash
npm install
npm run dev
```

Backend:

```bash
cd services/backend
GOCACHE="$(cd ../.. && pwd)/.gocache" go run ./cmd/api
```
