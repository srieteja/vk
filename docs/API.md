# API Documentation

## Authentication
- POST /api/auth/userA/register
- POST /api/auth/userA/login
- POST /api/auth/userB/register
- POST /api/auth/userB/login

## UserA (Providers)
- PUT /api/userA/availability
- GET /api/userA/profile
- GET /api/userA/earnings

## UserB (Consumers)
- GET /api/userB/available-users
- POST /api/userB/payment/initiate
- POST /api/userB/payment/verify

## Calls
- POST /api/calls/initiate
- POST /api/calls/accept
- POST /api/calls/end
- GET /api/calls/token

## Health
- GET /health