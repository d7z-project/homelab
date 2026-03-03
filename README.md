# Homelab

[中文版](README_ZH.md)

> [!WARNING]
> This project is a product of "vibe coding". It is not recommended for production environments.

A modern, secure infrastructure management system for homelab environments.

## Features

- DNS: Full domain & record management.
- RBAC: Fine-grained, wildcard-based access control.
- Audit: Comprehensive action tracking.
- Security: Root & SA authentication with TOTP.
- UI: Minimalist Swiss-style (M3 + Tailwind v4).

## Tech

- Backend: Go 1.21+, Chi, BoltDB.
- Frontend: Angular 17+, Material Design 3.

## Quick Start

```bash
make install
make all
```

- Backend: `cd backend && go run main.go`
- Frontend: `cd frontend && npm start`

## Development

Run `make backend-generate` after API changes to sync Swagger and Frontend clients.

---
*MIT License © 2026 Homelab Infrastructure System*
