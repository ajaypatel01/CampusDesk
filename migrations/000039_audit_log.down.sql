DO $$
DECLARE t TEXT;
BEGIN
    FOR t IN SELECT event_object_table FROM information_schema.triggers
              WHERE trigger_name = 'audit_row_change' GROUP BY event_object_table
    LOOP
        EXECUTE format('DROP TRIGGER audit_row_change ON %I', t);
    END LOOP;
END;
$$;
DROP FUNCTION IF EXISTS audit_enable(REGCLASS);
DROP FUNCTION IF EXISTS audit_row_change();
DROP FUNCTION IF EXISTS audit_strip(JSONB);
DROP TABLE IF EXISTS audit_log;
