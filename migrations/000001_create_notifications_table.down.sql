CREATE TABLE IF NOT EXISTS notifications (
    id          UUID PRIMARY KEY,
    channel     TEXT NOT NULL,
    recipient   TEXT NOT NULL,
    message     TEXT NOT NULL,
    send_at     TIMESTAMP NOT NULL,
    status      TEXT NOT NULL, -- pending, sent, failed, canceled
    created_at  TIMESTAMP NOT NULL DEFAULT now(),
    updated_at  TIMESTAMP NOT NULL DEFAULT now()
);