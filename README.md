# Kai

Local-first productivity command layer prototype based on the V1 design document.

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
go run ./cmd/api
```

