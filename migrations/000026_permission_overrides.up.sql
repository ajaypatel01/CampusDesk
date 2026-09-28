-- permission_overrides lets a super_admin/school_admin narrow (or restore) a
-- specific user's view/write access to a section of the app ("feature"),
-- layered on top of whatever their role already allows. Absence of a row for
-- (user_id, feature_key) means "use the default" -- fully permitted -- so
-- installing this table changes nothing until someone actually sets one.
CREATE TABLE IF NOT EXISTS permission_overrides (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id   UUID REFERENCES schools(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    feature_key VARCHAR(50) NOT NULL,
    can_view    BOOLEAN NOT NULL DEFAULT true,
    can_write   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, feature_key)
);

CREATE INDEX IF NOT EXISTS idx_permission_overrides_user ON permission_overrides(user_id);
CREATE INDEX IF NOT EXISTS idx_permission_overrides_school ON permission_overrides(school_id);
