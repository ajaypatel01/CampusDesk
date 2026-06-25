CREATE TYPE broadcast_target AS ENUM ('manual', 'grade', 'all_parents', 'staff');
CREATE TYPE broadcast_status AS ENUM ('pending', 'sending', 'done', 'failed');
CREATE TYPE recipient_status AS ENUM ('sent', 'failed');

CREATE TABLE broadcasts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id       UUID NOT NULL REFERENCES schools(id),
    title           VARCHAR(255) NOT NULL,
    message         TEXT NOT NULL,
    target          broadcast_target NOT NULL DEFAULT 'manual',
    grade_level_id  UUID REFERENCES grade_levels(id),
    template_name   VARCHAR(100),
    template_lang   VARCHAR(20) DEFAULT 'en_US',
    is_template     BOOLEAN NOT NULL DEFAULT FALSE,
    sent_by         UUID REFERENCES users(id),
    total_count     INT NOT NULL DEFAULT 0,
    sent_count      INT NOT NULL DEFAULT 0,
    failed_count    INT NOT NULL DEFAULT 0,
    status          broadcast_status NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE broadcast_recipients (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    broadcast_id    UUID NOT NULL REFERENCES broadcasts(id) ON DELETE CASCADE,
    phone           VARCHAR(20) NOT NULL,
    name            VARCHAR(255),
    status          recipient_status NOT NULL DEFAULT 'sent',
    error_message   TEXT,
    sent_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_broadcast_recipients_broadcast_id ON broadcast_recipients(broadcast_id);
CREATE INDEX idx_broadcasts_school_id ON broadcasts(school_id);
