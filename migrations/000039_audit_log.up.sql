-- Audit log: one row per inserted, updated or deleted row in any table,
-- with who did it. The API sets the app.* session settings for each write
-- request (see internal/platform/database), so the trigger knows the user;
-- changes made outside the API (scripts, migrations) are logged without one.
CREATE TABLE audit_log (
    id          BIGSERIAL PRIMARY KEY,
    at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    table_name  TEXT NOT NULL,
    action      TEXT NOT NULL CHECK (action IN ('INSERT', 'UPDATE', 'DELETE')),
    row_id      TEXT,
    school_id   UUID,
    user_id     UUID,
    user_role   TEXT,
    request     TEXT,  -- e.g. "POST /api/v1/exam-marks/bulk"
    request_id  TEXT,
    ip          TEXT,
    user_agent  TEXT,
    -- INSERT: new row; DELETE: old row; UPDATE: only the columns that changed.
    old_data    JSONB,
    new_data    JSONB
);
CREATE INDEX audit_log_at_idx ON audit_log (at DESC);
CREATE INDEX audit_log_school_at_idx ON audit_log (school_id, at DESC);
CREATE INDEX audit_log_user_at_idx ON audit_log (user_id, at DESC);
CREATE INDEX audit_log_row_idx ON audit_log (table_name, row_id);

-- Columns never copied into the log.
CREATE FUNCTION audit_strip(j JSONB) RETURNS JSONB LANGUAGE sql IMMUTABLE AS $$
    SELECT j - 'password_hash' - 'token_version' - 'aadhar_number'
$$;

CREATE FUNCTION audit_row_change() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
    o JSONB;
    n JSONB;
    sid TEXT;
BEGIN
    IF TG_OP <> 'INSERT' THEN o := to_jsonb(OLD); END IF;
    IF TG_OP <> 'DELETE' THEN n := to_jsonb(NEW); END IF;

    IF TG_OP = 'UPDATE' THEN
        -- Keep only what changed; skip updates that changed nothing.
        SELECT jsonb_object_agg(key, o -> key), jsonb_object_agg(key, value)
          INTO o, n
          FROM jsonb_each(to_jsonb(NEW))
         WHERE key <> 'updated_at' AND (to_jsonb(OLD) -> key) IS DISTINCT FROM value;
        IF n IS NULL THEN
            RETURN NULL;
        END IF;
    END IF;

    -- Aadhar numbers and passwords are recorded as changed, never stored.
    IF o ? 'password_hash' OR n ? 'password_hash' THEN
        n := COALESCE(n, '{}'::jsonb) || '{"password": "changed"}';
    END IF;
    IF (o ? 'aadhar_number' OR n ? 'aadhar_number') AND TG_OP = 'UPDATE' THEN
        n := n || '{"aadhar": "changed"}';
    END IF;

    sid := COALESCE(to_jsonb(NEW) ->> 'school_id', to_jsonb(OLD) ->> 'school_id',
                    NULLIF(current_setting('app.school_id', true), ''));

    INSERT INTO audit_log (table_name, action, row_id, school_id, user_id, user_role,
                           request, request_id, ip, user_agent, old_data, new_data)
    VALUES (
        TG_TABLE_NAME, TG_OP,
        COALESCE(to_jsonb(NEW) ->> 'id', to_jsonb(OLD) ->> 'id'),
        sid::uuid,
        NULLIF(current_setting('app.user_id', true), '')::uuid,
        NULLIF(current_setting('app.user_role', true), ''),
        NULLIF(current_setting('app.request', true), ''),
        NULLIF(current_setting('app.request_id', true), ''),
        NULLIF(current_setting('app.ip', true), ''),
        NULLIF(current_setting('app.user_agent', true), ''),
        audit_strip(o), audit_strip(n)
    );
    RETURN NULL;
EXCEPTION WHEN OTHERS THEN
    -- Never let logging block the change itself.
    RAISE WARNING 'audit_log: could not log % on %: %', TG_OP, TG_TABLE_NAME, SQLERRM;
    RETURN NULL;
END;
$$;

-- audit_enable attaches the trigger to a table. Every table that exists now
-- is audited below; a migration that creates a new table should call
-- SELECT audit_enable('new_table');
CREATE FUNCTION audit_enable(tbl REGCLASS) RETURNS void LANGUAGE plpgsql AS $$
BEGIN
    EXECUTE format('CREATE TRIGGER audit_row_change AFTER INSERT OR UPDATE OR DELETE ON %s
                    FOR EACH ROW EXECUTE FUNCTION audit_row_change()', tbl);
END;
$$;

DO $$
DECLARE t TEXT;
BEGIN
    FOR t IN
        SELECT tablename FROM pg_tables
         WHERE schemaname = 'public'
           AND tablename NOT IN ('audit_log', 'schema_migrations', 'password_reset_tokens')
           AND tablename NOT LIKE '%\_backfill\_%' AND tablename NOT LIKE '%\_fix\_%'
    LOOP
        PERFORM audit_enable(t::regclass);
    END LOOP;
END;
$$;
