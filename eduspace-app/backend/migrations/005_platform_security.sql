-- EduSpace platform and security foundation.
ALTER TABLE institute_members
    ADD CONSTRAINT institute_members_role_check_v2
    CHECK (role IN ('owner','admin','bursar','teacher','parent','student'));

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS token_version INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE institutes
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS trial_ends_at TIMESTAMPTZ;

ALTER TABLE institutes DROP CONSTRAINT IF EXISTS institutes_status_check;
ALTER TABLE institutes
    ADD CONSTRAINT institutes_status_check CHECK (status IN ('pending','active','suspended'));

ALTER TABLE subscriptions
    ADD COLUMN IF NOT EXISTS trial_ends_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS current_period_ends_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS gateway_customer_id TEXT,
    ADD COLUMN IF NOT EXISTS gateway_subscription_id TEXT,
    ADD COLUMN IF NOT EXISTS max_students INTEGER,
    ADD COLUMN IF NOT EXISTS max_staff INTEGER;

CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID NOT NULL REFERENCES institutes(id),
    user_id UUID NOT NULL REFERENCES users(id),
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS login_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID REFERENCES institutes(id),
    email TEXT,
    ip_address INET,
    succeeded BOOLEAN NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institute_id UUID REFERENCES institutes(id),
    actor_user_id UUID REFERENCES users(id),
    action TEXT NOT NULL,
    entity_type TEXT,
    entity_id TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS fee_invoice_once_per_period
    ON fee_invoices (institute_id, student_id, term_id, description);

CREATE UNIQUE INDEX IF NOT EXISTS teacher_assignment_once
    ON teacher_assignments (institute_id, user_id, class_id, subject_id);

CREATE UNIQUE INDEX IF NOT EXISTS payment_reference_unique
    ON payments (institute_id, reference)
    WHERE reference IS NOT NULL AND btrim(reference) <> '';

CREATE INDEX IF NOT EXISTS password_reset_active_idx
    ON password_reset_tokens (user_id, expires_at) WHERE used_at IS NULL;
CREATE INDEX IF NOT EXISTS login_attempts_ip_time_idx
    ON login_attempts (ip_address, created_at DESC);
CREATE INDEX IF NOT EXISTS audit_events_school_time_idx
    ON audit_events (institute_id, created_at DESC);
