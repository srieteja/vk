CREATE TABLE IF NOT EXISTS advocates (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    availability VARCHAR(50) DEFAULT 'available',
    uuid UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    location VARCHAR(255),
    profile_image TEXT,
    bio TEXT,
    earnings DECIMAL(10, 2) DEFAULT 0.00,
    hourly_rate DECIMAL(10, 2) DEFAULT 0.00,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS clients (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    uuid UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    profile_image TEXT,
    balance DECIMAL(10, 2) DEFAULT 0.00,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS calls (
    id SERIAL PRIMARY KEY,
    caller_id INTEGER NOT NULL,
    caller_type VARCHAR(50) NOT NULL,
    receiver_id INTEGER NOT NULL,
    receiver_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) DEFAULT 'initiated',
    duration BIGINT DEFAULT 0,
    charge_amount DECIMAL(10, 2),
    payment_id INTEGER,
    started_at TIMESTAMP,
    ended_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS payments (
    id SERIAL PRIMARY KEY,
    client_id INTEGER NOT NULL,
    advocate_id INTEGER NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    transaction_id VARCHAR(255) UNIQUE NOT NULL,
    payment_gateway VARCHAR(50) NOT NULL,
    advocate_commission DECIMAL(10, 2) DEFAULT 0.00,
    platform_fee DECIMAL(10, 2) DEFAULT 0.00,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    user_type VARCHAR(50) NOT NULL,
    token VARCHAR(500) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_advocates_email ON advocates(email);
CREATE INDEX idx_clients_email ON clients(email);
CREATE INDEX idx_calls_caller ON calls(caller_id);
CREATE INDEX idx_calls_receiver ON calls(receiver_id);
CREATE INDEX idx_payments_client ON payments(client_id);
CREATE INDEX idx_payments_advocate ON payments(advocate_id);