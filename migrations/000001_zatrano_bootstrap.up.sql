-- ZATRANO bootstrap migration (extend in your app).
CREATE TABLE IF NOT EXISTS zatrano_schema_migrations_meta (
    id         smallint PRIMARY KEY DEFAULT 1,
    note       text NOT NULL DEFAULT 'ZATRANO placeholder — replace with your domain tables',
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT zatrano_single CHECK (id = 1)
);
