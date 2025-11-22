# Enterprise Go Backend API

Production-ready Go backend API with dual user system, video calling, and payments.

## Quick Start

```bash
python3 generate_project.py  # Generate complete project
cd enterprise-api
cp .env.example .env
go mod download
docker-compose up -d postgres
sleep 5
psql -U enterprise_user -d enterprise_db -h localhost < migrations/init.sql
go run cmd/main.go
```

Server: http://localhost:8080

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
make test              # Run tests
make test-coverage     # Coverage
make docker-up         # Start Docker
```

## Documentation

- [Setup Guide](SETUP_GUIDE.md)
- [API Docs](docs/API.md)
- [Architecture](docs/ARCHITECTURE.md)