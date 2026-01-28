CREATE INDEX IF NOT EXISTS idx_calls_receiver_type_status
    ON calls(receiver_id, receiver_type, status);

CREATE INDEX IF NOT EXISTS idx_calls_receiver_schedule
    ON calls(receiver_id, receiver_type, scheduled_at, started_at);

CREATE INDEX IF NOT EXISTS idx_outbox_status_id
    ON outbox_events(status, id);

CREATE INDEX IF NOT EXISTS idx_advocates_availability_rate
    ON advocates(availability, hourly_rate);
