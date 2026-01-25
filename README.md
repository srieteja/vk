# Enterprise Go Backend API

Production-ready Go backend API with dual user system, video calling, and payments.

## Quick Start

```bash
cp .env.example .env
go mod download
make docker-up
make run
```

Server: http://localhost:8080

Postgres runs `migrations/init.sql` automatically on first boot (empty volume).

## Features

✅ Dual user system (Providers & Consumers)
✅ Email/Password + Google OAuth
✅ P2P Video Calling (WebRTC/VP9)
✅ Payment Integration (Stripe & Razorpay)
✅ Availability Management
✅ Comprehensive tests
✅ Docker ready

## Commands

```bash
make run               # Start server
make worker            # Run outbox worker
make test              # Run tests
make test-coverage     # Coverage
make docker-up         # Start Docker
```

## Documentation

- [Setup Guide](SETUP_GUIDE.md)
- [API Docs](docs/API.md)
- [Architecture](docs/ARCHITECTURE.md)
