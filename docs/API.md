# API Documentation

## Authentication

### POST /api/auth/advocate/register
Register a new Advocate (Provider)

**Request Body:**
```json
{
  "email": "string (required)",
  "password": "string (required)",
  "name": "string (required)"
}
```

**Response (200 OK):**
```json
{
  "id": 1,
  "email": "advocate@example.com",
  "name": "John Doe",
  "availability": "available",
  "uuid": "550e8400-e29b-41d4-a716-446655440000",
  "profile_image": "",
  "bio": "",
  "earnings": 0.0,
  "hourly_rate": 0.0,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Error Response (400/500):**
```json
{
  "error": "error message"
}
```

---

### POST /api/auth/advocate/login
Login as Advocate (Provider)

**Request Body:**
```json
{
  "email": "string (required)",
  "password": "string (required)"
}
```

**Response (200 OK):**
```json
{
  "id": 1,
  "email": "advocate@example.com",
  "name": "John Doe",
  "availability": "available",
  "uuid": "550e8400-e29b-41d4-a716-446655440000",
  "profile_image": "",
  "bio": "",
  "earnings": 0.0,
  "hourly_rate": 0.0,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z",
  "token": "session_token_here"
}
```

**Error Response (401/500):**
```json
{
  "error": "invalid credentials"
}
```

---

### POST /api/auth/client/register
Register a new Client (Consumer)

**Request Body:**
```json
{
  "email": "string (required)",
  "password": "string (required)",
  "name": "string (required)"
}
```

**Response (200 OK):**
```json
{
  "id": 1,
  "email": "client@example.com",
  "name": "Jane Smith",
  "uuid": "550e8400-e29b-41d4-a716-446655440000",
  "profile_image": "",
  "balance": 0.0,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Error Response (400/500):**
```json
{
  "error": "error message"
}
```

---

### POST /api/auth/client/login
Login as Client (Consumer)

**Request Body:**
```json
{
  "email": "string (required)",
  "password": "string (required)"
}
```

**Response (200 OK):**
```json
{
  "id": 1,
  "email": "client@example.com",
  "name": "Jane Smith",
  "uuid": "550e8400-e29b-41d4-a716-446655440000",
  "profile_image": "",
  "balance": 0.0,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z",
  "token": "session_token_here"
}
```

**Error Response (401/500):**
```json
{
  "error": "invalid credentials"
}
```

---

## Advocate (Providers)

### PUT /api/advocate/availability
Update Advocate availability status

**Request Body:**
```json
{
  "availability": "string (available|busy|offline)"
}
```

**Response (200 OK):**
```json
{
  "id": 1,
  "email": "advocate@example.com",
  "name": "John Doe",
  "availability": "available",
  "uuid": "550e8400-e29b-41d4-a716-446655440000",
  "profile_image": "",
  "bio": "",
  "earnings": 0.0,
  "hourly_rate": 0.0,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Error Response (400/401/500):**
```json
{
  "error": "error message"
}
```

---

### GET /api/advocate/profile
Get Advocate profile information

**Headers:**
```
Authorization: Bearer <token>
```

**Response (200 OK):**
```json
{
  "id": 1,
  "email": "advocate@example.com",
  "name": "John Doe",
  "availability": "available",
  "uuid": "550e8400-e29b-41d4-a716-446655440000",
  "profile_image": "",
  "bio": "",
  "earnings": 0.0,
  "hourly_rate": 0.0,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Error Response (401/404/500):**
```json
{
  "error": "error message"
}
```

---

### GET /api/advocate/earnings
Get Advocate earnings information

**Headers:**
```
Authorization: Bearer <token>
```

**Response (200 OK):**
```json
{
  "earnings": 1500.50,
  "total_calls": 25,
  "hourly_rate": 60.0
}
```

**Error Response (401/500):**
```json
{
  "error": "error message"
}
```

---

## Client (Consumers)

### GET /api/client/available-users
Get list of available Advocates (Providers)

**Headers:**
```
Authorization: Bearer <token>
```

**Response (200 OK):**
```json
{
  "users": [
    {
      "id": 1,
      "email": "advocate@example.com",
      "name": "John Doe",
      "availability": "available",
      "profile_image": "",
      "bio": "",
      "hourly_rate": 60.0
    }
  ],
  "count": 1
}
```

**Error Response (401/500):**
```json
{
  "error": "error message"
}
```

---

### POST /api/client/payment/initiate
Initiate a payment

**Headers:**
```
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "advocate_id": 1,
  "amount": 100.0
}
```

**Response (200 OK):**
```json
{
  "id": 1,
  "client_id": 1,
  "advocate_id": 1,
  "amount": 100.0,
  "status": "pending",
  "transaction_id": "txn_1",
  "payment_gateway": "stripe",
  "advocate_commission": 0.0,
  "platform_fee": 0.0,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Error Response (400/401/500):**
```json
{
  "error": "invalid amount"
}
```

---

### POST /api/client/payment/verify
Verify a payment transaction

**Headers:**
```
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "transaction_id": "txn_1"
}
```

**Response (200 OK):**
```json
{
  "id": 1,
  "client_id": 1,
  "advocate_id": 1,
  "amount": 100.0,
  "status": "completed",
  "transaction_id": "txn_1",
  "payment_gateway": "stripe",
  "advocate_commission": 0.0,
  "platform_fee": 0.0,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Error Response (400/401/404/500):**
```json
{
  "error": "Invalid transaction ID"
}
```

---

## Calls

### POST /api/calls/initiate
Initiate a call

**Headers:**
```
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "receiver_id": 1
}
```

**Response (200 OK):**
```json
{
  "id": 1,
  "caller_id": 1,
  "caller_type": "client",
  "receiver_id": 1,
  "receiver_type": "advocate",
  "status": "initiated",
  "duration": 0,
  "charge_amount": 0.0,
  "payment_id": null,
  "started_at": null,
  "ended_at": null,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Error Response (400/401/500):**
```json
{
  "error": "invalid caller or receiver ID"
}
```

---

### POST /api/calls/accept
Accept an incoming call

**Headers:**
```
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "call_id": 1
}
```

**Response (200 OK):**
```json
{
  "id": 1,
  "caller_id": 1,
  "caller_type": "client",
  "receiver_id": 1,
  "receiver_type": "advocate",
  "status": "accepted",
  "duration": 0,
  "charge_amount": 0.0,
  "payment_id": null,
  "started_at": "2024-01-01T00:00:00Z",
  "ended_at": null,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

**Error Response (400/401/404/500):**
```json
{
  "error": "invalid call ID"
}
```

---

### POST /api/calls/end
End an active call

**Headers:**
```
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "call_id": 1
}
```

**Response (200 OK):**
```json
{
  "id": 1,
  "caller_id": 1,
  "caller_type": "client",
  "receiver_id": 1,
  "receiver_type": "advocate",
  "status": "completed",
  "duration": 300,
  "charge_amount": 50.0,
  "payment_id": 1,
  "started_at": "2024-01-01T00:00:00Z",
  "ended_at": "2024-01-01T00:05:00Z",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:05:00Z"
}
```

**Error Response (400/401/404/500):**
```json
{
  "error": "invalid call ID"
}
```

---

### GET /api/calls/token
Get WebRTC token for a call

**Headers:**
```
Authorization: Bearer <token>
```

**Query Parameters:**
```
call_id: integer (required)
```

**Response (200 OK):**
```json
{
  "token": "webrtc_token_here",
  "call_id": 1,
  "expires_at": "2024-01-01T01:00:00Z"
}
```

**Error Response (400/401/404/500):**
```json
{
  "error": "error message"
}
```

---

## Health

### GET /health
Health check endpoint

**Response (200 OK):**
```json
{
  "status": "ok",
  "service": "enterprise-api"
}
```