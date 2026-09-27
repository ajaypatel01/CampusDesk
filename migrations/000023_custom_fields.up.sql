-- Lets super_admin define arbitrary extra fields on a section of the app
-- (e.g. Student Detail, Results) without a schema change per field.
CREATE TABLE custom_field_definitions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id      UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    -- Free text, not an enum type: a new section (entity_type) can start
    -- using this table with no migration, as long as its frontend page
    -- renders CustomFieldsSection for it.
    entity_type    TEXT NOT NULL,
    field_key      TEXT NOT NULL,
    label          TEXT NOT NULL,
    field_type     TEXT NOT NULL CHECK (field_type IN ('int', 'float', 'string', 'boolean', 'enum')),
    -- Only meaningful when field_type='enum'; a JSON array of option strings.
    enum_options   JSONB,
    sort_order     INT NOT NULL DEFAULT 0,
    created_by     UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (school_id, entity_type, field_key)
);

-- One value of one definition, for one entity instance (e.g. one student),
-- optionally scoped further (e.g. by academic year for a "result" field,
-- since the same definition means something different each year).
CREATE TABLE custom_field_values (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    definition_id  UUID NOT NULL REFERENCES custom_field_definitions(id) ON DELETE CASCADE,
    -- No FK: the referenced table depends on the definition's entity_type.
    entity_id      UUID NOT NULL,
    -- Sentinel "no scope" value rather than NULL: Postgres treats NULL as
    -- distinct from itself in a UNIQUE index, which would silently allow
    -- duplicate rows for entity types that never set a scope.
    scope_id       UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    -- Stored as text regardless of field_type; parsed/validated against the
    -- definition's field_type at the service layer.
    value_text     TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (definition_id, entity_id, scope_id)
);

CREATE INDEX idx_custom_field_values_entity ON custom_field_values (entity_id, scope_id);
