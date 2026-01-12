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
  "location": "New York",
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
  "location": "New York",
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

### GET /api/auth/google/advocate
Initiate Google OAuth login for Advocate

**Description:** Redirects to Google OAuth consent screen for advocate authentication.

**Response:** HTTP 307 Redirect to Google OAuth

---

### GET /api/auth/google/client
Initiate Google OAuth login for Client

**Description:** Redirects to Google OAuth consent screen for client authentication.

**Response:** HTTP 307 Redirect to Google OAuth

---

### GET /api/auth/google/callback
Google OAuth callback endpoint

**Query Parameters:**
- `code`: Authorization code from Google
- `state`: State token containing user type

**Response (200 OK):**
```json
{
  "id": 1,
  "email": "user@example.com",
  "name": "John Doe",
  "uuid": "550e8400-e29b-41d4-a716-446655440000",
  "profile_image": "https://...",
  "token": "session_token_here",
  ...
}
```

**Error Response (400):**
```json
{
  "error": "error message"
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
  "location": "New York",
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
  "location": "New York",
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

### PUT /api/advocate/profile-image
Update Advocate profile image

**Headers:**
```
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "profile_image": "https://example.com/image.jpg"
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
  "profile_image": "https://example.com/image.jpg",
  ...
}
```

**Error Response (400/401/403):**
```json
{
  "error": "error message"
}
```

---

## Client (Consumers)

### PUT /api/client/profile-image
Update Client profile image

**Headers:**
```
Authorization: Bearer <token>
```

**Request Body:**
```json
{
  "profile_image": "https://example.com/image.jpg"
}
```

**Response (200 OK):**
```json
{
  "id": 1,
  "email": "client@example.com",
  "name": "Jane Smith",
  "uuid": "550e8400-e29b-41d4-a716-446655440000",
  "profile_image": "https://example.com/image.jpg",
  "balance": 0.0,
  ...
}
```

**Error Response (400/401/403):**
```json
{
  "error": "error message"
}
```

---

### GET /api/client/available-users
Get list of available Advocates (Providers) with optional filters

**Headers:**
```
Authorization: Bearer <token>
```

**Query Parameters:**
- `availability` (optional): Filter by availability status. Values: `"all"` (all advocates) or `"available"`/`"online"` (only available advocates). Default: `"available"`
- `location` (optional): Filter by location/city (case-insensitive partial match)
- `min_rate` (optional): Minimum hourly rate filter (numeric)
- `max_rate` (optional): Maximum hourly rate filter (numeric)

**Example Request:**
```
GET /api/client/available-users?availability=available&location=New York&min_rate=50&max_rate=100
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
      "location": "New York",
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

## LLM Service

### Chat Completion

`POST /api/llm/chat`

Get a chat completion from the LLM service.

#### Headers

```
Authorization: Bearer <your_jwt_token>
Content-Type: application/json
```

#### Request Body

```json
{
  "user_id": "user123",
  "message": "Hello, how are you?",
  "history": [
    {"role": "user", "content": "Hi there"},
    {"role": "assistant", "content": "Hello! How can I help you today?"}
  ]
}
```

#### Response

```json
{
  "response": "I'm doing well, thank you for asking! How can I assist you today?",
  "tokens_used": 42,
  "provider": "anthropic",
  "latency_ms": 1234.56,
  "cached": false
}
```

### Health Check

`GET /api/llm/health`

Check the health status of the LLM service.

#### Response

Success (200 OK):
```json
{
  "status": "healthy"
}
```

Error (503 Service Unavailable):
```json
{
  "status": "unhealthy",
  "error": "error message"
}
```
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