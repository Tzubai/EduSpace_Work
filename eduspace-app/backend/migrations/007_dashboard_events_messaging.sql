CREATE TABLE IF NOT EXISTS school_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    title VARCHAR(150) NOT NULL,
    description TEXT,
    event_type VARCHAR(30) NOT NULL DEFAULT 'school',
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ,
    audience VARCHAR(30) NOT NULL DEFAULT 'all',
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS parent_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    sender_user_id UUID NOT NULL REFERENCES users(id),
    recipient_guardian_id UUID NOT NULL,
    subject VARCHAR(150) NOT NULL,
    body TEXT NOT NULL,
    channel VARCHAR(20) NOT NULL DEFAULT 'in_app',
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    read_at TIMESTAMPTZ,
    FOREIGN KEY (institute_id, recipient_guardian_id) REFERENCES guardians(institute_id,id)
);

CREATE INDEX IF NOT EXISTS school_events_upcoming_idx
    ON school_events(institute_id,starts_at);
CREATE INDEX IF NOT EXISTS parent_messages_school_recipient_idx
    ON parent_messages(institute_id,recipient_guardian_id,sent_at DESC);
